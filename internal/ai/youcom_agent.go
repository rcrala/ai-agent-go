package ai

import (
	logger "ai-agent-go/internal/logger"
	"context"
	"fmt"
	"strconv"
)

// YouComEvaluator implements CodeEvaluator for you.com LLM API
// Uses the standardized evaluator pattern so behavior is consistent across agents.

type YouComEvaluator struct {
	Client *YouComClient
}

func NewYouComEvaluator(cfg AIAgentConfig) *YouComEvaluator {
	return &YouComEvaluator{Client: NewYouComClient(cfg.Key, cfg.Model, cfg.MaxTokens, cfg.Temperature, cfg.UseMockMotorAI)}
}

func (y *YouComEvaluator) Evaluate(ctx context.Context, fileName, code string) (*EvaluationResult, error) {
	lg := logger.NewLogger()
	lg.Debug("youcom", "Evaluate", fmt.Sprintf("Evaluando archivo %s", fileName))
	res, err := evaluateYouComCode(ctx, fileName, code, y)
	if err != nil {
		lg.Error("youcom", "Evaluate", fmt.Sprintf("Error evaluando %s: %v", fileName, err))
	}
	return res, err
}

type YouComClient struct {
	APIKey        string
	Model         string
	MaxTokens     int
	Temperature   float64
	IsMockEnabled bool
}

func NewYouComClient(apiKey, model string, maxTokens int, temperature float64, isMock bool) *YouComClient {
	if apiKey == "" {
		fmt.Printf("Warning: You.com API key is empty\n")
	}
	return &YouComClient{APIKey: apiKey, Model: model, MaxTokens: maxTokens, Temperature: temperature, IsMockEnabled: isMock}
}

func evaluateYouComCode(ctx context.Context, fileName, code string, y *YouComEvaluator) (*EvaluationResult, error) {
	if y.Client.IsMockEnabled {
		return evaluateYouComCodeMock(fileName, code, y)
	}
	return nil, fmt.Errorf("you.com real API integration not implemented")
}

func evaluateYouComCodeMock(fileName, code string, y *YouComEvaluator) (*EvaluationResult, error) {
	lg := logger.NewLogger()
	lg.Debug("youcom", "evaluateYouComCodeMock", fmt.Sprintf("Evaluando archivo %s in Mock mode %s", fileName, strconv.FormatBool(y.Client.IsMockEnabled)))
	return &EvaluationResult{
		File:                       fileName,
		Score:                      75,
		FactoresNoCumple:           []string{"Dependencias no declaradas (mock - You.com)"},
		ProblemasConcurrencia:      []string{},
		RecomendacionesRefactor:    "Separar responsabilidades y reducir acoplamientos (mock).",
		RecomendacionesComentarios: "Mejorar comentarios en funciones públicas (mock).",
		Documentacion:              "Mock: recomendaciones generadas por You.com.",
		EvaluacionFunciones:        []FuncionEvaluationResult{},
	}, nil
}
