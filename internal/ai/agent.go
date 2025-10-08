package ai

import (
	"context"
	"fmt"
	"strings"

	openai "github.com/sashabaranov/go-openai"
)

type EvaluationResult struct {
	File            string   `json:"file"`
	Score           int      `json:"score"`
	FactorsNoCumple []string `json:"factores_no_cumple"`
	Recomendaciones string   `json:"recomendaciones"`
	Documentacion   string   `json:"documentacion"`
}

func EvaluateCode(ctx context.Context, client *openai.Client, fileName, code string) (*EvaluationResult, error) {
	prompt := fmt.Sprintf(`
Evalúa el siguiente código según los principios de The Twelve-Factor App:
Devuelve el resultado en JSON con este formato exacto:
{
  "file": "nombre_del_archivo",
  "score": <0-100>,
  "factores_no_cumple": ["Factor1", "Factor2"],
  "recomendaciones": "Texto corto sobre cómo refactorizar para cumplir.",
  "documentacion": "Markdown con descripción de la arquitectura o configuración recomendada."
}
Código:
%s
`, code)

	resp, err := client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model: "gpt-5",
		Messages: []openai.ChatCompletionMessage{
			{Role: openai.ChatMessageRoleUser, Content: prompt},
		},
		Temperature: 0,
	})
	if err != nil {
		return nil, err
	}

	content := resp.Choices[0].Message.Content
	content = strings.TrimSpace(content)

	result := &EvaluationResult{
		File:            fileName,
		Score:           0,
		FactorsNoCumple: []string{},
		Recomendaciones: content,
		Documentacion:   "",
	}

	return result, nil
}
