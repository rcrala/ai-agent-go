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
	prompt := fmt.Sprintf(`Eres un experto en desarrollo en Go (Golang) y arquitectura de software siguiendo los principios de The Twelve-Factor App. Evalúa el siguiente código considerando:
1. Cumplimiento de los 12 factores (configuración, dependencias, logs, procesos, etc.).
2. Buenas prácticas de Go, incluyendo:
   - Nombres claros y consistentes de variables y funciones.
   - Manejo adecuado de errores y defer.
   - Modularidad y claridad en paquetes.
   - Eficiencia y seguridad en la concurrencia usando goroutines y channels.
   - Evitar bloqueos o deadlocks.
   - Uso adecuado de buffers en channels y patrones de sincronización.
3. Oportunidades para mejorar la concurrencia y rendimiento.
4. Recomendaciones de **refactorización** para mantener simplicidad y claridad, mejorar mantenimiento y legibilidad.
5. Recomendaciones sobre **comentarios claros** y documentación inline para facilitar la comprensión del código.
6. Posibles problemas de mantenimiento o escalabilidad.

Devuelve el resultado **en JSON con este formato exacto**:

{
  "file": "nombre_del_archivo",
  "score": <0-100>,
  "factores_no_cumple": ["Factor1", "Factor2"],
  "problemas_concurrencia": ["Descripción corta de issues en goroutines/channels"],
  "recomendaciones_refactor": "Texto corto sobre cómo simplificar, clarificar y mejorar el mantenimiento del código",
  "recomendaciones_comentarios": "Sugerencias sobre dónde agregar comentarios y cómo redactarlos para claridad",
  "documentacion": "Markdown con descripción de la arquitectura, patrones de concurrencia, configuración recomendada y buenas prácticas de mantenimiento"
}

Código a evaluar:
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
