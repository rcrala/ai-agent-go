package ai

import (
	logger "ai-agent-go/internal/logger"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
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
	lg := logger.NewLogger()
	prompt := GetEvaluationPrompt(code)
	lg.Debug("cohere", "evaluateCohereCode", fmt.Sprintf("Calling Cohere generate for %s (model=%s)", fileName, c.Client.Model))

	reqBody := map[string]interface{}{
		"model":       c.Client.Model,
		"prompt":      prompt,
		"max_tokens":  c.Client.MaxTokens,
		"temperature": c.Client.Temperature,
	}

	b, err := json.Marshal(reqBody)
	if err != nil {
		lg.Error("cohere", "evaluateCohereCode", fmt.Sprintf("error marshalling request: %v", err))
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.cohere.ai/v1/generate", bytes.NewReader(b))
	if err != nil {
		lg.Error("cohere", "evaluateCohereCode", fmt.Sprintf("error creating request: %v", err))
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.Client.APIKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		lg.Error("cohere", "evaluateCohereCode", fmt.Sprintf("http request error: %v", err))
		return nil, err
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		lg.Error("cohere", "evaluateCohereCode", fmt.Sprintf("error reading response: %v", err))
		return nil, err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		lg.Error("cohere", "evaluateCohereCode", fmt.Sprintf("non-2xx response: %d body=%s", resp.StatusCode, string(bodyBytes)))
		return nil, fmt.Errorf("cohere API returned status %d", resp.StatusCode)
	}

	// Expected response: { "generations": [ { "text": "..." } ] }
	var cohResp struct {
		Generations []struct {
			Text string `json:"text"`
		} `json:"generations"`
	}
	if err := json.Unmarshal(bodyBytes, &cohResp); err != nil {
		lg.Error("cohere", "evaluateCohereCode", fmt.Sprintf("error parsing Cohere response JSON: %v body=%s", err, string(bodyBytes)))
		return nil, err
	}
	if len(cohResp.Generations) == 0 {
		lg.Error("cohere", "evaluateCohereCode", fmt.Sprintf("empty generations in response: %s", string(bodyBytes)))
		return nil, fmt.Errorf("empty response from cohere")
	}

	content := strings.TrimSpace(cohResp.Generations[0].Text)

	var result EvaluationResult
	if err := json.Unmarshal([]byte(content), &result); err != nil {
		lg.Error("cohere", "evaluateCohereCode", fmt.Sprintf("error parseando JSON de Cohere: %v content=%s", err, content))
		return nil, fmt.Errorf("error parseando JSON de AI: %w", err)
	}
	return &result, nil
}

func evaluateCohereCodeMock(fileName, code string, c *CohereEvaluator) (*EvaluationResult, error) {
	lg := logger.NewLogger()
	lg.Debug("cohere", "evaluateCohereCodeMock", fmt.Sprintf("Evaluando archivo %s in Mock mode %s", fileName, strconv.FormatBool(c.Client.IsMockEnabled)))
	return &EvaluationResult{
		File:                       fileName,
		Score:                      76,
		FactoresNoCumple:           []string{"Dependencias no declaradas (mock - Cohere)"},
		ProblemasConcurrencia:      []string{},
		RecomendacionesRefactor:    "Reducir complejidad y separar responsabilidades (mock).",
		RecomendacionesComentarios: "Agregar comentarios y ejemplos de uso (mock).",
		Documentacion:              "Mock: recomendaciones generadas por Cohere.",
		EvaluacionFunciones: []FuncionEvaluationResult{{
			Funcion:            "MockFunc",
			Claridad:           "Media",
			Complejidad:        "Media",
			RiesgoConcurrencia: "Medio",
			Sugerencias:        "Usar canales o mutex donde corresponda.",
		}},
	}, nil
}
