package ai

import (
	logger "ai-agent-go/internal/logger"
	"context"
	"fmt"
	"os"
)

// CohereEvaluator implements CodeEvaluator for Cohere API
// Follows the same wrapper pattern as OpenAI for consistency and testing.

type CohereEvaluator struct {
	Client *CohereClient
}

func NewCohereEvaluator(cfg AIAgentConfig) *CohereEvaluator {
	return &CohereEvaluator{
		Client: NewCohereClient(cfg.Key, cfg.Model, cfg.MaxTokens, cfg.Temperature, cfg.UseMockMotorAI),
	}
}

func (c *CohereEvaluator) Evaluate(ctx context.Context, fileName, code string) (*EvaluationResult, error) {
	lg := logger.NewLogger()
	lg.Debug("cohere", "Evaluate", fmt.Sprintf("Evaluando archivo %s", fileName))
	res, err := evaluateCohereCode(ctx, fileName, code, c)
	if err != nil {
		lg.Error("cohere", "Evaluate", fmt.Sprintf("Error evaluando %s: %v", fileName, err))
	}
	return res, err
}

type CohereClient struct {
	APIKey        string
	Model         string
	MaxTokens     int
	Temperature   float64
	IsMockEnabled bool
}

func NewCohereClient(apiKey, model string, maxTokens int, temperature float64, isMock bool) *CohereClient {
	if apiKey == "" {
		fmt.Printf("Warning: Cohere API key is empty\n")
	}
	return &CohereClient{APIKey: apiKey, Model: model, MaxTokens: maxTokens, Temperature: temperature, IsMockEnabled: isMock}
}

func evaluateCohereCode(ctx context.Context, fileName, code string, c *CohereEvaluator) (*EvaluationResult, error) {
	if c.Client.IsMockEnabled {
		return evaluateCohereCodeMock(fileName, code, c)
	}
	return nil, fmt.Errorf("cohere real API integration not implemented")
}

func evaluateCohereCodeMock(fileName, code string, c *CohereEvaluator) (*EvaluationResult, error) {
	lg := logger.NewLogger()
	lg.Debug("cohere", "evaluateCohereCodeMock", fmt.Sprintf("Evaluando archivo %s in Mock mode %s", fileName, os.Getenv("USE_MOCK_MOTOR_AI")))
	return &EvaluationResult{
		File:                       fileName,
		Score:                      76,
		FactoresNoCumple:           []string{"Dependencias no declaradas (mock - Cohere)"},
		ProblemasConcurrencia:      []string{},
		RecomendacionesRefactor:    "Reducir complejidad y separar responsabilidades (mock).",
		RecomendacionesComentarios: "Agregar comentarios y ejemplos de uso (mock).",
		Documentacion:              "Mock: recomendaciones generadas por Cohere.",
		EvaluacionFunciones:        []FuncionEvaluationResult{},
	}, nil
}
