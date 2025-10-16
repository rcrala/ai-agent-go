
# ai-agent-go

Agente de revisión de código en Go con arquitectura extensible de agentes IA (OpenAI, Copilot, etc.).

## 📚 Documentación

### 🚀 Inicio Rápido
- **[Quick Start Guide](docs/quick-start.md)** - Integración en 5 minutos con tu proyecto de GitHub

### 📖 Guías Completas
- **[Guía de Integración con GitHub](docs/github-integration-guide.md)** - Configuración completa y uso del binario en proyectos
- **[Workflows de Ejemplo](examples/workflows/)** - Plantillas de GitHub Actions listas para usar
  - PR Review automático
  - Auditoría semanal programada
  - Verificación obligatoria (bloquea merge)
  - Pipeline multi-etapa

### 🔧 Documentación Técnica
- **[Manejo de Errores HTTP](docs/http-error-handling.md)** - Circuit breaker, retry logic, y rate limiting
- **[Twelve-Factor App Compliance](docs/twelve-factor-compliance.md)** - Análisis de cumplimiento arquitectónico (Score: 92/100)
- **[GitHub Actions Permissions](docs/actions-permissions.md)** - Configuración de permisos para CI/CD

## Arquitectura
- **cmd/ai-agent/main.go**: Punto de entrada. Carga configuración, inicializa clientes, ejecuta todos los agentes IA habilitados (vía interfaz genérica), análisis SonarQube y notificaciones.
- **internal/ai/agent.go**: Define la interfaz `CodeEvaluator`, fábrica de agentes, carga de config y evaluación genérica de archivos.
- **internal/ai/openai_agent.go**: Implementa el agente OpenAI.
- **internal/ai/copilot_agent.go**: Implementa el agente Copilot.
- **config/config_AIAgent.json**: Configuración principal. Soporta colección de agentes con sus propios parámetros.

## Configuración de agentes
Ejemplo de `config/config_AIAgent.json`:
```json
{
	"Agents": [
		{
			"Type": "openai",
			"Enabled": true,
			"Key": "",
			"Model": "gpt-4o-mini",
			"MaxTokens": 1200,
			"Temperature": 0.0,
			"BatchSize": 3
		},
		{
			"Type": "copilot",
			"Enabled": false,
			"Key": "",
			"Model": "copilot-2025",
			"MaxTokens": 1200,
			"Temperature": 0.0,
			"BatchSize": 3
		}
	],
	"TargetDir": "./",
	"GitHubToken": "",
	"GitHubRepo": "rcrala/ai-agent-go",
	"BaseBranch": "dev",
	"SonarHostURL": "",
	"SonarProjectKey": "",
	"SonarToken": "",
	"TeamsWebhookURL": "",
	"RunSonar": false,
	"SendTeamsNotification": true
}
```

## Uso de agentes IA
- Cada agente implementa la interfaz `CodeEvaluator`.
- Puedes agregar nuevos agentes creando un archivo y registrando el tipo en la fábrica de `agent.go`.
- La clave de cada agente puede configurarse por variable de entorno (`OPENAI_API_KEY`, `COPILOT_API_KEY`, `COHERE_API_KEY`, etc.).

## ✨ Nuevas características avanzadas

### Manejo robusto de errores HTTP y rate limits (429)

El sistema incluye:
- **Detección estructurada de errores HTTP**: Tipo `HTTPError` con status code para lógica precisa de retry
- **Reintentos con backoff exponencial**: Configurables vía `MaxRetries` y `BackoffInitialMs`
- **Jitter aleatorio**: Evita "thundering herd" añadiendo varianza al backoff (configurable con `BackoffJitter`)
- **Circuit breaker**: Abre el circuito tras N fallos consecutivos para proteger APIs sobrecargadas
- **Métricas detalladas**: Tracking de intentos, éxitos, fallos, retries, 429s, latencia promedio y circuit breaker trips
- **Límite de concurrencia**: `MaxConcurrency` controla el número máximo de evaluaciones simultáneas

### Configuración avanzada por agente

```json
{
  "Type": "openai",
  "Enabled": true,
  "BatchSize": 2,
  "RequestIntervalMs": 2000,
  "MaxConcurrency": 1,
  "MaxRetries": 5,
  "BackoffInitialMs": 1000,
  "BackoffJitter": true,
  "CircuitBreakerMax": 3,
  "CircuitBreakerWait": 30
}
```

**Parámetros:**
- `BatchSize`: Archivos por batch (controla ráfaga de requests)
- `RequestIntervalMs`: Milisegundos de espera entre batches
- `MaxConcurrency`: Máximo de evaluaciones concurrentes (default: BatchSize)
- `MaxRetries`: Número de reintentos en errores retryables (429, 5xx)
- `BackoffInitialMs`: Backoff inicial en ms (se duplica exponencialmente)
- `BackoffJitter`: Añade hasta 25% de jitter aleatorio al backoff
- `CircuitBreakerMax`: Fallos consecutivos antes de abrir circuito (0=deshabilitado)
- `CircuitBreakerWait`: Segundos que el circuito permanece abierto

Ver [docs/http-error-handling.md](docs/http-error-handling.md) para más detalles.

## Ejecución
- **Build**: `go build -o ai-agent-linux ./cmd/ai-agent`
- **Local**: Exporta las variables de entorno necesarias y ejecuta el binario.
- **GitHub Action**: Usa los secretos para cada agente que quieras habilitar.

## Ejemplo: Habilitar Copilot
1. Agrega un bloque en `Agents` con `"Type": "copilot"` y `"Enabled": true`.
2. Exporta la variable de entorno `COPILOT_API_KEY`.
3. Ejecuta el agente normalmente.

## Mock mode for testing
You can run the agent without calling external LLM services by enabling mock mode.

- Per-agent mock (OpenAI only): set `OPENAI_MOCK=true` in the environment.
- Global mock for all agents: set `USE_MOCK_MOTOR_AI=true` in the environment. This overrides per-agent mock flags and forces mock responses across agents.

PowerShell example:
```powershell
$env:USE_MOCK_MOTOR_AI = 'true'
go run ./cmd/ai-agent
```
