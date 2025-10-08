package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"ai-agent-go/internal/ai"
	githubclient "ai-agent-go/internal/github"
	"ai-agent-go/internal/teams"

	openai "github.com/sashabaranov/go-openai"
)

func randomSuffix(n int) string {
	b := make([]byte, n)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func main() {
	start := time.Now()
	fmt.Println("🚀 Iniciando ejecución del AI Agent...")

	// Variables de entorno
	repoFull := os.Getenv("GITHUB_REPOSITORY")
	token := os.Getenv("GITHUB_TOKEN")
	webhook := os.Getenv("TEAMS_WEBHOOK_URL")
	branch := os.Getenv("GITHUB_REF_NAME")
	openaiKey := os.Getenv("OPENAI_API_KEY")
	targetDir := os.Getenv("TARGET_DIR")
	if targetDir == "" {
		targetDir = "./"
	}

	if repoFull == "" || token == "" || openaiKey == "" {
		fmt.Println("❌ Faltan variables de entorno GITHUB_REPOSITORY, GITHUB_TOKEN o OPENAI_API_KEY")
		os.Exit(1)
	}

	ctx := context.Background()
	githubClient := githubclient.NewGHClient(ctx, token, repoFull)
	openaiClient := openai.NewClient(openaiKey)

	// Buscar archivos Go
	files, err := filepath.Glob(filepath.Join(targetDir, "*.go"))
	if err != nil || len(files) == 0 {
		fmt.Println("⚠️  No se encontraron archivos Go para evaluar.")
		os.Exit(0)
	}

	var allResults []*ai.EvaluationResult
	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			fmt.Printf("⚠️  Error leyendo archivo %s: %v\n", file, err)
			continue
		}

		result, err := ai.EvaluateCode(ctx, openaiClient, file, string(content))
		if err != nil {
			fmt.Printf("⚠️  Error evaluando código en %s: %v\n", file, err)
			continue
		}

		allResults = append(allResults, result)
		fmt.Printf("📄 Archivo %s evaluado con puntaje %d\n", result.File, result.Score)
	}

	// Generar Markdown
	finalMD := "# 📘 Architecture Compliance Report\n\n"
	finalMD += fmt.Sprintf("_Generado automáticamente: %s_\n\n", time.Now().Format(time.RFC822))

	for _, r := range allResults {
		finalMD += fmt.Sprintf("## %s\n\n", r.File)
		finalMD += fmt.Sprintf("**Puntaje:** %d/100\n\n", r.Score)

		if len(r.FactoresNoCumple) > 0 {
			finalMD += "**Factores no cumplidos:**\n"
			for _, f := range r.FactoresNoCumple {
				finalMD += fmt.Sprintf("- %s\n", f)
			}
			finalMD += "\n"
		}

		if len(r.ProblemasConcurrencia) > 0 {
			finalMD += "**Problemas de concurrencia:**\n"
			for _, p := range r.ProblemasConcurrencia {
				finalMD += fmt.Sprintf("- %s\n", p)
			}
			finalMD += "\n"
		}

		finalMD += fmt.Sprintf("**Recomendaciones de refactor:** %s\n\n", r.RecomendacionesRefactor)
		finalMD += fmt.Sprintf("**Recomendaciones de comentarios:** %s\n\n", r.RecomendacionesComentarios)

		if len(r.EvaluacionFunciones) > 0 {
			finalMD += "### Evaluación por función\n"
			for _, f := range r.EvaluacionFunciones {
				finalMD += fmt.Sprintf("- **%s**: Claridad=%s, Complejidad=%s, Riesgo Concurrencia=%s, Sugerencias=%s\n",
					f.Funcion, f.Claridad, f.Complejidad, f.RiesgoConcurrencia, f.Sugerencias)
			}
			finalMD += "\n"
		}

		finalMD += "### Documentación Técnica\n" + r.Documentacion + "\n\n---\n\n"
	}

	// Crear rama temporal única
	tempBranch := fmt.Sprintf("ai-architecture-%d-%s", time.Now().UnixMilli(), randomSuffix(3))
	fmt.Println("🪄 Creando rama temporal:", tempBranch)
	err = githubClient.CreateBranch(tempBranch, branch)
	if err != nil {
		fmt.Printf("❌ Error creando rama temporal: %v\n", err)
		os.Exit(1)
	}

	// Crear o actualizar archivo
	fmt.Println("📄 Actualizando ARQUITECTURA_COMPLIANCE.md...")
	existingFile, _ := githubClient.GetFile(tempBranch, "ARQUITECTURA_COMPLIANCE.md")
	if existingFile != nil {
		err = githubClient.UpdateFile(tempBranch, "ARQUITECTURA_COMPLIANCE.md", finalMD, existingFile.SHA)
	} else {
		err = githubClient.CreateFile(tempBranch, "ARQUITECTURA_COMPLIANCE.md", finalMD)
	}
	if err != nil {
		fmt.Printf("❌ Error creando o actualizando archivo: %v\n", err)
		os.Exit(1)
	}

	// Crear Pull Request
	fmt.Println("🔄 Creando Pull Request...")
	prNumber, err := githubClient.CreatePullRequest(
		tempBranch,
		branch,
		"AI Agent: Architecture Compliance Report",
		"Este PR fue generado automáticamente con el resultado del análisis de cumplimiento y calidad de código en Go.\nIncluye recomendaciones de refactorización y documentación técnica basada en los principios de The Twelve-Factor App y buenas prácticas de concurrencia.",
	)
	if err != nil {
		fmt.Printf("❌ Error creando Pull Request: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("✅ Pull Request creado exitosamente: #%d\n", prNumber)

	// Notificación a Teams
	if webhook != "" {
		message := fmt.Sprintf("🤖 **AI Agent completado**\nRepositorio: `%s`\nRama base: `%s`\nPull Request: #%d\nDuración total: %s",
			repoFull, branch, prNumber, time.Since(start).Truncate(time.Second))
		if err := teams.SendMessage(webhook, message); err != nil {
			fmt.Printf("⚠️  Error enviando notificación a Teams: %v\n", err)
		} else {
			fmt.Println("📩 Notificación enviada a Teams.")
		}
	}

	fmt.Printf("🏁 Ejecución finalizada en %s.\n", time.Since(start).Truncate(time.Second))
}
