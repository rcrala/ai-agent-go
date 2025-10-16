package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
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
	UseMockMotorAI        bool `json:"UseMockMotorAI,omitempty"` // Global mock switch for all agents

	SonarHostURL    string `json:"SonarHostURL,omitempty"`
	SonarProjectKey string `json:"SonarProjectKey,omitempty"`
	SonarToken      string `json:"SonarToken,omitempty"`
	TeamsWebhookURL string `json:"TeamsWebhookURL,omitempty"`
}

type AIAgentConfig struct {
	Type              string  `json:"Type"`
	Enabled           bool    `json:"Enabled"`
	Key               string  `json:"Key"`
	Model             string  `json:"Model"`
	MaxTokens         int     `json:"MaxTokens"`
	Temperature       float64 `json:"Temperature"`
	BatchSize         int     `json:"BatchSize"`
	RequestIntervalMs int     `json:"RequestIntervalMs,omitempty"` // Optional: ms to wait between requests
	UseMockMotorAI    bool    `json:"UseMockMotorAI,omitempty"`
	// Optional retry/backoff/concurrency controls
	MaxRetries         int  `json:"MaxRetries,omitempty"`         // number of retries on retryable errors
	BackoffInitialMs   int  `json:"BackoffInitialMs,omitempty"`   // initial backoff in ms
	MaxConcurrency     int  `json:"MaxConcurrency,omitempty"`     // max concurrent evaluations
	BackoffJitter      bool `json:"BackoffJitter,omitempty"`      // add random jitter to backoff (default: true)
	CircuitBreakerMax  int  `json:"CircuitBreakerMax,omitempty"`  // max consecutive failures before circuit opens (0=disabled)
	CircuitBreakerWait int  `json:"CircuitBreakerWait,omitempty"` // seconds to wait when circuit is open
}

// OpenAIClient envuelve el cliente OpenAI
// OpenAIClient moved to internal/ai/openai_agent.go

// CodeEvaluator defines the interface for all AI agents
type CodeEvaluator interface {
	Evaluate(ctx context.Context, fileName, code string) (*EvaluationResult, error)
}

// EvaluationMetrics tracks metrics for evaluation runs
type EvaluationMetrics struct {
	TotalAttempts       int
	SuccessCount        int
	FailureCount        int
	RetryCount          int
	RateLimitCount      int // HTTP 429 errors
	TotalLatencyMs      int64
	CircuitBreakerTrips int
}

// CircuitBreakerState tracks circuit breaker state per agent
type CircuitBreakerState struct {
	mu                  sync.Mutex
	consecutiveFailures int
	isOpen              bool
	openUntil           time.Time
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
// EvaluateFilesGeneric evalúa los archivos usando batching y respetando los
// parámetros del agente: BatchSize y RequestIntervalMs. Si batchSize es 0
// o negativo se procesan todos los archivos en una sola tanda.
// Recopila métricas y gestiona circuit breaker si está configurado.
func EvaluateFilesGeneric(ctx context.Context, evaluator CodeEvaluator, files []string, agentCfg AIAgentConfig) ([]*EvaluationResult, error) {
	results, metrics := EvaluateFilesGenericWithMetrics(ctx, evaluator, files, agentCfg)

	// Log metrics summary
	if metrics.TotalAttempts > 0 {
		avgLatency := metrics.TotalLatencyMs / int64(metrics.TotalAttempts)
		fmt.Printf("[Metrics] Agent: %s | Success: %d | Failures: %d | Retries: %d | RateLimits(429): %d | AvgLatency: %dms | CircuitBreaks: %d\n",
			agentCfg.Type, metrics.SuccessCount, metrics.FailureCount, metrics.RetryCount, metrics.RateLimitCount, avgLatency, metrics.CircuitBreakerTrips)
	}

	return results, nil
}

// EvaluateFilesGenericWithMetrics returns results and detailed metrics
func EvaluateFilesGenericWithMetrics(ctx context.Context, evaluator CodeEvaluator, files []string, agentCfg AIAgentConfig) ([]*EvaluationResult, *EvaluationMetrics) {
	metrics := &EvaluationMetrics{}

	if len(files) == 0 {
		return []*EvaluationResult{}, metrics
	}

	// Initialize circuit breaker if configured
	var cb *CircuitBreakerState
	if agentCfg.CircuitBreakerMax > 0 {
		cb = &CircuitBreakerState{}
	}

	batchSize := determineBatchSize(agentCfg.BatchSize, len(files))
	maxConc := determineMaxConcurrency(agentCfg.MaxConcurrency, batchSize)
	sem := make(chan struct{}, maxConc)

	evaluateWithRetries := createFileEvaluatorWithMetrics(ctx, evaluator, agentCfg, sem, metrics, cb)

	results, _ := processBatches(files, batchSize, agentCfg.RequestIntervalMs, evaluateWithRetries)
	return results, metrics
}

// determineBatchSize calculates the appropriate batch size
func determineBatchSize(configBatchSize, totalFiles int) int {
	if configBatchSize <= 0 {
		return totalFiles
	}
	return configBatchSize
}

// determineMaxConcurrency calculates the maximum concurrency limit
func determineMaxConcurrency(configMaxConc, batchSize int) int {
	if configMaxConc <= 0 {
		return batchSize
	}
	return configMaxConc
}

// createFileEvaluator returns a function that evaluates a single file with semaphore control
func createFileEvaluator(ctx context.Context, evaluator CodeEvaluator, agentCfg AIAgentConfig, sem chan struct{}) func(string) (*EvaluationResult, error) {
	return func(fpath string) (*EvaluationResult, error) {
		sem <- struct{}{}
		defer func() { <-sem }()

		contentBytes, err := os.ReadFile(fpath)
		if err != nil {
			return nil, fmt.Errorf("error leyendo archivo %s: %w", fpath, err)
		}
		return evaluateWithBackoff(ctx, evaluator, fpath, string(contentBytes), agentCfg)
	}
}

// createFileEvaluatorWithMetrics returns a function that evaluates with metrics and circuit breaker
func createFileEvaluatorWithMetrics(ctx context.Context, evaluator CodeEvaluator, agentCfg AIAgentConfig, sem chan struct{}, metrics *EvaluationMetrics, cb *CircuitBreakerState) func(string) (*EvaluationResult, error) {
	return func(fpath string) (*EvaluationResult, error) {
		sem <- struct{}{}
		defer func() { <-sem }()

		contentBytes, err := os.ReadFile(fpath)
		if err != nil {
			return nil, fmt.Errorf("error leyendo archivo %s: %w", fpath, err)
		}
		return evaluateWithBackoffAndMetrics(ctx, evaluator, fpath, string(contentBytes), agentCfg, metrics, cb)
	}
}

// processBatches processes files in batches with concurrency control
func processBatches(files []string, batchSize, intervalMs int, evaluateFunc func(string) (*EvaluationResult, error)) ([]*EvaluationResult, error) {
	results := []*EvaluationResult{}

	for i := 0; i < len(files); i += batchSize {
		batch := getBatch(files, i, batchSize)
		batchResults := processBatch(batch, evaluateFunc)
		results = append(results, batchResults...)

		if shouldSleep(intervalMs, i+batchSize, len(files)) {
			time.Sleep(time.Duration(intervalMs) * time.Millisecond)
		}
	}

	return results, nil
}

// getBatch extracts a batch of files starting at the given index
func getBatch(files []string, start, batchSize int) []string {
	end := start + batchSize
	if end > len(files) {
		end = len(files)
	}
	return files[start:end]
}

// processBatch processes a single batch of files concurrently
func processBatch(batch []string, evaluateFunc func(string) (*EvaluationResult, error)) []*EvaluationResult {
	var wg sync.WaitGroup
	var mu sync.Mutex
	results := []*EvaluationResult{}

	for _, file := range batch {
		wg.Add(1)
		go func(f string) {
			defer wg.Done()
			res, err := evaluateFunc(f)
			if err != nil {
				fmt.Printf("Error evaluando: %s %v\n", f, err)
				return
			}
			mu.Lock()
			results = append(results, res)
			mu.Unlock()
		}(file)
	}

	wg.Wait()
	return results
}

// shouldSleep determines if we should sleep between batches
func shouldSleep(intervalMs, currentEnd, totalFiles int) bool {
	return intervalMs > 0 && currentEnd < totalFiles
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

// GenerateMarkdown genera un Markdown combinando resultados de evaluación
func GenerateMarkdown(results []*EvaluationResult) string {
	const paragraphFmt = "%s\n\n"
	md := strings.Builder{}
	if len(results) == 0 {
		return ""
	}
	for _, r := range results {
		md.WriteString(fmt.Sprintf("## %s\n\n", r.File))
		md.WriteString(fmt.Sprintf("**Score:** %d/100\n\n", r.Score))

		writeListSection(&md, "### Factores no cumplidos", r.FactoresNoCumple)
		writeListSection(&md, "### Problemas de concurrencia", r.ProblemasConcurrencia)

		writeParagraphSection(&md, "### Recomendaciones de refactorización", r.RecomendacionesRefactor, paragraphFmt)
		writeParagraphSection(&md, "### Recomendaciones sobre comentarios", r.RecomendacionesComentarios, paragraphFmt)
		writeParagraphSection(&md, "### Documentación recomendada", r.Documentacion, paragraphFmt)

		if len(r.EvaluacionFunciones) > 0 {
			md.WriteString("### Evaluación por función\n\n")
			md.WriteString("| Función | Claridad | Complejidad | Riesgo concurrencia | Sugerencias |\n")
			md.WriteString("|---|---:|---:|---:|---|\n")
			for _, f := range r.EvaluacionFunciones {
				funcName := strings.ReplaceAll(f.Funcion, "|", "\\|")
				suger := strings.ReplaceAll(f.Sugerencias, "|", "\\|")
				md.WriteString(fmt.Sprintf("| %s | %s | %s | %s | %s |\n", funcName, f.Claridad, f.Complejidad, f.RiesgoConcurrencia, suger))
			}
			md.WriteString("\n")
		}

		md.WriteString("---\n\n")
	}
	return md.String()
}

// helper: write a list section (title + bullet list) if items exist
func writeListSection(sb *strings.Builder, title string, items []string) {
	if len(items) == 0 {
		return
	}
	sb.WriteString(title + "\n\n")
	for _, it := range items {
		sb.WriteString(fmt.Sprintf("- %s\n", it))
	}
	sb.WriteString("\n")
}

// helper: write a paragraph section (title + paragraph) if content present
func writeParagraphSection(sb *strings.Builder, title, content, fmtStr string) {
	if strings.TrimSpace(content) == "" {
		return
	}
	sb.WriteString(title + "\n\n")
	sb.WriteString(fmt.Sprintf(fmtStr, content))
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

	// Refactor: Move agent env override logic to helper
	overrideAgentKeysWithEnv(cfg)

	// Sobrescribir con ENV solo para campos generales
	applyGeneralEnvOverrides(cfg)

	// Global mock override
	if v := os.Getenv("USE_MOCK_MOTOR_AI"); v != "" {
		cfg.UseMockMotorAI = parseBoolEnv(v)
	}

	if cfg.TargetDir == "" {
		cfg.TargetDir = "./"
	}
	if cfg.BaseBranch == "" {
		cfg.BaseBranch = "main"
	}

	return cfg, nil
}

// overrideAgentKeysWithEnv moves agent key/env override logic out of LoadConfig
func overrideAgentKeysWithEnv(cfg *AgentConfig) {
	for i, agent := range cfg.Agents {
		envKey := ""
		switch strings.ToLower(agent.Type) {
		case "openai":
			envKey = "OPENAI_API_KEY"
		case "copilot":
			envKey = "COPILOT_API_KEY"
		case "cohere":
			envKey = "COHERE_API_KEY"
		}

		if envKey != "" {
			if v := os.Getenv(envKey); v != "" {
				cfg.Agents[i].Key = v
			}
		}
		// Global mock override for agent (if ever needed per agent)
		if v := os.Getenv("USE_MOCK_MOTOR_AI"); v != "" {
			cfg.Agents[i].UseMockMotorAI = parseBoolEnv(v)
		}
	}
}

// parseBoolEnv interpreta valores de entorno como booleanos ('true','1','yes')
func parseBoolEnv(v string) bool {
	v = strings.ToLower(strings.TrimSpace(v))
	return v == "true" || v == "1" || v == "yes" || v == "y"
}

func applyGeneralEnvOverrides(cfg *AgentConfig) {
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
}

// evaluateWithBackoff performs evaluation with retries, exponential backoff, jitter, and metrics
func evaluateWithBackoff(ctx context.Context, evaluator CodeEvaluator, filePath, content string, cfg AIAgentConfig) (*EvaluationResult, error) {
	return evaluateWithBackoffAndMetrics(ctx, evaluator, filePath, content, cfg, nil, nil)
}

// evaluateWithBackoffAndMetrics performs evaluation with full observability
func evaluateWithBackoffAndMetrics(ctx context.Context, evaluator CodeEvaluator, filePath, content string, cfg AIAgentConfig, metrics *EvaluationMetrics, cb *CircuitBreakerState) (*EvaluationResult, error) {
	if err := checkCircuitBreaker(cb, metrics); err != nil {
		return nil, err
	}

	attempts := calculateAttempts(cfg.MaxRetries)
	backoffMs := getInitialBackoff(cfg.BackoffInitialMs)

	return executeWithRetries(ctx, evaluator, filePath, content, cfg, metrics, cb, attempts, backoffMs)
}

// checkCircuitBreaker validates circuit breaker state
func checkCircuitBreaker(cb *CircuitBreakerState, metrics *EvaluationMetrics) error {
	if cb != nil && cb.isCircuitOpen() {
		if metrics != nil {
			metrics.CircuitBreakerTrips++
		}
		return fmt.Errorf("circuit breaker is open for this agent")
	}
	return nil
}

// calculateAttempts determines retry attempts from config
func calculateAttempts(maxRetries int) int {
	if maxRetries > 0 {
		return maxRetries + 1
	}
	return 1
}

// getInitialBackoff returns initial backoff delay
func getInitialBackoff(configBackoff int) int {
	if configBackoff <= 0 {
		return 500
	}
	return configBackoff
}

// executeWithRetries handles the retry loop with backoff
func executeWithRetries(ctx context.Context, evaluator CodeEvaluator, filePath, content string, cfg AIAgentConfig, metrics *EvaluationMetrics, cb *CircuitBreakerState, attempts, backoffMs int) (*EvaluationResult, error) {
	var lastErr error

	for attempt := 0; attempt < attempts; attempt++ {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}

		result, err := performSingleAttempt(ctx, evaluator, filePath, content, metrics)
		if err == nil {
			recordSuccess(metrics, cb)
			return result, nil
		}

		lastErr = err
		recordFailureMetrics(err, attempt, metrics)

		if !isRetryableError(err) {
			recordCircuitBreakerFailure(cb, cfg, metrics)
			break
		}

		if attempt == attempts-1 {
			recordCircuitBreakerFailure(cb, cfg, metrics)
			break
		}

		sleepWithBackoff(attempt, backoffMs, cfg)
	}

	return nil, lastErr
}

// performSingleAttempt executes a single evaluation attempt
func performSingleAttempt(ctx context.Context, evaluator CodeEvaluator, filePath, content string, metrics *EvaluationMetrics) (*EvaluationResult, error) {
	start := time.Now()
	result, err := evaluator.Evaluate(ctx, filePath, content)
	latency := time.Since(start).Milliseconds()

	if metrics != nil {
		metrics.TotalAttempts++
		metrics.TotalLatencyMs += latency
	}

	return result, err
}

// recordSuccess updates metrics and circuit breaker on successful evaluation
func recordSuccess(metrics *EvaluationMetrics, cb *CircuitBreakerState) {
	if metrics != nil {
		metrics.SuccessCount++
	}
	if cb != nil {
		cb.recordSuccess()
	}
}

// recordFailureMetrics tracks failure-related metrics
func recordFailureMetrics(err error, attempt int, metrics *EvaluationMetrics) {
	if metrics == nil {
		return
	}

	if attempt > 0 {
		metrics.RetryCount++
	}
	metrics.FailureCount++

	if httpErr, ok := err.(*HTTPError); ok && httpErr.StatusCode == 429 {
		metrics.RateLimitCount++
	}
}

// recordCircuitBreakerFailure handles circuit breaker failure recording
func recordCircuitBreakerFailure(cb *CircuitBreakerState, cfg AIAgentConfig, metrics *EvaluationMetrics) {
	if cb == nil {
		return
	}

	tripped := cb.recordFailure(cfg.CircuitBreakerMax, cfg.CircuitBreakerWait)
	if tripped && metrics != nil {
		metrics.CircuitBreakerTrips++
	}
}

// sleepWithBackoff applies exponential backoff with jitter
func sleepWithBackoff(attempt, backoffMs int, cfg AIAgentConfig) {
	sleepMs := backoffMs * (1 << attempt)

	if shouldUseJitter(cfg) {
		sleepMs = addJitter(sleepMs)
	}

	time.Sleep(time.Duration(sleepMs) * time.Millisecond)
}

// shouldUseJitter determines if jitter should be applied
func shouldUseJitter(cfg AIAgentConfig) bool {
	return cfg.BackoffJitter || (!cfg.BackoffJitter && cfg.MaxRetries > 0)
}

// addJitter adds random jitter to backoff delay
func addJitter(sleepMs int) int {
	jitterRange := sleepMs / 4
	if jitterRange > 0 {
		jitter := time.Now().UnixNano() % int64(jitterRange)
		sleepMs += int(jitter)
	}
	return sleepMs
}

// HTTPError wraps an HTTP error response with status code for structured retry logic
type HTTPError struct {
	StatusCode int
	Message    string
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("HTTP %d: %s", e.StatusCode, e.Message)
}

// CircuitBreaker methods
func (cb *CircuitBreakerState) recordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.consecutiveFailures = 0
	cb.isOpen = false
}

func (cb *CircuitBreakerState) recordFailure(maxFailures int, waitSeconds int) bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.consecutiveFailures++
	if maxFailures > 0 && cb.consecutiveFailures >= maxFailures {
		cb.isOpen = true
		cb.openUntil = time.Now().Add(time.Duration(waitSeconds) * time.Second)
		return true // circuit opened
	}
	return false
}

func (cb *CircuitBreakerState) isCircuitOpen() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	if cb.isOpen && time.Now().After(cb.openUntil) {
		// circuit can be retried (half-open state)
		cb.isOpen = false
		cb.consecutiveFailures = 0
	}
	return cb.isOpen
}

// isRetryableError inspects the error to decide if a retry may help.
// First checks for structured HTTPError with status code, then falls back to string inspection.
func isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	// Check for structured HTTPError with status code
	var httpErr *HTTPError
	if errors, ok := err.(*HTTPError); ok {
		httpErr = errors
	}
	if httpErr != nil {
		// Retry on 429 (rate limit) and 5xx (server errors)
		return httpErr.StatusCode == 429 || (httpErr.StatusCode >= 500 && httpErr.StatusCode < 600)
	}

	// Fallback: string-based detection for non-structured errors
	s := strings.ToLower(err.Error())
	if strings.Contains(s, "429") || strings.Contains(s, "too many requests") || strings.Contains(s, "rate limit") {
		return true
	}
	if strings.Contains(s, "timeout") || strings.Contains(s, "temporary") || strings.Contains(s, "connection reset") {
		return true
	}
	if strings.Contains(s, "quota") || strings.Contains(s, "exceeded") {
		return true
	}
	return false
}
