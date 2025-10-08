package ai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"

	openai "github.com/sashabaranov/go-openai"
)

// ------------------------------
// Estructuras de resultado
// ------------------------------

type EvaluationResult struct {
	File                      string                    `json:"file"`
	Score                     int                       `json:"score"`
	FactoresNoCumple           []string                  `json:"factores_no_cumple"`
	ProblemasConcurrencia      []string                  `json:"problemas_concurrencia"`
	RecomendacionesRefactor    string                    `json:"recomendaciones_refactor"`
	RecomendacionesComentarios string                    `json:"recomendaciones_comentarios"`
	Documentacion              string                    `json:"documentacion"`
	EvaluacionFunciones        []FuncionEvaluationResult  `json:"evaluacion_funciones"`
}

type FuncionEvaluationResult struct {
	Funcion            string `json:"funcion"`
	Claridad           string `json:"claridad"`             // Alta / Media / Baja
	Complejidad        string `json:"complejidad"`          // Alta / Media / Baja
	RiesgoConcurrencia string `json:"riesgo_concurrencia"`  // Alto / Medio / Bajo
	Sugerencias        string `json:"sugerencias"`
}

// ------------------------------
// Evaluación de código con OpenAI
// ------------------------------

func EvaluateCode(ctx context.Context, client *openai.Client, fileName, code string) (*EvaluationResult, error) {
	prompt := fmt.Sprintf(`Eres un experto en desarrollo en Go (Golang) y arquitectura de software que sigue los principios de The Twelve-Factor App.
Evalúa el siguiente código considerando:
1. Cumplimiento de los 12 factores (configuración, dependencias, logs, procesos, etc.).
2. Buenas prácticas de Go:
   - Nombres claros y consistentes.
   - Manejo adecuado de errores y defer.
   - Modularidad y claridad en paquetes.
   - Eficiencia y seguridad en la concurrencia usando goroutines y channels.
   - Evitar bloqueos o deadlocks.
   - Uso adecuado de buffers en channels y patrones de sincronización.
3. Oportunidades para mejorar la concurrencia y rendimiento.
4. Recomendaciones de **refactorización** para mantener simplicidad, claridad y facilidad de mantenimiento.
5. Recomendaciones sobre **comentarios claros** y documentación inline.
6. Posibles problemas de mantenimiento o escalabilidad.

Devuelve SOLO un JSON con este formato exacto:

{
  "file": "nombre_del_archivo",
  "score": <0-100>,
  "factores_no_cumple": ["Factor1", "Factor2"],
  "problemas_concurrencia": ["Descripción corta de issues en goroutines/channels"],
  "recomendaciones_refactor": "Texto corto sobre cómo simplificar, clarificar y mejorar el mantenimiento del código",
  "recomendaciones_comentarios": "Sugerencias sobre dónde agregar comentarios y cómo redactarlos para claridad",
  "documentacion": "Markdown con descripción de la arquitectura, patrones de concurrencia, configuración recomendada y buenas prácticas de mantenimiento",
  "evaluacion_funciones": [
    {
      "funcion": "nombre",
      "claridad": "Alta/Media/Baja",
      "complejidad": "Alta/Media/Baja",
      "riesgo_concurrencia": "Alto/Medio/Bajo",
      "sugerencias": "Texto corto"
    }
  ]
}

Código a evaluar:
%s
`, code)

	resp, err := client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model:       "gpt-5",
		Temperature: 0,
		Messages: []openai.ChatCompletionMessage{
			{Role: openai.ChatMessageRoleSystem, Content: "Eres un analista de calidad de código especializado en Go concurrente y arquitectura Twelve-Factor."},
			{Role: openai.ChatMessageRoleUser, Content: prompt},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("error al generar respuesta del modelo: %w", err)
	}

	raw := strings.TrimSpace(resp.Choices[0].Message.Content)

	// Extraer sólo el JSON (por si el modelo agrega texto)
	jsonData := extractJSON(raw)
	if jsonData == "" {
		return nil, errors.New("no se pudo extraer un JSON válido de la respuesta del modelo")
	}

	var result EvaluationResult
	if err := json.Unmarshal([]byte(jsonData), &result); err != nil {
		return nil, fmt.Errorf("error al parsear JSON: %w\nContenido: %s", err, jsonData)
	}

	// Asegurar que siempre tenga el nombre de archivo
	if result.File == "" {
		result.File = fileName
	}

	return &result, nil
}

// ------------------------------
// Utilidad para limpiar respuesta
// ------------------------------

func extractJSON(s string) string {
	re := regexp.MustCompile(`(?s)\{.*\}`)
	match := re.FindString(s)
	return strings.TrimSpace(match)
}
