package ai

import (
	logger "ai-agent-go/internal/logger"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	openai "github.com/sashabaranov/go-openai"
)

// OpenAIEvaluator implements CodeEvaluator for OpenAI
// Wraps the OpenAIClient and config

type OpenAIEvaluator struct {
	Client *OpenAIClient
}

func NewOpenAIEvaluator(cfg AIAgentConfig) *OpenAIEvaluator {
	return &OpenAIEvaluator{
		Client: NewOpenAIClient(cfg.Key, cfg.Model, cfg.MaxTokens, cfg.Temperature, cfg.UseMockMotorAI),
	}
}

func (o *OpenAIEvaluator) Evaluate(ctx context.Context, fileName, code string) (*EvaluationResult, error) {
	lg := logger.NewLogger()
	lg.Debug("openai", "Evaluate", fmt.Sprintf("Evaluando archivo %s", fileName))
	res, err := evaluateCode(ctx, fileName, code, o)
	if err != nil {
		lg.Error("openai", "Evaluate", fmt.Sprintf("Error evaluando %s: %v", fileName, err))
	}
	return res, err
}

// OpenAI-specific client wrapper
type OpenAIClient struct {
	Client        *openai.Client
	Model         string
	MaxTokens     int
	Temperature   float64
	IsMockEnabled bool
}

func NewOpenAIClient(apiKey, model string, maxTokens int, temperature float64, IsMockEnabled bool) *OpenAIClient {
	if apiKey == "" {
		fmt.Printf("Warning: OpenAI API key is empty")
	}
	return &OpenAIClient{
		Client:        openai.NewClient(apiKey),
		Model:         model,
		MaxTokens:     maxTokens,
		Temperature:   temperature,
		IsMockEnabled: IsMockEnabled,
	}
}

// evaluateCode
func evaluateCode(ctx context.Context, fileName, code string, o *OpenAIEvaluator) (*EvaluationResult, error) {
	// Validate if mock is enabled
	if o.Client.IsMockEnabled {
		return evaluateCodeMock(fileName, code, o)
	}
	// Call the real OpenAI API
	return evaluateCodeReal(ctx, o.Client.Client, fileName, code, o.Client.Model, o.Client.MaxTokens, float32(o.Client.Temperature))
}

// evaluateCode use a Mock Response and parses the result
func evaluateCodeMock(fileName, code string, o *OpenAIEvaluator) (*EvaluationResult, error) {
	lg := logger.NewLogger()
	// prompt := GetEvaluationPrompt(code)
	lg.Debug("openai", "Evaluate", fmt.Sprintf("Evaluando archivo %s with %s model in Mock mode %s", fileName, o.Client.Model, os.Getenv("USE_MOCK_MOTOR_AI")))

	lg.Info("openai", "evaluateCode", "OPENAI_MOCK=true -> returning mock EvaluationResult")
	mock := &EvaluationResult{
		File:                       fileName,
		Score:                      75,
		FactoresNoCumple:           []string{"Dependencias no declaradas"},
		ProblemasConcurrencia:      []string{"Uso de goroutines sin sincronización"},
		RecomendacionesRefactor:    "Extraer funciones y simplificar responsabilidades.",
		RecomendacionesComentarios: "Agregar comentarios en funciones públicas explicando el propósito y los efectos colaterales.",
		Documentacion:              "Mock: arquitectura recomendada: modularizar paquetes y usar context.",
		EvaluacionFunciones: []FuncionEvaluationResult{{
			Funcion:            "MockFunc",
			Claridad:           "Media",
			Complejidad:        "Media",
			RiesgoConcurrencia: "Medio",
			Sugerencias:        "Usar canales o mutex donde corresponda.",
		}},
	}
	return mock, nil

}

// evaluateCode calls the OpenAI API and parses the result
func evaluateCodeReal(ctx context.Context, client *openai.Client, fileName, code, model string, maxTokens int, temperature float32) (*EvaluationResult, error) {
	lg := logger.NewLogger()
	prompt := GetEvaluationPrompt(code)

	lg.Debug("openai", "Evaluate", fmt.Sprintf("Evaluando archivo %s with %s model in Mock mode %s", fileName, model, os.Getenv("USE_MOCK_MOTOR_AI")))

	resp, err := client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model:       model,
		Messages:    []openai.ChatCompletionMessage{{Role: openai.ChatMessageRoleUser, Content: prompt}},
		Temperature: temperature,
		MaxTokens:   maxTokens,
	})
	if err != nil {
		lg.Error("openai", "evaluateCode", fmt.Sprintf("API error: %v", err))
		return nil, err
	}

	content := strings.TrimSpace(resp.Choices[0].Message.Content)

	var result EvaluationResult
	if err := json.Unmarshal([]byte(content), &result); err != nil {
		lg.Error("openai", "evaluateCode", fmt.Sprintf("error parseando JSON: %v", err))
		return nil, fmt.Errorf("error parseando JSON de AI: %w", err)
	}

	return &result, nil
}
