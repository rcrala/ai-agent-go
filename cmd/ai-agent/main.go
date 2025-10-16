package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"ai-agent-go/internal/ai"
	githubclient "ai-agent-go/internal/github"
	"ai-agent-go/internal/logger"
	teams "ai-agent-go/internal/teams"
)

// runSonarIfEnabled runs SonarQube analysis if enabled in config
func runSonarIfEnabled(cfg *ai.AgentConfig, log *logger.Logger) string {
	if !cfg.RunSonar {
		return ""
	}
	markdownSonar, err := runSonarQube(cfg, log)
	if err != nil {
		log.Error("main", "runSonarQube", fmt.Sprintf("Error ejecutando SonarQube: %v", err))
		return ""
	}
	return markdownSonar
}

// createOrUpdatePR creates or updates a PR with the report
func createOrUpdatePR(ctx context.Context, githubClient *githubclient.GHClient, tempBranch string, cfg *ai.AgentConfig, finalReport string, log *logger.Logger) int {
	prNumber, err := githubclient.CreateOrUpdateFileWithPR(ctx, githubClient, tempBranch, cfg.BaseBranch, "ARQUITECTURA_COMPLIANCE.md", finalReport)
	if err != nil {
		log.Error("main", "CreateOrUpdateFileWithPR", fmt.Sprintf("Error creando/actualizando archivo o PR: %v", err))
		return 0
	}
	log.Info("main", "createOrUpdatePR", fmt.Sprintf("PR creado/actualizado: %d", prNumber))
	return prNumber
}

// sendTeamsNotificationIfNeeded sends a Teams notification if enabled
func sendTeamsNotificationIfNeeded(cfg *ai.AgentConfig, tempBranch string, prNumber int) {
	if cfg.SendTeamsNotification && cfg.TeamsWebhookURL != "" {
		if err := teams.SendMessage(cfg.TeamsWebhookURL, fmt.Sprintf("AI Agent report generado para branch %s. PR: %d", tempBranch, prNumber)); err != nil {
			log := logger.NewLogger()
			log.Error("main", "sendTeamsNotificationIfNeeded", fmt.Sprintf("error enviando Teams: %v", err))
		} else {
			log := logger.NewLogger()
			log.Info("main", "sendTeamsNotificationIfNeeded", "Notificación enviada a Teams")
		}
	}
}

func main() {
	ctx := context.Background()
	log := logger.NewLogger()

	cfg := loadConfigOrExit(log)
	githubClient := githubclient.NewGHClient(ctx, cfg.GitHubToken, cfg.GitHubRepo)

	// Record execution start
	startTime := time.Now().UTC()

	markdownAI := runAIAgents(ctx, log, cfg)
	markdownSonar := runSonarIfEnabled(cfg, log)

	finalReport := combineReports(markdownAI, markdownSonar)

	// Record execution end
	endTime := time.Now().UTC()

	// Build engine/agent metadata
	engineInfo := ""
	engineInfo += "## Execution Metadata\n\n"
	engineInfo += fmt.Sprintf("- Start: %s\n", startTime.Format(time.RFC3339))
	engineInfo += fmt.Sprintf("- End:   %s\n", endTime.Format(time.RFC3339))
	// Try to get current commit hash
	commitHash := "unknown"
	if out, err := exec.Command("git", "rev-parse", "--short", "HEAD").Output(); err == nil {
		commitHash = strings.TrimSpace(string(out))
	}
	engineInfo += fmt.Sprintf("- Commit: %s\n", commitHash)
	engineInfo += fmt.Sprintf("- Global UseMockMotorAI: %v\n", cfg.UseMockMotorAI)
	engineInfo += "- Agents:\n"
	for _, a := range cfg.Agents {
		if !a.Enabled {
			continue
		}
		// determine effective mock for agent (agent-level override or global)
		useMock := a.UseMockMotorAI || cfg.UseMockMotorAI
		engineInfo += fmt.Sprintf("  - %s (model: %s) - UseMock: %v\n", a.Type, a.Model, useMock)
	}
	engineInfo += "\n---\n\n"

	// Prepend metadata to report
	finalReport = engineInfo + finalReport

	// Persist a local copy of the generated report for review before creating PR
	if err := os.WriteFile("ARQUITECTURE_COMPLIANCE.md", []byte(finalReport), 0644); err != nil {
		log.Error("main", "WriteReportLocal", fmt.Sprintf("Error escribiendo ARCHITECTURE_COMPLIANCE.md localmente: %v", err))
	} else {
		log.Info("main", "WriteReportLocal", "Archivo ARCHITECTURE_COMPLIANCE.md escrito localmente")
	}
	tempBranch := fmt.Sprintf("ai-agent-update-%d", os.Getpid())
	prNumber := createOrUpdatePR(ctx, githubClient, tempBranch, cfg, finalReport, log)

	sendTeamsNotificationIfNeeded(cfg, tempBranch, prNumber)
	log.Info("main", "Completed", "Proceso finalizado")
}

func loadConfigOrExit(log *logger.Logger) *ai.AgentConfig {
	cfg, err := ai.LoadConfig("config", "config_AIAgent.json")
	if err != nil {
		log.Error("main", "LoadConfig", fmt.Sprintf("Error cargando configuración: %v", err))
		os.Exit(1)
	}
	return cfg
}

func runAIAgents(ctx context.Context, log *logger.Logger, cfg *ai.AgentConfig) string {
	var markdownAI string
	if len(cfg.Agents) == 0 {
		return markdownAI
	}
	for _, agentCfg := range cfg.Agents {
		if !agentCfg.Enabled {
			continue
		}
		evaluator := ai.NewCodeEvaluator(agentCfg)
		if evaluator == nil {
			log.Error("AI", "NewCodeEvaluator", fmt.Sprintf("Tipo de agente no soportado: %s", agentCfg.Type))
			continue
		}
		log.Info("AI", "runAIAgents", fmt.Sprintf("Evaluando archivos con agente: %s", agentCfg.Type))
		files := ai.ScanFiles(cfg.TargetDir, []string{".go", ".py"})
		if len(files) == 0 {
			log.Info("AI", "runAIAgents", "No se encontraron archivos para evaluar")
			continue
		}
		results, err := ai.EvaluateFilesGeneric(ctx, evaluator, files)
		if err != nil {
			log.Error("AI", "EvaluateFilesGeneric", fmt.Sprintf("Error evaluando archivos: %v", err))
			continue
		}
		log.Info("AI", "runAIAgents", fmt.Sprintf("%d archivos evaluados por %s", len(results), agentCfg.Type))
		markdownAI += ai.GenerateMarkdown(results)
	}
	return markdownAI
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
