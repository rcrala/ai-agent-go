package ai

import (
	logger "ai-agent-go/internal/logger"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"

	openai "github.com/sashabaranov/go-openai"
)

// OpenAIEvaluator implements CodeEvaluator for OpenAI
// Wraps the OpenAIClient and config

type OpenAIEvaluator struct {
	Client *OpenAIClient
}

func NewOpenAIEvaluator(cfg AIAgentConfig) *OpenAIEvaluator {
	return &OpenAIEvaluator{
		Client: NewOpenAIClient(cfg.Key, cfg.Model, cfg.MaxTokens, cfg.Temperature),
	}
}

func (o *OpenAIEvaluator) Evaluate(ctx context.Context, fileName, code string) (*EvaluationResult, error) {
	lg := logger.NewLogger()
	lg.Debug("openai", "Evaluate", fmt.Sprintf("Evaluando archivo %s", fileName))
	res, err := evaluateCode(ctx, o.Client.Client, fileName, code, o.Client.Model, o.Client.MaxTokens, float32(o.Client.Temperature))
	if err != nil {
		lg.Error("openai", "Evaluate", fmt.Sprintf("Error evaluando %s: %v", fileName, err))
	}
	return res, err
}

// OpenAI-specific client wrapper
type OpenAIClient struct {
	Client      *openai.Client
	Model       string
	MaxTokens   int
	Temperature float64
}

func NewOpenAIClient(apiKey, model string, maxTokens int, temperature float64) *OpenAIClient {
	if apiKey == "" {
		fmt.Printf("Warning: OpenAI API key is empty")
	}
	return &OpenAIClient{
		Client:      openai.NewClient(apiKey),
		Model:       model,
		MaxTokens:   maxTokens,
		Temperature: temperature,
	}
}

// evaluateCode calls the OpenAI API and parses the result
func evaluateCode(ctx context.Context, client *openai.Client, fileName, code, model string, maxTokens int, temperature float32) (*EvaluationResult, error) {
	lg := logger.NewLogger()
	prompt := GetEvaluationPrompt(code)

	lg.Debug("openai", "Evaluate", fmt.Sprintf("Evaluando archivo %s with %s model", fileName, model))

	// Mock mode: when OPENAI_MOCK or global USE_MOCK_MOTOR_AI is set to "true", return a canned response for testing the full pipeline
	if os.Getenv("OPENAI_MOCK") == "true" || os.Getenv("USE_MOCK_MOTOR_AI") == "true" {
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

// EvaluateFiles evaluates files concurrently using OpenAI client (kept here for backwards compatibility)
func EvaluateFiles(ctx context.Context, client *OpenAIClient, files []string, batchSize int) ([]*EvaluationResult, error) {
	results := []*EvaluationResult{}
	sem := make(chan struct{}, batchSize)
	wg := sync.WaitGroup{}
	resCh := make(chan *EvaluationResult)
	lg := logger.NewLogger()
	lg.Info("openai", "EvaluateFiles", fmt.Sprintf("Iniciando evaluación concurrente de %d archivos (batch %d)", len(files), batchSize))

	for _, f := range files {
		wg.Add(1)
		sem <- struct{}{}
		go func(file string) {
			defer func() { <-sem; wg.Done() }()
			contentBytes, _ := os.ReadFile(file)
			res, err := evaluateCode(ctx, client.Client, file, string(contentBytes), client.Model, client.MaxTokens, float32(client.Temperature))
			if err != nil {
				lg.Error("openai", "EvaluateFiles", fmt.Sprintf("Error evaluando %s: %v", file, err))
				return
			}
			resCh <- res
		}(f)
	}

	go func() {
		wg.Wait()
		close(resCh)
	}()

	for r := range resCh {
		results = append(results, r)
	}
	return results, nil
}
