package ai

import (
	logger "ai-agent-go/internal/logger"
	"context"
	"fmt"
	"strconv"
)

// CopilotEvaluator implements CodeEvaluator for Copilot
// Keeps the placeholder behavior but follows the same structure
// as other agents so it's easier to extend and test.

type CopilotEvaluator struct {
	Client *CopilotClient
}

func NewCopilotEvaluator(cfg AIAgentConfig) *CopilotEvaluator {
	return &CopilotEvaluator{Client: NewCopilotClient(cfg.Key, cfg.Model, cfg.MaxTokens, cfg.Temperature, cfg.UseMockMotorAI)}
}

func (c *CopilotEvaluator) Evaluate(ctx context.Context, fileName, code string) (*EvaluationResult, error) {
	lg := logger.NewLogger()
	lg.Debug("copilot", "Evaluate", fmt.Sprintf("Evaluando archivo %s", fileName))
	res, err := evaluateCopilotCode(ctx, fileName, code, c)
	if err != nil {
		lg.Error("copilot", "Evaluate", fmt.Sprintf("Error evaluando %s: %v", fileName, err))
	}
	return res, err
}

type CopilotClient struct {
	APIKey        string
	Model         string
	MaxTokens     int
	Temperature   float64
	IsMockEnabled bool
}

func NewCopilotClient(apiKey, model string, maxTokens int, temperature float64, isMock bool) *CopilotClient {
	if apiKey == "" {
		fmt.Printf("Warning: Copilot API key is empty\n")
	}
	return &CopilotClient{APIKey: apiKey, Model: model, MaxTokens: maxTokens, Temperature: temperature, IsMockEnabled: isMock}
}

func evaluateCopilotCode(ctx context.Context, fileName, code string, c *CopilotEvaluator) (*EvaluationResult, error) {
	if c.Client.IsMockEnabled {
		return evaluateCopilotCodeMock(fileName, code, c)
	}
	// No real Copilot API integration implemented yet
	// When implemented, wrap HTTP errors with HTTPError for proper retry logic:
	// if resp.StatusCode >= 400 {
	//   return nil, &HTTPError{StatusCode: resp.StatusCode, Message: "..."}
	// }
	return nil, fmt.Errorf("copilot real API integration not implemented")
}

func evaluateCopilotCodeMock(fileName, code string, c *CopilotEvaluator) (*EvaluationResult, error) {
	lg := logger.NewLogger()
	lg.Debug("copilot", "evaluateCopilotCodeMock", fmt.Sprintf("Evaluando archivo %s in Mock mode %s", fileName, strconv.FormatBool(c.Client.IsMockEnabled)))
	prompt := GetEvaluationPrompt(code)
	lg.Debug("copilot", "prompt", prompt)
	return &EvaluationResult{
		File:                       fileName,
		Score:                      80,
		FactoresNoCumple:           []string{"Dependencias no declaradas (mock - Copilot)"},
		ProblemasConcurrencia:      []string{},
		RecomendacionesRefactor:    "Aplicar las sugerencias detectadas por Copilot (mock).",
		RecomendacionesComentarios: "Mejorar comentarios en funciones públicas (mock).",
		Documentacion:              "Mock: documentación sugerida por Copilot.",
		EvaluacionFunciones:        []FuncionEvaluationResult{},
	}, nil
}
