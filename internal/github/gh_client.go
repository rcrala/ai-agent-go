package githubclient

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/go-github/v55/github"
	"golang.org/x/oauth2"
)

type GHClient struct {
	Client *github.Client
	Ctx    context.Context
	Owner  string
	Repo   string
}

func NewGHClient(ctx context.Context, token, repo string) *GHClient {
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)

	parts := strings.Split(repo, "/")
	return &GHClient{
		Client: client,
		Ctx:    ctx,
		Owner:  parts[0],
		Repo:   parts[1],
	}
}

// CreateOrUpdateFile crea o actualiza ARCHITECTURE_COMPLIANCE.md en la rama actual
func (g *GHClient) CreateOrUpdateFile(branch, content string) error {
	path := "ARCHITECTURE_COMPLIANCE.md"
	file, _, _, err := g.Client.Repositories.GetContents(g.Ctx, g.Owner, g.Repo, path, &github.RepositoryContentGetOptions{Ref: branch})

	options := &github.RepositoryContentFileOptions{
		Message: github.String("AI: update architecture compliance report"),
		Content: []byte(content),
		Branch:  github.String(branch),
	}

	if err == nil && file != nil && file.GetSHA() != "" {
		options.SHA = github.String(file.GetSHA())
		_, _, err = g.Client.Repositories.UpdateFile(g.Ctx, g.Owner, g.Repo, path, options)
	} else {
		_, _, err = g.Client.Repositories.CreateFile(g.Ctx, g.Owner, g.Repo, path, options)
	}
	return err
}

// CommentOnPR agrega o actualiza un comentario en el PR con el resultado
func (g *GHClient) CommentOnPR(prNumber int, body string) error {
	comments, _, err := g.Client.Issues.ListComments(g.Ctx, g.Owner, g.Repo, prNumber, nil)
	if err != nil {
		return err
	}

	var existing *github.IssueComment
	for _, c := range comments {
		if strings.Contains(c.GetBody(), "AI Compliance Report") {
			existing = c
			break
		}
	}

	commentBody := fmt.Sprintf("🤖 **AI Compliance Report**\n\n%s", body)

	if existing != nil {
		_, _, err = g.Client.Issues.EditComment(g.Ctx, g.Owner, g.Repo, existing.GetID(), &github.IssueComment{Body: &commentBody})
	} else {
		_, _, err = g.Client.Issues.CreateComment(g.Ctx, g.Owner, g.Repo, prNumber, &github.IssueComment{Body: &commentBody})
	}

	return err
}
