package main

import (
	"ai-agent-go/internal/ai"
	githubclient "ai-agent-go/internal/github"
	"ai-agent-go/internal/teams"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	openai "github.com/sashabaranov/go-openai"
)

func main() {
	ctx := context.Background()
	openaiKey := os.Getenv("OPENAI_API_KEY")
	ghToken := os.Getenv("GITHUB_TOKEN")
	repo := os.Getenv("REPO_NAME")
	branch := os.Getenv("BRANCH_NAME")
	prNumber := os.Getenv("PR_NUMBER")
	webhook := os.Getenv("TEAMS_WEBHOOK_URL")

	if openaiKey == "" || ghToken == "" {
		fmt.Println("Missing OPENAI_API_KEY or GITHUB_TOKEN.")
		os.Exit(1)
	}

	client := openai.NewClient(openaiKey)
	githubClient := githubclient.NewGHClient(ctx, ghToken, repo)

	results := []string{}
	_ = filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() && (strings.HasSuffix(path, ".go") || strings.HasSuffix(path, ".py")) {
			data, _ := ioutil.ReadFile(path)
			res, err := ai.EvaluateCode(ctx, client, path, string(data))
			if err == nil {
				results = append(results, fmt.Sprintf("### %s\n%s", res.File, res.Recomendaciones))
			}
		}
		return nil
	})

	report := strings.Join(results, "\n\n---\n\n")
	finalMD := fmt.Sprintf("# Architecture Compliance Report\n\n%s", report)

	// Actualiza archivo en rama
	err := githubClient.CreateOrUpdateFile(branch, finalMD)
	if err != nil {
		fmt.Println("Error updating file:", err)
	}

	// Comenta en PR si aplica
	if prNumber != "" && prNumber != "0" {
		fmt.Println("Commenting on PR...")
		// convertir prNumber a int
		var prInt int
		fmt.Sscanf(prNumber, "%d", &prInt)
		_ = githubClient.CommentOnPR(prInt, report)
	}

	// Notificación a Teams
	if webhook != "" {
		_ = teams.SendMessage(webhook, fmt.Sprintf("AI Agent report generated for branch %s", branch))
	}
}
