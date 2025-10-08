package main

import (
	"ai-agent-go/internal/ai"
	githubclient "ai-agent-go/internal/github"
	"ai-agent-go/internal/teams"
	"context"
	"fmt"
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
			data, _ := os.ReadFile(path)
			res, err := ai.EvaluateCode(ctx, client, path, string(data))
			if err == nil {
				results = append(results, fmt.Sprintf("### %s\n%s", res.File, res.Documentacion))
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
	var prInt int = 0
	// Comenta en PR si aplica
	if prNumber != "" && prNumber != "0" {
		fmt.Println("Commenting on PR...")
		// convertir prNumber a int
		
		fmt.Sscanf(prNumber, "%d", &prInt)
		_ = githubClient.CommentOnPR(prInt, report)
	}

	// Notificación a Teams
	if webhook != "" {
		statusMsg := "✅ AI Agent ejecutado correctamente"
		if err != nil {
			statusMsg = fmt.Sprintf("⚠️ AI Agent completado con advertencias: %v", err)
		}

		var prInfo string
		if prInt > 0 {
			prInfo = fmt.Sprintf("Pull Request creado: [#%d](%s/pull/%d)", prNumber, repo, prInt)
		} else {
			prInfo = "No se generó Pull Request (posiblemente sin cambios detectables)."
		}

		message := fmt.Sprintf(`
	**🤖 AI Code Review Agent**

	%s  
	**Repositorio:** %s  
	**Rama:** %s  
	**Resultado:** %s  

	%s
	`, statusMsg, repo, branch, prInfo, "El archivo *ARQUITECTURA_COMPLIANCE.md* fue analizado y actualizado con los hallazgos de concurrencia y cumplimiento de Twelve-Factor App.")

		if err := teams.SendMessage(webhook, message); err != nil {
			fmt.Println("Error al enviar notificación a Teams:", err)
		}
	}
}
