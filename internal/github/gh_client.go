package githubclient

import (
	"context"
	"fmt"

	"github.com/google/go-github/v55/github"
	"golang.org/x/oauth2"
)

type GHClient struct {
	ctx    context.Context
	client *github.Client
	repo   string // formato: "owner/repo"
}

type FileContent struct {
	SHA string
}

func NewGHClient(ctx context.Context, token, repo string) *GHClient {
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)

	return &GHClient{
		ctx:    ctx,
		client: client,
		repo:   repo,
	}
}

// CreateBranch crea una nueva rama basada en la rama base
func (g *GHClient) CreateBranch(branchName, baseBranch string) error {
	owner, repo := parseRepo(g.repo)

	// Obtener referencia base
	baseRef, _, err := g.client.Git.GetRef(g.ctx, owner, repo, "refs/heads/"+baseBranch)
	if err != nil {
		return fmt.Errorf("error obteniendo referencia base: %w", err)
	}

	// Crear referencia nueva
	ref := &github.Reference{
		Ref: github.String("refs/heads/" + branchName),
		Object: &github.GitObject{
			SHA: baseRef.Object.SHA,
		},
	}

	_, _, err = g.client.Git.CreateRef(g.ctx, owner, repo, ref)
	if err != nil {
		return fmt.Errorf("error creando rama temporal: %w", err)
	}

	return nil
}

// GetFile obtiene el SHA de un archivo en una rama específica
func (g *GHClient) GetFile(branch, path string) (*FileContent, error) {
	owner, repo := parseRepo(g.repo)

	file, _, resp, err := g.client.Repositories.GetContents(g.ctx, owner, repo, path, &github.RepositoryContentGetOptions{
		Ref: branch,
	})
	if err != nil {
		if resp != nil && resp.StatusCode == 404 {
			return nil, nil // archivo no existe
		}
		return nil, fmt.Errorf("error obteniendo archivo: %w", err)
	}

	return &FileContent{SHA: file.GetSHA()}, nil
}

// CreateFile crea un archivo nuevo en la rama especificada
func (g *GHClient) CreateFile(branch, path, content string) error {
	owner, repo := parseRepo(g.repo)

	opts := &github.RepositoryContentFileOptions{
		Message: github.String("AI Agent: crear archivo " + path),
		Content: []byte(content),
		Branch:  github.String(branch),
	}

	_, _, err := g.client.Repositories.CreateFile(g.ctx, owner, repo, path, opts)
	if err != nil {
		return fmt.Errorf("error creando archivo: %w", err)
	}
	return nil
}

// UpdateFile actualiza un archivo existente usando su SHA
func (g *GHClient) UpdateFile(branch, path, content, sha string) error {
	owner, repo := parseRepo(g.repo)

	opts := &github.RepositoryContentFileOptions{
		Message: github.String("AI Agent: actualizar archivo " + path),
		Content: []byte(content),
		SHA:     github.String(sha),
		Branch:  github.String(branch),
	}

	_, _, err := g.client.Repositories.UpdateFile(g.ctx, owner, repo, path, opts)
	if err != nil {
		return fmt.Errorf("error actualizando archivo: %w", err)
	}
	return nil
}

// CreatePullRequest crea un PR desde branch temporal a branch base
func (g *GHClient) CreatePullRequest(head, base, title, body string) (int, error) {
	owner, repo := parseRepo(g.repo)

	newPR := &github.NewPullRequest{
		Title: github.String(title),
		Head:  github.String(head),
		Base:  github.String(base),
		Body:  github.String(body),
	}

	pr, _, err := g.client.PullRequests.Create(g.ctx, owner, repo, newPR)
	if err != nil {
		return 0, fmt.Errorf("error creando Pull Request: %w", err)
	}
	return pr.GetNumber(), nil
}

// parseRepo divide "owner/repo"
func parseRepo(full string) (string, string) {
	var owner, repo string
	fmt.Sscanf(full, "%[^/]/%s", &owner, &repo)
	return owner, repo
}
