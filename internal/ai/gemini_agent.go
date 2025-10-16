package ai

import (
	logger "ai-agent-go/internal/logger"
	"context"
	"fmt"
	"os"
)

// GeminiEvaluator implements CodeEvaluator for Google Gemini (Vertex AI) API
// Wrapper follows the consistent pattern used for other agents.

type GeminiEvaluator struct {
	Client *GeminiClient
}

func NewGeminiEvaluator(cfg AIAgentConfig) *GeminiEvaluator {
	return &GeminiEvaluator{Client: NewGeminiClient(cfg.Key, cfg.Model, cfg.MaxTokens, cfg.Temperature, cfg.UseMockMotorAI)}
}

func (g *GeminiEvaluator) Evaluate(ctx context.Context, fileName, code string) (*EvaluationResult, error) {
	lg := logger.NewLogger()
	lg.Debug("gemini", "Evaluate", fmt.Sprintf("Evaluando archivo %s", fileName))
	res, err := evaluateGeminiCode(ctx, fileName, code, g)
	if err != nil {
		lg.Error("gemini", "Evaluate", fmt.Sprintf("Error evaluando %s: %v", fileName, err))
	}
	return res, err
}

type GeminiClient struct {
	APIKey        string
	Model         string
	MaxTokens     int
	Temperature   float64
	IsMockEnabled bool
}

func NewGeminiClient(apiKey, model string, maxTokens int, temperature float64, isMock bool) *GeminiClient {
	if apiKey == "" {
		fmt.Printf("Warning: Gemini API key is empty\n")
	}
	return &GeminiClient{APIKey: apiKey, Model: model, MaxTokens: maxTokens, Temperature: temperature, IsMockEnabled: isMock}
}

func evaluateGeminiCode(ctx context.Context, fileName, code string, g *GeminiEvaluator) (*EvaluationResult, error) {
	if g.Client.IsMockEnabled {
		return evaluateGeminiCodeMock(fileName, code, g)
	}
	return nil, fmt.Errorf("gemini real API integration not implemented")
}

func evaluateGeminiCodeMock(fileName, code string, g *GeminiEvaluator) (*EvaluationResult, error) {
	lg := logger.NewLogger()
	lg.Debug("gemini", "evaluateGeminiCodeMock", fmt.Sprintf("Evaluando archivo %s in Mock mode %s", fileName, os.Getenv("USE_MOCK_MOTOR_AI")))
	return &EvaluationResult{
		File:                       fileName,
		Score:                      77,
		FactoresNoCumple:           []string{"Dependencias no declaradas (mock - Gemini)"},
		ProblemasConcurrencia:      []string{},
		RecomendacionesRefactor:    "Reestructurar módulos para claridad (mock).",
		RecomendacionesComentarios: "Agregar ejemplos y comentarios en puntos críticos (mock).",
		Documentacion:              "Mock: documentación generada por Gemini.",
		EvaluacionFunciones:        []FuncionEvaluationResult{},
	}, nil
}
