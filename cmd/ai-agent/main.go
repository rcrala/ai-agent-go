package main

import (
	"context"
	"fmt"
	"os"

	"ai-agent-go/internal/ai"
	githubclient "ai-agent-go/internal/github"
	"ai-agent-go/internal/logger"
	"ai-agent-go/internal/teams"
)

func main() {
	ctx := context.Background()
	log := logger.NewLogger()

	// -----------------------------
	// 1️⃣ Cargar configuración (JSON + ENV)
	// -----------------------------
	cfg, err := ai.LoadConfig("config\\config_AIAgent.json")
	if err != nil {
		log.Error("main", "LoadConfig", fmt.Sprintf("Error cargando configuración: %v", err))
		os.Exit(1)
	}

	// -----------------------------
	// 2️⃣ Inicializar clientes externos
	// -----------------------------
	var openAIClient *ai.OpenAIClient
	if cfg.RunAI {
		openAIClient = ai.NewOpenAIClient(cfg.OpenAIKey, cfg.OpenAIModel, cfg.MaxTokens, cfg.Temperature)
	}

	githubClient := githubclient.NewGHClient(ctx, cfg.GitHubToken, cfg.GitHubRepo)

	// -----------------------------
	// 3️⃣ Ejecutar AI Agent (opcional)
	// -----------------------------
	var markdownAI string
	if cfg.RunAI {
		markdownAI, err = runAIAgent(ctx, openAIClient, cfg, log)
		if err != nil {
			log.Error("main", "runAIAgent", fmt.Sprintf("Error ejecutando AI Agent: %v", err))
		}
	}

	// -----------------------------
	// 4️⃣ Ejecutar SonarQube (opcional)
	// -----------------------------
	var markdownSonar string
	if cfg.RunSonar {
		markdownSonar, err = runSonarQube(cfg, log)
		if err != nil {
			log.Error("main", "runSonarQube", fmt.Sprintf("Error ejecutando SonarQube: %v", err))
		}
	}

	// -----------------------------
	// 5️⃣ Combinar reportes
	// -----------------------------
	finalReport := combineReports(markdownAI, markdownSonar)

	// -----------------------------
	// 6️⃣ Crear rama temporal, archivo y PR
	// -----------------------------
	tempBranch := fmt.Sprintf("ai-agent-update-%d", os.Getpid())
	prNumber, err := githubclient.CreateOrUpdateFileWithPR(ctx, githubClient, tempBranch, cfg.BaseBranch, "ARQUITECTURA_COMPLIANCE.md", finalReport)
	if err != nil {
		log.Error("main", "CreateOrUpdateFileWithPR", fmt.Sprintf("Error creando/actualizando archivo o PR: %v", err))
	}

	// -----------------------------
	// 7️⃣ Notificación a Teams (opcional)
	// -----------------------------
	if cfg.SendTeamsNotification && cfg.TeamsWebhookURL != "" {
		teams.SendMessage(cfg.TeamsWebhookURL, fmt.Sprintf("AI Agent report generado para branch %s. PR: %d", tempBranch, prNumber))
	}

	log.Info("main", "Completed", "Proceso finalizado")
}

// runAIAgent evalúa todos los archivos con AI Agent y devuelve Markdown
func runAIAgent(ctx context.Context, client *ai.OpenAIClient, cfg *ai.AgentConfig, log *logger.Logger) (string, error) {
	log.Info("AI", "runAIAgent", "Escaneando archivos en "+cfg.TargetDir)
	files := ai.ScanFiles(cfg.TargetDir, []string{".go", ".py"})
	if len(files) == 0 {
		log.Info("AI", "runAIAgent", "No se encontraron archivos para evaluar")
		return "", nil
	}

	results, err := ai.EvaluateFiles(ctx, client, files, cfg.BatchSize)
	if err != nil {
		return "", err
	}

	log.Info("AI", "runAIAgent", fmt.Sprintf("%d archivos evaluados", len(results)))
	return ai.GenerateMarkdown(results), nil
}

// runSonarQube ejecuta análisis y genera resumen Markdown
func runSonarQube(cfg *ai.AgentConfig, log *logger.Logger) (string, error) {
	if cfg.SonarHostURL == "" || cfg.SonarProjectKey == "" || cfg.SonarToken == "" {
		log.Info("SonarQube", "runSonarQube", "No hay configuración para SonarQube, se omite")
		return "", nil
	}

	log.Info("SonarQube", "runSonarQube", "Iniciando análisis")
	err := ai.RunSonarAnalysis(cfg.TargetDir, cfg.SonarHostURL, cfg.SonarProjectKey, cfg.SonarToken)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("## SonarQube Summary\n- Proyecto: %s\n- Dashboard: %s/dashboard?id=%s\n",
		cfg.SonarProjectKey, cfg.SonarHostURL, cfg.SonarProjectKey), nil
}

// combineReports junta reportes AI + Sonar en un único Markdown
func combineReports(aiReport, sonarReport string) string {
	report := "# Architecture Compliance Report\n\n"
	if aiReport != "" {
		report += aiReport + "\n"
	}
	if sonarReport != "" {
		report += sonarReport + "\n"
	}
	return report
}
