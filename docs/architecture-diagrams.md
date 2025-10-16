# Arquitectura Visual del Sistema
## AI Agent - Diagramas y Flujos

Este documento proporciona representaciones visuales de la arquitectura y flujos del AI Agent.

---

## 🏗️ Arquitectura General

```
┌─────────────────────────────────────────────────────────────────┐
│                    GITHUB REPOSITORY                             │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐       │
│  │  main    │  │   dev    │  │ feature  │  │   PR     │       │
│  └────┬─────┘  └────┬─────┘  └────┬─────┘  └────┬─────┘       │
│       │             │             │             │               │
│       └─────────────┴─────────────┴─────────────┘               │
│                          │                                       │
│                          │ Push/PR Trigger                       │
└──────────────────────────┼───────────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────────────┐
│               GITHUB ACTIONS RUNNER (Ubuntu)                     │
│                                                                   │
│  ┌────────────────────────────────────────────────────────────┐ │
│  │ Step 1: Checkout Code                                      │ │
│  │   actions/checkout@v4                                      │ │
│  └────────────────────────────────────────────────────────────┘ │
│                           │                                      │
│  ┌────────────────────────▼───────────────────────────────────┐ │
│  │ Step 2: Download AI Agent Binary                          │ │
│  │   wget ai-agent-linux + chmod +x                          │ │
│  └────────────────────────────────────────────────────────────┘ │
│                           │                                      │
│  ┌────────────────────────▼───────────────────────────────────┐ │
│  │ Step 3: Set Environment Variables                         │ │
│  │   OPENAI_API_KEY, GITHUB_TOKEN, etc.                     │ │
│  └────────────────────────────────────────────────────────────┘ │
│                           │                                      │
│  ┌────────────────────────▼───────────────────────────────────┐ │
│  │ Step 4: Execute Binary                                    │ │
│  │   ./ai-agent-linux                                        │ │
│  └────────────────────────┬───────────────────────────────────┘ │
└───────────────────────────┼──────────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────────┐
│                    AI AGENT BINARY                               │
│                                                                   │
│  ┌────────────────────────────────────────────────────────────┐ │
│  │ main.go                                                    │ │
│  │  │                                                         │ │
│  │  ├─► LoadConfig()                                         │ │
│  │  │    ├─ config_AIAgent.json (defaults)                   │ │
│  │  │    └─ Environment Variables Override                   │ │
│  │  │                                                         │ │
│  │  ├─► ScanFiles(targetDir)                                 │ │
│  │  │    └─ Find *.go, *.py files                            │ │
│  │  │                                                         │ │
│  │  ├─► runAIAgents()                                        │ │
│  │  │    │                                                    │ │
│  │  │    └─► For each enabled agent:                         │ │
│  │  │         ├─ NewCodeEvaluator(agentCfg)                  │ │
│  │  │         └─ EvaluateFilesGeneric()                      │ │
│  │  │              │                                          │ │
│  │  │              ├─ Batch processing                       │ │
│  │  │              ├─ Rate limiting                          │ │
│  │  │              ├─ Circuit breaker                        │ │
│  │  │              └─ Metrics tracking                       │ │
│  │  │                                                         │ │
│  │  ├─► GenerateMarkdown(results)                            │ │
│  │  │    └─ ARQUITECTURE_COMPLIANCE.md                       │ │
│  │  │                                                         │ │
│  │  ├─► CreateOrUpdatePR()                                   │ │
│  │  │    └─ Push report to GitHub                            │ │
│  │  │                                                         │ │
│  │  └─► sendTeamsNotification()                              │ │
│  │       └─ POST to Teams webhook                            │ │
│  └────────────────────────────────────────────────────────────┘ │
└──────────┬────────────────────────────────────┬─────────────────┘
           │                                    │
           │ API Calls                          │ Notifications
           ▼                                    ▼
┌──────────────────────┐          ┌──────────────────────────────┐
│  AI PROVIDERS        │          │  EXTERNAL SERVICES           │
│                      │          │                              │
│  ┌────────────────┐ │          │  ┌────────────────────────┐ │
│  │ OpenAI API     │ │          │  │ GitHub API             │ │
│  │ gpt-4o-mini    │ │          │  │  - Create PR           │ │
│  └────────────────┘ │          │  │  - Post comments       │ │
│                      │          │  └────────────────────────┘ │
│  ┌────────────────┐ │          │                              │
│  │ Cohere API     │ │          │  ┌────────────────────────┐ │
│  │ command        │ │          │  │ Microsoft Teams        │ │
│  └────────────────┘ │          │  │  - Webhook POST        │ │
│                      │          │  └────────────────────────┘ │
│  ┌────────────────┐ │          │                              │
│  │ Copilot API    │ │          │  ┌────────────────────────┐ │
│  └────────────────┘ │          │  │ SonarQube (optional)   │ │
│                      │          │  └────────────────────────┘ │
└──────────────────────┘          └──────────────────────────────┘
```

---

## 🔄 Flujo de Evaluación de Archivos

```
EvaluateFilesGeneric()
    │
    ├─► determineBatchSize()
    │    └─ BatchSize from config (default: all files)
    │
    ├─► determineMaxConcurrency()
    │    └─ MaxConcurrency from config (default: BatchSize)
    │
    ├─► Initialize Circuit Breaker
    │    └─ If CircuitBreakerMax > 0
    │
    └─► processBatches()
         │
         ├─► For each batch:
         │    │
         │    ├─► getBatch(files, index, batchSize)
         │    │    └─ Extract subset of files
         │    │
         │    ├─► processBatch()
         │    │    │
         │    │    ├─► For each file in batch (parallel):
         │    │    │    │
         │    │    │    ├─► Acquire semaphore
         │    │    │    │    └─ sem <- struct{}
         │    │    │    │
         │    │    │    ├─► ReadFile()
         │    │    │    │
         │    │    │    ├─► evaluateWithBackoffAndMetrics()
         │    │    │    │    │
         │    │    │    │    ├─► Check Circuit Breaker
         │    │    │    │    │    └─ If open: wait CircuitBreakerWait
         │    │    │    │    │
         │    │    │    │    ├─► Retry Loop (0 to MaxRetries)
         │    │    │    │    │    │
         │    │    │    │    │    ├─► evaluator.Evaluate()
         │    │    │    │    │    │    │
         │    │    │    │    │    │    ├─ OpenAI API call
         │    │    │    │    │    │    ├─ Cohere API call
         │    │    │    │    │    │    └─ etc.
         │    │    │    │    │    │
         │    │    │    │    │    ├─► On Success:
         │    │    │    │    │    │    ├─ metrics.SuccessCount++
         │    │    │    │    │    │    ├─ circuitBreaker.recordSuccess()
         │    │    │    │    │    │    └─ Return result
         │    │    │    │    │    │
         │    │    │    │    │    └─► On Error:
         │    │    │    │    │         │
         │    │    │    │    │         ├─ isRetryableError()?
         │    │    │    │    │         │   ├─ HTTPError 429? → Yes
         │    │    │    │    │         │   ├─ HTTPError 5xx? → Yes
         │    │    │    │    │         │   └─ HTTPError 4xx? → No
         │    │    │    │    │         │
         │    │    │    │    │         ├─ If retryable:
         │    │    │    │    │         │   ├─ metrics.RetryCount++
         │    │    │    │    │         │   ├─ Calculate backoff
         │    │    │    │    │         │   │   └─ baseMs * 2^attempt
         │    │    │    │    │         │   ├─ Add jitter (if enabled)
         │    │    │    │    │         │   │   └─ +0-25% random
         │    │    │    │    │         │   ├─ time.Sleep(backoff)
         │    │    │    │    │         │   └─ Retry
         │    │    │    │    │         │
         │    │    │    │    │         └─ If not retryable or max retries:
         │    │    │    │    │             ├─ metrics.FailureCount++
         │    │    │    │    │             ├─ circuitBreaker.recordFailure()
         │    │    │    │    │             │   └─ If >= CircuitBreakerMax:
         │    │    │    │    │             │       ├─ Open circuit
         │    │    │    │    │             │       └─ metrics.CircuitBreakerTrips++
         │    │    │    │    │             └─ Return error
         │    │    │    │    │
         │    │    │    │    └─► Release semaphore
         │    │    │    │         └─ <-sem
         │    │    │    │
         │    │    │    └─► Collect result
         │    │    │
         │    │    └─► Wait for all goroutines
         │    │         └─ wg.Wait()
         │    │
         │    └─► Sleep between batches
         │         └─ time.Sleep(RequestIntervalMs)
         │
         └─► Return all results + metrics
```

---

## 🔁 Estado del Circuit Breaker

```
┌─────────────┐
│   CLOSED    │ ← Initial State
│  (Normal)   │
└──────┬──────┘
       │
       │ Record Failure
       │ consecutiveFailures++
       │
       │ If consecutiveFailures >= CircuitBreakerMax
       ▼
┌─────────────┐
│    OPEN     │
│ (Blocking)  │
└──────┬──────┘
       │
       │ Wait CircuitBreakerWait seconds
       │
       │ time.Now() > openUntil
       ▼
┌─────────────┐
│  HALF-OPEN  │
│  (Testing)  │
└──────┬──────┘
       │
       ├─► Next request succeeds
       │   └─► CLOSED (consecutiveFailures = 0)
       │
       └─► Next request fails
           └─► OPEN (reset timer)
```

---

## 🎯 Retry Logic con Backoff Exponencial

```
Attempt 0 (Initial):
├─► Call API
├─► Success? → Return result
└─► Failure? → Is retryable?
     ├─ No → Return error immediately
     └─ Yes → Proceed to retry

Attempt 1 (First Retry):
├─► Wait: 1000ms * 2^0 = 1000ms
│    └─► + jitter (0-250ms) = 1000-1250ms
├─► Call API
├─► Success? → Return result
└─► Failure? → Continue

Attempt 2 (Second Retry):
├─► Wait: 1000ms * 2^1 = 2000ms
│    └─► + jitter (0-500ms) = 2000-2500ms
├─► Call API
├─► Success? → Return result
└─► Failure? → Continue

Attempt 3 (Third Retry):
├─► Wait: 1000ms * 2^2 = 4000ms
│    └─► + jitter (0-1000ms) = 4000-5000ms
├─► Call API
├─► Success? → Return result
└─► Failure? → Continue

Attempt 4 (Fourth Retry):
├─► Wait: 1000ms * 2^3 = 8000ms
│    └─► + jitter (0-2000ms) = 8000-10000ms
├─► Call API
├─► Success? → Return result
└─► Failure? → Continue

Attempt 5 (Fifth Retry - Last):
├─► Wait: 1000ms * 2^4 = 16000ms
│    └─► + jitter (0-4000ms) = 16000-20000ms
├─► Call API
├─► Success? → Return result
└─► Failure? → Return error (exhausted retries)

Total max wait: ~1 + 2 + 4 + 8 + 16 = ~31 seconds (+ jitter)
```

---

## 📊 Estructura de Datos

### AIAgentConfig
```
AIAgentConfig
├── Type: string                    (openai, cohere, copilot, etc.)
├── Enabled: bool                   (true/false)
├── Key: string                     (API key - use env var)
├── Model: string                   (gpt-4o-mini, command, etc.)
├── MaxTokens: int                  (1200)
├── Temperature: float64            (0.0 - 2.0)
├── BatchSize: int                  (files per batch)
├── RequestIntervalMs: int          (ms between batches)
├── MaxConcurrency: int             (max parallel evaluations)
├── MaxRetries: int                 (retry attempts)
├── BackoffInitialMs: int           (initial backoff ms)
├── BackoffJitter: bool             (add random variance)
├── CircuitBreakerMax: int          (failures before open)
└── CircuitBreakerWait: int         (wait seconds when open)
```

### EvaluationMetrics
```
EvaluationMetrics
├── TotalAttempts: int              (total eval attempts)
├── SuccessCount: int               (successful evals)
├── FailureCount: int               (failed evals)
├── RetryCount: int                 (total retries performed)
├── RateLimitCount: int             (429 errors encountered)
├── TotalLatencyMs: int64           (cumulative latency)
└── CircuitBreakerTrips: int        (times circuit opened)
```

### EvaluationResult
```
EvaluationResult
├── File: string                    (file path)
├── Score: int                      (0-100)
├── FactoresNoCumple: []string      (twelve-factor violations)
├── ProblemasConcurrencia: []string (goroutine/channel issues)
├── RecomendacionesRefactor: string (refactoring suggestions)
├── RecomendacionesComentarios: str (doc improvements)
├── Documentacion: string           (markdown guide)
└── EvaluacionFunciones: []FuncEval (per-function analysis)
     └── FuncionEvaluationResult
          ├── Funcion: string
          ├── Claridad: string       (Alta/Media/Baja)
          ├── Complejidad: string    (Alta/Media/Baja)
          ├── RiesgoConcurrencia: str (Alto/Medio/Bajo)
          └── Sugerencias: string
```

---

## 🔐 Flujo de Configuración

```
LoadConfig()
    │
    ├─► Read config/config_AIAgent.json
    │    └─ Parse JSON into AgentConfig struct
    │
    ├─► overrideAgentKeysWithEnv()
    │    │
    │    └─► For each agent:
    │         ├─ If Type == "openai"
    │         │   └─ Key = os.Getenv("OPENAI_API_KEY") || Key
    │         ├─ If Type == "cohere"
    │         │   └─ Key = os.Getenv("COHERE_API_KEY") || Key
    │         ├─ If Type == "copilot"
    │         │   └─ Key = os.Getenv("COPILOT_API_KEY") || Key
    │         └─ UseMockMotorAI = os.Getenv("USE_MOCK_MOTOR_AI") || UseMockMotorAI
    │
    ├─► applyGeneralEnvOverrides()
    │    │
    │    └─► Override if env var exists:
    │         ├─ TARGET_DIR
    │         ├─ GITHUB_TOKEN
    │         ├─ GITHUB_REPO
    │         ├─ BASE_BRANCH
    │         ├─ SONAR_HOST_URL
    │         ├─ SONAR_PROJECT_KEY
    │         ├─ SONAR_TOKEN
    │         └─ TEAMS_WEBHOOK_URL
    │
    ├─► Set defaults
    │    ├─ TargetDir = "./" if empty
    │    └─ BaseBranch = "main" if empty
    │
    └─► Return AgentConfig

Priority: Environment Variables > config_AIAgent.json
```

---

## 🌐 Interacción con GitHub

```
GitHub Repository
    │
    ├─► Push/PR Trigger
    │    └─► GitHub Actions Runner
    │         └─► Execute ai-agent-linux
    │              │
    │              ├─► Read code files
    │              ├─► Evaluate with AI
    │              └─► Generate report
    │
    └─► AI Agent (via GitHub API)
         │
         ├─► Create temp branch
         │    └─ Branch name: ai-agent-update-{PID}
         │
         ├─► Commit report
         │    └─ File: ARQUITECTURE_COMPLIANCE.md
         │
         ├─► Create/Update Pull Request
         │    ├─ Title: "AI Agent - Architecture Compliance Report"
         │    ├─ Body: Report summary + metadata
         │    └─ Base: BaseBranch (dev/main)
         │
         ├─► Comment on PR (if PR review workflow)
         │    ├─ Score summary
         │    ├─ Quick metrics
         │    └─ Full report (collapsible)
         │
         └─► Create Issue (if weekly audit + low score)
              ├─ Title: "⚠️ Architecture Compliance Alert"
              ├─ Body: Issues + recommendations
              └─ Labels: tech-debt, architecture, automated-audit
```

---

## 🧩 Componentes del Sistema

```
┌─────────────────────────────────────────────────────────────┐
│                        AI AGENT                              │
│                                                               │
│  ┌─────────────────────────────────────────────────────┐   │
│  │ cmd/ai-agent/main.go                                │   │
│  │  - Entry point                                       │   │
│  │  - Orchestration                                     │   │
│  │  - Execution metadata                                │   │
│  └─────────────────────────────────────────────────────┘   │
│                                                               │
│  ┌─────────────────────────────────────────────────────┐   │
│  │ internal/ai/                                         │   │
│  │  ├─ agent.go (core logic)                           │   │
│  │  │   ├─ CodeEvaluator interface                     │   │
│  │  │   ├─ EvaluateFilesGeneric()                      │   │
│  │  │   ├─ Circuit breaker                             │   │
│  │  │   ├─ Retry logic                                 │   │
│  │  │   └─ Metrics tracking                            │   │
│  │  ├─ openai_agent.go (OpenAI)                        │   │
│  │  ├─ cohere_agent.go (Cohere)                        │   │
│  │  ├─ copilot_agent.go (Copilot)                      │   │
│  │  ├─ anthropic_agent.go (Anthropic)                  │   │
│  │  ├─ gemini_agent.go (Gemini)                        │   │
│  │  └─ mistral_agent.go (Mistral)                      │   │
│  └─────────────────────────────────────────────────────┘   │
│                                                               │
│  ┌─────────────────────────────────────────────────────┐   │
│  │ internal/github/                                     │   │
│  │  └─ gh_client.go                                     │   │
│  │      ├─ CreateOrUpdateFileWithPR()                   │   │
│  │      ├─ Branch operations                            │   │
│  │      └─ PR management                                │   │
│  └─────────────────────────────────────────────────────┘   │
│                                                               │
│  ┌─────────────────────────────────────────────────────┐   │
│  │ internal/teams/                                      │   │
│  │  └─ webhook.go                                       │   │
│  │      └─ SendMessage()                                │   │
│  └─────────────────────────────────────────────────────┘   │
│                                                               │
│  ┌─────────────────────────────────────────────────────┐   │
│  │ internal/logger/                                     │   │
│  │  └─ logger.go                                        │   │
│  │      ├─ Info()                                       │   │
│  │      └─ Error()                                      │   │
│  └─────────────────────────────────────────────────────┘   │
│                                                               │
│  ┌─────────────────────────────────────────────────────┐   │
│  │ config/                                              │   │
│  │  └─ config_AIAgent.json                              │   │
│  │      └─ Agent configurations                         │   │
│  └─────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
```

---

## 📈 Métricas en Acción

```
Ejemplo de Ejecución:

Input:
- 50 archivos .go
- BatchSize: 2
- RequestIntervalMs: 2000ms
- MaxRetries: 5
- OpenAI API (gpt-4o-mini)

Ejecución:
├─ Batch 1 (2 files): 3s eval + 2s wait
├─ Batch 2 (2 files): 3s eval + 2s wait
├─ ...
├─ Batch 24 (2 files): 3s eval + 2s wait
└─ Batch 25 (2 files): 3s eval (last batch, no wait)

Total Time: ~125 segundos (~2 minutos)

Métricas Reportadas:
[Metrics] Agent: openai | Success: 48 | Failures: 2 | Retries: 5 | 
          RateLimits(429): 2 | AvgLatency: 2500ms | CircuitBreaks: 0

Análisis:
- 96% success rate (48/50) ✅
- 2 failures retried successfully
- 2 rate limits encountered but recovered
- No circuit breaker trips
- Avg 2.5s per file evaluation
```

---

## 🎨 Decisión de Agent Type

```
                    ┌──────────────┐
                    │   Agent      │
                    │   Factory    │
                    └──────┬───────┘
                           │
                           │ Type?
              ┌────────────┼────────────┐
              │            │            │
       ┌──────▼────┐ ┌────▼────┐ ┌────▼────┐
       │  "openai" │ │"cohere" │ │"copilot"│
       └──────┬────┘ └────┬────┘ └────┬────┘
              │            │            │
     ┌────────▼──────┐ ┌──▼────┐ ┌────▼──────┐
     │OpenAIEvaluator│ │Cohere │ │ Copilot   │
     │               │ │Eval..│ │ Evaluator │
     └────────┬──────┘ └──┬────┘ └────┬──────┘
              │            │            │
              │            │            │
              └────────────┴────────────┘
                           │
                     implements
                           │
                    ┌──────▼──────┐
                    │CodeEvaluator│
                    │ interface   │
                    └─────────────┘
                           │
                  Evaluate(ctx, file, code)
                           │
                    ┌──────▼──────────┐
                    │EvaluationResult │
                    └─────────────────┘
```

---

Este documento proporciona una visión completa de cómo funcionan los diferentes componentes del sistema y cómo interactúan entre sí. Úsalo como referencia rápida para entender el flujo de datos y la arquitectura.
