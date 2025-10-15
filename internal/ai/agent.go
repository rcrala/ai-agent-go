package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	openai "github.com/sashabaranov/go-openai"
)

// ------------------------------
// Estructuras de resultado
// ------------------------------

type EvaluationResult struct {
	File                       string                    `json:"file"`
	Score                      int                       `json:"score"`
	FactoresNoCumple           []string                  `json:"factores_no_cumple"`
	ProblemasConcurrencia      []string                  `json:"problemas_concurrencia"`
	RecomendacionesRefactor    string                    `json:"recomendaciones_refactor"`
	RecomendacionesComentarios string                    `json:"recomendaciones_comentarios"`
	Documentacion              string                    `json:"documentacion"`
	EvaluacionFunciones        []FuncionEvaluationResult `json:"evaluacion_funciones"`
}

type FuncionEvaluationResult struct {
	Funcion            string `json:"funcion"`
	Claridad           string `json:"claridad"`            // Alta / Media / Baja
	Complejidad        string `json:"complejidad"`         // Alta / Media / Baja
	RiesgoConcurrencia string `json:"riesgo_concurrencia"` // Alto / Medio / Bajo
	Sugerencias        string `json:"sugerencias"`
}

// -----------------------------
// CONFIGURACIÓN DEL AGENTE
// -----------------------------

type AgentConfig struct {
	Agents      []AIAgentConfig `json:"Agents"`
	TargetDir   string          `json:"TargetDir,omitempty"`
	GitHubToken string          `json:"GitHubToken,omitempty"`
	GitHubRepo  string          `json:"GitHubRepo,omitempty"`
	BaseBranch  string          `json:"BaseBranch,omitempty"`

	RunSonar              bool `json:"RunSonar,omitempty"`
	SendTeamsNotification bool `json:"SendTeamsNotification,omitempty"`

	SonarHostURL    string `json:"SonarHostURL,omitempty"`
	SonarProjectKey string `json:"SonarProjectKey,omitempty"`
	SonarToken      string `json:"SonarToken,omitempty"`
	TeamsWebhookURL string `json:"TeamsWebhookURL,omitempty"`
}

type AIAgentConfig struct {
	Type        string  `json:"Type"`
	Enabled     bool    `json:"Enabled"`
	Key         string  `json:"Key"`
	Model       string  `json:"Model"`
	MaxTokens   int     `json:"MaxTokens"`
	Temperature float64 `json:"Temperature"`
	BatchSize   int     `json:"BatchSize"`
}

// OpenAIClient envuelve el cliente OpenAI
type OpenAIClient struct {
	Client      *openai.Client
	Model       string
	MaxTokens   int
	Temperature float64
}

// CodeEvaluator defines the interface for all AI agents
type CodeEvaluator interface {
	Evaluate(ctx context.Context, fileName, code string) (*EvaluationResult, error)
}

// NewCodeEvaluator returns the appropriate agent implementation for the given config
func NewCodeEvaluator(agentCfg AIAgentConfig) CodeEvaluator {
	switch strings.ToLower(agentCfg.Type) {
	case "openai":
		return NewOpenAIEvaluator(agentCfg)
	case "copilot":
		return NewCopilotEvaluator(agentCfg)
	case "youcom":
		return NewYouComEvaluator(agentCfg)
	case "anthropic":
		return NewAnthropicEvaluator(agentCfg)
	case "gemini":
		return NewGeminiEvaluator(agentCfg)
	case "cohere":
		return NewCohereEvaluator(agentCfg)
	case "mistral":
		return NewMistralEvaluator(agentCfg)
	default:
		return nil
	}
}

// GetEvaluationPrompt returns the evaluation prompt for any LLM agent
func GetEvaluationPrompt(code string) string {
	return fmt.Sprintf(`Eres un experto en desarrollo en Go (Golang) y arquitectura de software siguiendo los principios de The Twelve-Factor App. Evalúa el siguiente código considerando:

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
7. Evaluación de cada función o método, indicando:
	- Claridad
	- Complejidad
	- Riesgos de concurrencia
	- Sugerencias de mejora

Devuelve el resultado **en JSON con este formato exacto**:

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
			"funcion": "NombreDeLaFuncion",
			"claridad": "Alta/Media/Baja",
			"complejidad": "Alta/Media/Baja",
			"riesgo_concurrencia": "Alto/Medio/Bajo",
			"sugerencias": "Texto corto con mejoras específicas"
		}
	]
}

Código a evaluar:
%s
`, code)
}

// EvaluateFilesGeneric evalúa todos los archivos usando cualquier agente que implemente CodeEvaluator
func EvaluateFilesGeneric(ctx context.Context, evaluator CodeEvaluator, files []string) ([]*EvaluationResult, error) {
	results := []*EvaluationResult{}
	for _, file := range files {
		contentBytes, err := os.ReadFile(file)
		if err != nil {
			return nil, fmt.Errorf("error leyendo archivo %s: %w", file, err)
		}
		res, err := evaluator.Evaluate(ctx, file, string(contentBytes))
		if err != nil {
			fmt.Println("Error evaluando:", file, err)
			continue
		}
		results = append(results, res)
	}
	return results, nil
}

func NewOpenAIClient(apiKey, model string, maxTokens int, temperature float64) *OpenAIClient {

	if apiKey == "" {
		fmt.Printf("Error evaluando: %s", apiKey)
	}

	return &OpenAIClient{
		Client:      openai.NewClient(apiKey),
		Model:       model,
		MaxTokens:   maxTokens,
		Temperature: temperature,
	}
}

// ScanFiles busca archivos según extensiones
func ScanFiles(root string, exts []string) []string {
	files := []string{}
	filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if !d.IsDir() {
			for _, ext := range exts {
				if strings.HasSuffix(path, ext) {
					files = append(files, path)
				}
			}
		}
		return nil
	})
	return files
}

// EvaluateFiles evalúa todos los archivos con concurrencia segura
func EvaluateFiles(ctx context.Context, client *OpenAIClient, files []string, batchSize int) ([]*EvaluationResult, error) {
	results := []*EvaluationResult{}
	sem := make(chan struct{}, batchSize)
	wg := sync.WaitGroup{}
	resCh := make(chan *EvaluationResult)

	for _, f := range files {
		wg.Add(1)
		sem <- struct{}{}
		go func(file string) {
			defer wg.Done()
			defer func() { <-sem }()
			contentBytes, _ := os.ReadFile(file)
			res, err := EvaluateCode(ctx, client.Client, file, string(contentBytes), client.Model, client.MaxTokens, float32(client.Temperature))
			if err != nil {
				fmt.Println("Error evaluando:", file, err)
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

// GenerateMarkdown genera un Markdown combinando resultados de evaluación
func GenerateMarkdown(results []*EvaluationResult) string {
	md := ""
	for _, r := range results {
		md += fmt.Sprintf("## %s\n\n%s\n\n---\n\n", r.File, r.RecomendacionesRefactor)
	}
	return md
}

// RunSonarAnalysis ejecuta SonarQube (puede invocar CLI o API)
func RunSonarAnalysis(targetDir, hostURL, projectKey, token string) error {
	// Implementar llamada a SonarQube CLI/API
	fmt.Println("Simulando ejecución de SonarQube en:", targetDir)
	return nil
}

// LoadConfig carga config desde JSON y sobrescribe con ENV
func LoadConfig(path string, filename string) (*AgentConfig, error) {
	cfg := &AgentConfig{}

	filepath := filepath.Join(path, filename)
	file, err := os.ReadFile(filepath)
	if err != nil {
		return nil, fmt.Errorf("error leyendo config default: %w", err)
	}
	if err := json.Unmarshal(file, cfg); err != nil {
		return nil, fmt.Errorf("error parseando config default: %w", err)
	}

	// Sobrescribir claves de agentes individuales por ENV (después de cargar config)
	for i, agent := range cfg.Agents {
		envKey := ""
		switch strings.ToLower(agent.Type) {
		case "openai":
			envKey = "OPENAI_API_KEY"
		case "copilot":
			envKey = "COPILOT_API_KEY"
		}
		if envKey != "" {
			if v := os.Getenv(envKey); v != "" {
				cfg.Agents[i].Key = v
			}
		}
	}

	// Sobrescribir con ENV solo para campos generales
	envMap := map[string]*string{
		"TARGET_DIR":        &cfg.TargetDir,
		"GITHUB_TOKEN":      &cfg.GitHubToken,
		"GITHUB_REPO":       &cfg.GitHubRepo,
		"BASE_BRANCH":       &cfg.BaseBranch,
		"SONAR_HOST_URL":    &cfg.SonarHostURL,
		"SONAR_PROJECT_KEY": &cfg.SonarProjectKey,
		"SONAR_TOKEN":       &cfg.SonarToken,
		"TEAMS_WEBHOOK_URL": &cfg.TeamsWebhookURL,
	}

	for key, ptr := range envMap {
		if v := os.Getenv(key); v != "" {
			*ptr = v
		}
	}

	if cfg.TargetDir == "" {
		cfg.TargetDir = "./"
	}
	if cfg.BaseBranch == "" {
		cfg.BaseBranch = "main"
	}

	return cfg, nil
}

// -----------------------------
// EVALUACIÓN DE CÓDIGO
// -----------------------------

func EvaluateCode(ctx context.Context, client *openai.Client, fileName, code, model string, maxTokens int, temperature float32) (*EvaluationResult, error) {
	prompt := GetEvaluationPrompt(code)

	resp, err := client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model:       model,
		Messages:    []openai.ChatCompletionMessage{{Role: openai.ChatMessageRoleUser, Content: prompt}},
		Temperature: temperature,
		MaxTokens:   maxTokens,
	})
	if err != nil {
		return nil, err
	}

	content := strings.TrimSpace(resp.Choices[0].Message.Content)

	var result EvaluationResult
	if err := json.Unmarshal([]byte(content), &result); err != nil {
		return nil, fmt.Errorf("error parseando JSON de AI: %w", err)
	}

	return &result, nil
}
