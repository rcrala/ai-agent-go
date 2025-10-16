# HTTP Error Handling & Retry Logic

## Resumen

El sistema ahora incluye detección robusta de errores HTTP (especialmente 429 "Too Many Requests") y lógica de reintentos con backoff exponencial para mitigar problemas de límites de tasa (rate limits) de los proveedores de AI.

## Características implementadas

### 1. Tipo HTTPError estructurado

Se añadió el tipo `HTTPError` en `internal/ai/agent.go`:

```go
type HTTPError struct {
    StatusCode int
    Message    string
}
```

Este tipo envuelve errores HTTP con el código de estado real del proveedor, permitiendo lógica de retry basada en el status code en lugar de inspección de strings.

### 2. Detección de errores retryables

La función `isRetryableError()` ahora:
- Verifica primero si el error es un `*HTTPError` estructurado.
- Si es HTTP 429 (rate limit) o 5xx (errores de servidor), lo marca como retryable.
- Como fallback, inspecciona el texto del error buscando patrones comunes: "429", "too many requests", "rate limit", "quota", "exceeded", "timeout", "temporary", "connection reset".

### 3. Propagación de status code desde agentes

#### OpenAI Agent
- `evaluateCodeReal` ahora extrae el status code del error de OpenAI usando `extractStatusCodeFromOpenAIError()`.
- Esta función intenta:
  1. Detectar si el error implementa `HTTPStatusCode() int` (interfaz del SDK).
  2. Parsear el string del error buscando "status code: 429" y patrones similares.
  3. Detectar menciones de códigos como "429" o "5xx" en el texto.
- Si encuentra un status code, devuelve `&HTTPError{StatusCode: code, Message: ...}`.

#### Cohere Agent
- En `evaluateCohereCode`, cuando `resp.StatusCode` es no-2xx, ahora devuelve directamente:
  ```go
  &HTTPError{StatusCode: resp.StatusCode, Message: "..."}
  ```

### 4. Configuración de retry/backoff

Se añadieron campos opcionales a `AIAgentConfig`:

```json
{
  "Type": "openai",
  "MaxRetries": 3,
  "BackoffInitialMs": 500,
  "MaxConcurrency": 2,
  "BatchSize": 3,
  "RequestIntervalMs": 1000
}
```

- **MaxRetries**: número de reintentos adicionales (0 = sin retries, 3 = hasta 4 intentos totales).
- **BackoffInitialMs**: backoff inicial en milisegundos. Se duplica exponencialmente en cada retry (500ms → 1s → 2s → 4s).
- **MaxConcurrency**: máximo número de evaluaciones concurrentes (por defecto = BatchSize).

### 5. Lógica de retry con backoff exponencial

La función `evaluateWithBackoff()` en `internal/ai/agent.go`:
1. Intenta la evaluación.
2. Si falla con error retryable (429, 5xx, timeout), espera `BackoffInitialMs * (2^attempt)` y reintenta.
3. Si el error no es retryable (400, 401, JSON parse error), falla inmediatamente sin reintentos.
4. Si se agotan los reintentos, devuelve el último error.

### 6. Límite de concurrencia

`EvaluateFilesGeneric` ahora usa un semáforo para limitar el número de goroutines concurrentes evaluando archivos, respetando `MaxConcurrency`. Esto reduce la probabilidad de saturar el endpoint con ráfagas de requests.

## Flujo completo para error 429

1. Usuario ejecuta `go run ./cmd/ai-agent` con config que tiene `MaxRetries=3`, `BackoffInitialMs=500`.
2. El sistema procesa archivos en batches de tamaño `BatchSize` (ej. 3).
3. Dentro de cada batch, lanza hasta `MaxConcurrency` goroutines (ej. 2).
4. Si OpenAI/Cohere devuelve 429:
   - El agente envuelve el error en `&HTTPError{StatusCode: 429, ...}`.
   - `evaluateWithBackoff` detecta que es retryable.
   - Espera 500ms y reintenta (intento 2).
   - Si falla otra vez, espera 1000ms (intento 3).
   - Si falla, espera 2000ms (intento 4).
   - Si todos los reintentos fallan, devuelve el error final.
5. Entre batches, el sistema espera `RequestIntervalMs` (ej. 1000ms) antes de procesar el siguiente batch.

## Tests unitarios

Se añadió `internal/ai/agent_test.go` con:
- `TestHTTPError`: verifica que `HTTPError` implemente `Error()` correctamente.
- `TestIsRetryableError`: valida detección de errores retryables (429, 5xx, strings, etc.).
- `TestEvaluateWithBackoff`: valida reintentos, backoff exponencial, detección de no-retryables, y exhaustión de reintentos.

Ejecutar:
```powershell
go test -v ./internal/ai
```

Todos los tests pasan ✅.

## Nuevas características avanzadas ✨

### 1. **Métricas de observabilidad**

El sistema ahora recopila y registra métricas detalladas por cada agente:

```go
type EvaluationMetrics struct {
    TotalAttempts      int   // Total de intentos de evaluación
    SuccessCount       int   // Evaluaciones exitosas
    FailureCount       int   // Evaluaciones fallidas
    RetryCount         int   // Número de reintentos ejecutados
    RateLimitCount     int   // Errores HTTP 429 detectados
    TotalLatencyMs     int64 // Latencia total acumulada
    CircuitBreakerTrips int  // Veces que se abrió el circuit breaker
}
```

Al finalizar cada lote de evaluaciones, se imprime un resumen:
```
[Metrics] Agent: openai | Success: 8 | Failures: 2 | Retries: 3 | RateLimits(429): 2 | AvgLatency: 1450ms | CircuitBreaks: 0
```

### 2. **Circuit breaker**

Implementación de circuit breaker para proteger contra fallos consecutivos:

**Configuración:**
```json
{
  "CircuitBreakerMax": 3,
  "CircuitBreakerWait": 30
}
```

- `CircuitBreakerMax`: número de fallos consecutivos antes de abrir el circuito (0=deshabilitado)
- `CircuitBreakerWait`: segundos que el circuito permanece abierto antes de intentar nuevamente

**Funcionamiento:**
1. Tras N fallos consecutivos, el circuit breaker se "abre"
2. Durante el período de espera, todas las evaluaciones fallan inmediatamente con error "circuit breaker is open"
3. Después del período de espera, el circuito entra en estado "half-open" y permite un intento
4. Si el intento tiene éxito, el circuito se "cierra" y todo vuelve a la normalidad
5. Si falla, el circuito se vuelve a abrir

**Beneficios:**
- Evita saturar APIs que ya están sobrecargadas
- Reduce latencia al no intentar llamadas que probablemente fallarán
- Permite que el servicio se recupere antes de recibir más requests

### 3. **Jitter en backoff exponencial**

Para evitar el problema de "thundering herd" (múltiples instancias reintentando simultáneamente), se añade jitter aleatorio al backoff:

**Configuración:**
```json
{
  "BackoffJitter": true
}
```

**Funcionamiento:**
- Sin jitter: backoff es exactamente 500ms → 1000ms → 2000ms → 4000ms
- Con jitter: se añade hasta 25% aleatorio: 500-625ms → 1000-1250ms → 2000-2500ms → 4000-5000ms

Esto dispersa los reintentos en el tiempo, reduciendo picos de carga en el servidor.

### 4. **Preparación para otros agentes**

Todos los agentes (Anthropic, Gemini, Mistral, YouCom, Copilot) ahora incluyen comentarios indicando cómo implementar detección de HTTPError cuando se integren sus APIs reales:

```go
// When implemented, wrap HTTP errors with HTTPError for proper retry logic:
// if resp.StatusCode >= 400 {
//   return nil, &HTTPError{StatusCode: resp.StatusCode, Message: "..."}
// }
```

## Próximos pasos opcionales

1. **Dashboard de métricas**: Exportar métricas a Prometheus/Grafana para visualización en tiempo real.
2. **Rate limiting adaptativo**: Ajustar `BatchSize` y `RequestIntervalMs` dinámicamente según la tasa de 429s.
3. **Priorización de archivos**: Evaluar primero archivos críticos cuando el circuit breaker está cerca de abrirse.
4. **Alertas**: Notificar a Teams/Slack cuando se detectan patrones de error (ej. >50% de 429s).
5. **Cache de evaluaciones**: Cachear resultados para archivos que no han cambiado.

## Tests unitarios

Se añadieron tests completos para todas las nuevas funcionalidades:

- `TestCircuitBreaker`: valida apertura/cierre del circuito y reset tras éxito
- `TestEvaluationMetrics`: verifica conteo preciso de intentos, éxitos, fallos, retries y 429s
- `TestBackoffJitter`: confirma que el jitter añade varianza al backoff

Ejecutar todos los tests:
```powershell
go test -v ./internal/ai
```

**Resultado:** ✅ Todos los tests pasan (6/6 test suites, 23 casos)

## Referencias

- OpenAI error codes: https://platform.openai.com/docs/guides/error-codes/api-errors
- Cohere API docs: https://docs.cohere.com/reference/overview
- Exponential backoff: https://en.wikipedia.org/wiki/Exponential_backoff
