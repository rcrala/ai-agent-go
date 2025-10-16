package ai

import (
	logger "ai-agent-go/internal/logger"
	"context"
	"fmt"
	"strconv"
)

// AnthropicEvaluator implements CodeEvaluator for Anthropic Claude API
// Minimal wrapper that follows the OpenAI evaluator pattern so it can be
// mocked and extended later with a real API client.

type AnthropicEvaluator struct {
	Client *AnthropicClient
}

func NewAnthropicEvaluator(cfg AIAgentConfig) *AnthropicEvaluator {
	return &AnthropicEvaluator{
		Client: NewAnthropicClient(cfg.Key, cfg.Model, cfg.MaxTokens, cfg.Temperature, cfg.UseMockMotorAI),
	}
}

func (a *AnthropicEvaluator) Evaluate(ctx context.Context, fileName, code string) (*EvaluationResult, error) {
	lg := logger.NewLogger()
	lg.Debug("anthropic", "Evaluate", fmt.Sprintf("Evaluando archivo %s", fileName))
	res, err := evaluateAnthropicCode(ctx, fileName, code, a)
	if err != nil {
		lg.Error("anthropic", "Evaluate", fmt.Sprintf("Error evaluando %s: %v", fileName, err))
	}
	return res, err
}

// AnthropicClient is a local wrapper that can later hold the real SDK client
type AnthropicClient struct {
	APIKey        string
	Model         string
	MaxTokens     int
	Temperature   float64
	IsMockEnabled bool
}

func NewAnthropicClient(apiKey, model string, maxTokens int, temperature float64, isMock bool) *AnthropicClient {
	if apiKey == "" {
		fmt.Printf("Warning: Anthropic API key is empty\n")
	}
	return &AnthropicClient{
		APIKey:        apiKey,
		Model:         model,
		MaxTokens:     maxTokens,
		Temperature:   temperature,
		IsMockEnabled: isMock,
	}
}

func evaluateAnthropicCode(ctx context.Context, fileName, code string, a *AnthropicEvaluator) (*EvaluationResult, error) {
	if a.Client.IsMockEnabled {
		return evaluateAnthropicCodeMock(fileName, code, a)
	}
	// Real integration not implemented yet
	// When implemented, wrap HTTP errors with HTTPError for proper retry logic:
	// if resp.StatusCode >= 400 {
	//   return nil, &HTTPError{StatusCode: resp.StatusCode, Message: "..."}
	// }
	return nil, fmt.Errorf("anthropic real API integration not implemented")
}

func evaluateAnthropicCodeMock(fileName, code string, a *AnthropicEvaluator) (*EvaluationResult, error) {
	lg := logger.NewLogger()
	lg.Debug("anthropic", "evaluateAnthropicCodeMock", fmt.Sprintf("Evaluando archivo %s in Mock mode %s", fileName, strconv.FormatBool(a.Client.IsMockEnabled)))
	return &EvaluationResult{
		File:                       fileName,
		Score:                      78,
		FactoresNoCumple:           []string{"Dependencias no declaradas (mock - Anthropic)"},
		ProblemasConcurrencia:      []string{},
		RecomendacionesRefactor:    "Segregar responsabilidades y reducir acoplamiento (mock).",
		RecomendacionesComentarios: "Agregar comentarios en funciones públicas (mock).",
		Documentacion:              "Mock: sugerencias de arquitectura generadas por Anthropic.",
		EvaluacionFunciones:        []FuncionEvaluationResult{},
	}, nil
}
