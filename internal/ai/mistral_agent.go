package ai

import (
	logger "ai-agent-go/internal/logger"
	"context"
	"fmt"
	"strconv"
)

// MistralEvaluator implements CodeEvaluator for Mistral API
// Follows the same wrapper pattern so real integration can be added later.

type MistralEvaluator struct {
	Client *MistralClient
}

func NewMistralEvaluator(cfg AIAgentConfig) *MistralEvaluator {
	return &MistralEvaluator{Client: NewMistralClient(cfg.Key, cfg.Model, cfg.MaxTokens, cfg.Temperature, cfg.UseMockMotorAI)}
}

func (m *MistralEvaluator) Evaluate(ctx context.Context, fileName, code string) (*EvaluationResult, error) {
	lg := logger.NewLogger()
	lg.Debug("mistral", "Evaluate", fmt.Sprintf("Evaluando archivo %s", fileName))
	res, err := evaluateMistralCode(ctx, fileName, code, m)
	if err != nil {
		lg.Error("mistral", "Evaluate", fmt.Sprintf("Error evaluando %s: %v", fileName, err))
	}
	return res, err
}

type MistralClient struct {
	APIKey        string
	Model         string
	MaxTokens     int
	Temperature   float64
	IsMockEnabled bool
}

func NewMistralClient(apiKey, model string, maxTokens int, temperature float64, isMock bool) *MistralClient {
	if apiKey == "" {
		fmt.Printf("Warning: Mistral API key is empty\n")
	}
	return &MistralClient{APIKey: apiKey, Model: model, MaxTokens: maxTokens, Temperature: temperature, IsMockEnabled: isMock}
}

func evaluateMistralCode(ctx context.Context, fileName, code string, m *MistralEvaluator) (*EvaluationResult, error) {
	if m.Client.IsMockEnabled {
		return evaluateMistralCodeMock(fileName, code, m)
	}
	// Real integration not implemented yet
	// When implemented, wrap HTTP errors with HTTPError for proper retry logic:
	// if resp.StatusCode >= 400 {
	//   return nil, &HTTPError{StatusCode: resp.StatusCode, Message: "..."}
	// }
	return nil, fmt.Errorf("mistral real API integration not implemented")
}

func evaluateMistralCodeMock(fileName, code string, m *MistralEvaluator) (*EvaluationResult, error) {
	lg := logger.NewLogger()
	lg.Debug("mistral", "evaluateMistralCodeMock", fmt.Sprintf("Evaluando archivo %s in Mock mode %s", fileName, strconv.FormatBool(m.Client.IsMockEnabled)))
	return &EvaluationResult{
		File:                       fileName,
		Score:                      74,
		FactoresNoCumple:           []string{"Dependencias no declaradas (mock - Mistral)"},
		ProblemasConcurrencia:      []string{},
		RecomendacionesRefactor:    "Refactorizar para responsabilidades claras (mock).",
		RecomendacionesComentarios: "Documentar puntos de sincronización y errores (mock).",
		Documentacion:              "Mock: recomendaciones generadas por Mistral.",
		EvaluacionFunciones:        []FuncionEvaluationResult{},
	}, nil
}
