# Guía de Integración con GitHub
## Configuración y Uso del Agente AI Compilado

**Versión:** 1.0  
**Fecha:** Octubre 16, 2025  
**Audiencia:** DevOps Engineers, Desarrolladores, Arquitectos

---

## 📋 Tabla de Contenidos

1. [Resumen Ejecutivo](#resumen-ejecutivo)
2. [Requisitos Previos](#requisitos-previos)
3. [Arquitectura de Integración](#arquitectura-de-integración)
4. [Configuración Inicial](#configuración-inicial)
5. [Uso del Binario](#uso-del-binario)
6. [Integración con GitHub Actions](#integración-con-github-actions)
7. [Configuración Avanzada](#configuración-avanzada)
8. [Troubleshooting](#troubleshooting)
9. [Mejores Prácticas](#mejores-prácticas)
10. [Ejemplos de Uso](#ejemplos-de-uso)

---

## Resumen Ejecutivo

El **AI Agent** es un binario compilado en Go que evalúa la calidad y cumplimiento arquitectónico del código en repositorios de GitHub. Este agente:

- ✅ **Evalúa código Go/Python** contra los principios Twelve-Factor App
- ✅ **Genera reportes automáticos** en Markdown con scores y recomendaciones
- ✅ **Crea Pull Requests** con los análisis en `ARQUITECTURE_COMPLIANCE.md`
- ✅ **Integra múltiples motores AI** (OpenAI, Copilot, Cohere, Anthropic, Gemini, Mistral)
- ✅ **Incluye protección anti rate-limit** (circuit breaker, backoff exponencial, jitter)
- ✅ **Notifica a Microsoft Teams** (opcional)
- ✅ **Compatible con SonarQube** (opcional)

**Casos de Uso:**
- Validación automática de PRs antes de merge
- Auditorías periódicas de calidad de código
- Evaluación de cumplimiento arquitectónico en CI/CD
- Reportes de deuda técnica y concurrencia

---

## Requisitos Previos

### 1. Binario Compilado

Obtener el binario `ai-agent-linux` (o `ai-agent.exe` para Windows):

```bash
# Opción A: Descargar release desde GitHub
wget https://github.com/rcrala/ai-agent-go/releases/latest/download/ai-agent-linux
chmod +x ai-agent-linux

# Opción B: Compilar desde fuente
git clone https://github.com/rcrala/ai-agent-go.git
cd ai-agent-go
go build -o ai-agent-linux ./cmd/ai-agent
```

### 2. Claves de API

Al menos una clave de API de un proveedor AI:

| Proveedor | Variable de Entorno | Donde Obtenerla |
|-----------|---------------------|-----------------|
| OpenAI | `OPENAI_API_KEY` | https://platform.openai.com/api-keys |
| GitHub Copilot | `COPILOT_API_KEY` | https://github.com/features/copilot |
| Cohere | `COHERE_API_KEY` | https://dashboard.cohere.ai/api-keys |
| Anthropic Claude | `ANTHROPIC_API_KEY` | https://console.anthropic.com/ |
| Google Gemini | `GEMINI_API_KEY` | https://makersuite.google.com/app/apikey |
| Mistral AI | `MISTRAL_API_KEY` | https://console.mistral.ai/ |

### 3. Token de GitHub

```bash
# Crear Personal Access Token con permisos:
# - repo (full control)
# - workflow (update GitHub Actions)

# En GitHub:
Settings → Developer settings → Personal access tokens → 
Generate new token (classic) → Seleccionar "repo" y "workflow"
```

### 4. Archivo de Configuración

Descargar plantilla de configuración:

```bash
# config/config_AIAgent.json
curl -o config_AIAgent.json https://raw.githubusercontent.com/rcrala/ai-agent-go/main/config/config_AIAgent.json
```

---

## Arquitectura de Integración

```
┌─────────────────────────────────────────────────────────────────┐
│                       GitHub Repository                          │
│  ┌────────────┐      ┌─────────────┐      ┌─────────────┐      │
│  │   main     │◄─────│     dev     │◄─────│ feature-123 │      │
│  └────────────┘      └─────────────┘      └─────────────┘      │
│         ▲                                                        │
│         │ PR created                                             │
│         │ with report                                            │
└─────────┼──────────────────────────────────────────────────────┘
          │
          │
┌─────────┼──────────────────────────────────────────────────────┐
│         │              GitHub Actions Runner                     │
│  ┌──────▼──────────────────────────────────────────────────┐   │
│  │  1. Trigger (on push/PR to dev/main)                     │   │
│  │  2. Checkout code                                        │   │
│  │  3. Download ai-agent-linux binary                       │   │
│  │  4. Set environment variables (secrets)                  │   │
│  │  5. Execute: ./ai-agent-linux                            │   │
│  └──────┬───────────────────────────────────────────────────┘   │
│         │                                                        │
└─────────┼──────────────────────────────────────────────────────┘
          │
          ▼
┌─────────────────────────────────────────────────────────────────┐
│                      AI Agent Binary                             │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │ LoadConfig()                                             │   │
│  │   ├─ config_AIAgent.json (defaults)                      │   │
│  │   └─ Environment variables (override)                    │   │
│  ├─────────────────────────────────────────────────────────┤   │
│  │ ScanFiles() → .go, .py files                             │   │
│  ├─────────────────────────────────────────────────────────┤   │
│  │ EvaluateFilesGeneric()                                   │   │
│  │   ├─ Batch processing (BatchSize: 2)                     │   │
│  │   ├─ Rate limiting (RequestIntervalMs: 2000)            │   │
│  │   ├─ Retry logic (MaxRetries: 5, backoff + jitter)      │   │
│  │   └─ Circuit breaker (CircuitBreakerMax: 3)             │   │
│  ├─────────────────────────────────────────────────────────┤   │
│  │ GenerateMarkdown() → ARQUITECTURE_COMPLIANCE.md          │   │
│  ├─────────────────────────────────────────────────────────┤   │
│  │ CreateOrUpdateFileWithPR()                               │   │
│  │   └─ Push to temp branch → Create PR                    │   │
│  └─────────────────────────────────────────────────────────┘   │
└────────┬──────────────────────────────────────┬─────────────────┘
         │                                      │
         │ API Calls                            │ Notifications
         ▼                                      ▼
┌──────────────────────┐            ┌──────────────────────┐
│   AI Providers       │            │  Microsoft Teams     │
│  • OpenAI API        │            │  (webhook)           │
│  • Cohere API        │            │  "Report generated   │
│  • Copilot API       │            │   for branch dev"    │
│  • Anthropic API     │            └──────────────────────┘
│  • Gemini API        │
│  • Mistral API       │
└──────────────────────┘
```

### Flujo de Ejecución

1. **Trigger**: Push o PR a branch `dev` o `main`
2. **GitHub Actions**: Descarga y ejecuta el binario
3. **Carga Config**: Lee `config_AIAgent.json` + variables de entorno
4. **Escaneo**: Encuentra archivos `.go` y `.py` en el repositorio
5. **Evaluación**: Envía código a motor AI (OpenAI, Cohere, etc.)
6. **Generación**: Crea reporte Markdown con scores y recomendaciones
7. **Publicación**: Crea branch temporal + PR con el reporte
8. **Notificación**: Envía mensaje a Teams (opcional)

---

## Configuración Inicial

### Paso 1: Crear Estructura de Configuración

En tu repositorio de GitHub, crear la siguiente estructura:

```bash
your-project/
├── .github/
│   └── workflows/
│       └── ai-agent.yml          # GitHub Actions workflow
├── config/
│   └── config_AIAgent.json       # Configuración del agente
└── [tu código fuente]
```

### Paso 2: Configurar `config_AIAgent.json`

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
      "BatchSize": 2,
      "RequestIntervalMs": 2000,
      "MaxConcurrency": 1,
      "MaxRetries": 5,
      "BackoffInitialMs": 1000,
      "BackoffJitter": true,
      "CircuitBreakerMax": 3,
      "CircuitBreakerWait": 30,
      "UseMockMotorAI": false
    },
    {
      "Type": "cohere",
      "Enabled": false,
      "Key": "",
      "Model": "command",
      "MaxTokens": 1200,
      "Temperature": 0.0,
      "BatchSize": 3,
      "RequestIntervalMs": 1500
    }
  ],
  "TargetDir": "./",
  "GitHubToken": "",
  "GitHubRepo": "your-org/your-repo",
  "BaseBranch": "dev",
  "SonarHostURL": "",
  "SonarProjectKey": "",
  "SonarToken": "",
  "TeamsWebhookURL": "",
  "RunSonar": false,
  "SendTeamsNotification": true
}
```

**Campos Importantes:**

| Campo | Descripción | Ejemplo |
|-------|-------------|---------|
| `Type` | Motor AI a usar | `openai`, `cohere`, `copilot` |
| `Enabled` | Activar/desactivar agente | `true` / `false` |
| `Key` | API key (dejar vacío, usar env var) | `""` |
| `Model` | Modelo específico del proveedor | `gpt-4o-mini`, `command` |
| `BatchSize` | Archivos procesados por lote | `2` (recomendado para evitar rate limits) |
| `RequestIntervalMs` | Pausa entre lotes (ms) | `2000` (2 segundos) |
| `MaxRetries` | Reintentos en caso de error | `5` |
| `BackoffJitter` | Añadir aleatoriedad al backoff | `true` (recomendado) |
| `CircuitBreakerMax` | Fallos antes de abrir circuito | `3` |
| `GitHubRepo` | Repositorio destino | `owner/repo-name` |
| `BaseBranch` | Branch base para PRs | `dev` o `main` |

### Paso 3: Configurar GitHub Secrets

En tu repositorio de GitHub:

```
Settings → Secrets and variables → Actions → New repository secret
```

**Secrets Requeridos:**

```bash
# API Keys (al menos una)
OPENAI_API_KEY=sk-proj-...
COHERE_API_KEY=...
COPILOT_API_KEY=...

# GitHub
GITHUB_TOKEN=ghp_...  # O usar el automático ${{ secrets.GITHUB_TOKEN }}

# Opcional: Teams
TEAMS_WEBHOOK_URL=https://outlook.office.com/webhook/...

# Opcional: SonarQube
SONAR_TOKEN=...
SONAR_HOST_URL=https://sonarcloud.io
SONAR_PROJECT_KEY=your-project-key
```

### Paso 4: Crear GitHub Actions Workflow

Crear `.github/workflows/ai-agent.yml`:

```yaml
name: AI Agent Code Review

on:
  push:
    branches: [main, dev]
  pull_request:
    branches: [main, dev]

jobs:
  ai-review:
    runs-on: ubuntu-latest
    
    permissions:
      contents: write
      pull-requests: write
      
    steps:
      - name: Checkout code
        uses: actions/checkout@v4
        with:
          fetch-depth: 0
      
      - name: Download AI Agent binary
        run: |
          # Opción A: Desde release de tu fork
          wget https://github.com/rcrala/ai-agent-go/releases/latest/download/ai-agent-linux
          
          # Opción B: Compilar desde fuente
          # git clone https://github.com/rcrala/ai-agent-go.git
          # cd ai-agent-go
          # go build -o ../ai-agent-linux ./cmd/ai-agent
          # cd ..
          
          chmod +x ai-agent-linux
      
      - name: Run AI Agent
        env:
          OPENAI_API_KEY: ${{ secrets.OPENAI_API_KEY }}
          COHERE_API_KEY: ${{ secrets.COHERE_API_KEY }}
          COPILOT_API_KEY: ${{ secrets.COPILOT_API_KEY }}
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
          TEAMS_WEBHOOK_URL: ${{ secrets.TEAMS_WEBHOOK_URL }}
          GITHUB_REPO: ${{ github.repository }}
          BASE_BRANCH: ${{ github.base_ref || 'dev' }}
          TARGET_DIR: ./
        run: |
          ./ai-agent-linux
      
      - name: Upload report artifact
        if: always()
        uses: actions/upload-artifact@v4
        with:
          name: architecture-compliance-report
          path: ARQUITECTURE_COMPLIANCE.md
          retention-days: 30
```

---

## Uso del Binario

### Ejecución Local (Testing)

```bash
# 1. Configurar variables de entorno
export OPENAI_API_KEY="sk-proj-..."
export GITHUB_TOKEN="ghp_..."
export GITHUB_REPO="owner/repo-name"
export BASE_BRANCH="dev"
export TARGET_DIR="./"

# 2. Ejecutar en modo mock (sin consumir API)
export USE_MOCK_MOTOR_AI=true
./ai-agent-linux

# 3. Ejecutar en modo real
export USE_MOCK_MOTOR_AI=false
./ai-agent-linux

# 4. Verificar el reporte generado
cat ARQUITECTURE_COMPLIANCE.md
```

### Ejecución en CI/CD

```bash
# En GitHub Actions (ver workflow arriba)
# En GitLab CI
# En Jenkins
# En Azure DevOps

# El binario lee automáticamente las variables de entorno
./ai-agent-linux
```

### Parámetros Configurables vía Variables de Entorno

Todas estas variables **sobrescriben** los valores en `config_AIAgent.json`:

```bash
# API Keys
OPENAI_API_KEY=sk-...
COPILOT_API_KEY=...
COHERE_API_KEY=...

# GitHub
GITHUB_TOKEN=ghp_...
GITHUB_REPO=owner/repo-name
BASE_BRANCH=main

# Directorios
TARGET_DIR=./src

# SonarQube (opcional)
SONAR_TOKEN=...
SONAR_HOST_URL=https://sonarcloud.io
SONAR_PROJECT_KEY=my-project

# Teams (opcional)
TEAMS_WEBHOOK_URL=https://outlook.office.com/webhook/...

# Testing
USE_MOCK_MOTOR_AI=true  # Modo mock global (no consume API)
OPENAI_MOCK=true        # Modo mock solo para OpenAI
```

---

## Integración con GitHub Actions

### Estrategia 1: Evaluación en PRs (Recomendada)

Ejecutar el agente cada vez que se crea o actualiza un PR:

```yaml
name: PR Code Review

on:
  pull_request:
    types: [opened, synchronize, reopened]
    branches: [main, dev]

jobs:
  ai-review:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Download AI Agent
        run: |
          wget https://github.com/rcrala/ai-agent-go/releases/latest/download/ai-agent-linux
          chmod +x ai-agent-linux
      
      - name: Run AI Review
        env:
          OPENAI_API_KEY: ${{ secrets.OPENAI_API_KEY }}
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
          GITHUB_REPO: ${{ github.repository }}
          BASE_BRANCH: ${{ github.base_ref }}
        run: ./ai-agent-linux
      
      - name: Comment on PR
        uses: actions/github-script@v7
        with:
          script: |
            const fs = require('fs');
            const report = fs.readFileSync('ARQUITECTURE_COMPLIANCE.md', 'utf8');
            
            github.rest.issues.createComment({
              issue_number: context.issue.number,
              owner: context.repo.owner,
              repo: context.repo.repo,
              body: `## 🤖 AI Agent Review\n\n${report.substring(0, 65000)}`
            });
```

### Estrategia 2: Auditoría Programada (Semanal)

```yaml
name: Weekly Architecture Audit

on:
  schedule:
    - cron: '0 9 * * 1'  # Lunes 9am UTC
  workflow_dispatch:      # Manual trigger

jobs:
  audit:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Download AI Agent
        run: |
          wget https://github.com/rcrala/ai-agent-go/releases/latest/download/ai-agent-linux
          chmod +x ai-agent-linux
      
      - name: Run Weekly Audit
        env:
          OPENAI_API_KEY: ${{ secrets.OPENAI_API_KEY }}
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
          GITHUB_REPO: ${{ github.repository }}
          BASE_BRANCH: main
          TEAMS_WEBHOOK_URL: ${{ secrets.TEAMS_WEBHOOK_URL }}
        run: ./ai-agent-linux
      
      - name: Create Issue if Low Score
        uses: actions/github-script@v7
        with:
          script: |
            const fs = require('fs');
            const report = fs.readFileSync('ARQUITECTURE_COMPLIANCE.md', 'utf8');
            
            // Parse score (simple regex)
            const scoreMatch = report.match(/Score:\*\*\s*(\d+)/);
            if (scoreMatch && parseInt(scoreMatch[1]) < 70) {
              github.rest.issues.create({
                owner: context.repo.owner,
                repo: context.repo.repo,
                title: '⚠️ Architecture Compliance Below 70%',
                body: `Weekly audit shows low compliance score.\n\n${report}`,
                labels: ['tech-debt', 'architecture']
              });
            }
```

### Estrategia 3: Evaluación Pre-merge (Requerida)

Bloquear merge si el score es bajo:

```yaml
name: Required Architecture Check

on:
  pull_request:
    branches: [main]

jobs:
  check:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Download AI Agent
        run: |
          wget https://github.com/rcrala/ai-agent-go/releases/latest/download/ai-agent-linux
          chmod +x ai-agent-linux
      
      - name: Run AI Review
        env:
          OPENAI_API_KEY: ${{ secrets.OPENAI_API_KEY }}
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
          GITHUB_REPO: ${{ github.repository }}
          BASE_BRANCH: ${{ github.base_ref }}
        run: ./ai-agent-linux
      
      - name: Check Minimum Score
        run: |
          score=$(grep -oP 'Score:\*\*\s*\K\d+' ARQUITECTURE_COMPLIANCE.md | head -1)
          echo "Architecture Score: $score"
          
          if [ "$score" -lt 70 ]; then
            echo "❌ Score $score is below minimum (70). Please fix issues before merging."
            exit 1
          else
            echo "✅ Score $score meets minimum requirements."
          fi
```

En GitHub Settings:
```
Settings → Branches → Branch protection rules → main
☑ Require status checks to pass before merging
  ☑ Required Architecture Check
```

---

## Configuración Avanzada

### 1. Múltiples Agentes AI (Redundancia)

Usar varios motores AI para comparar resultados:

```json
{
  "Agents": [
    {
      "Type": "openai",
      "Enabled": true,
      "Model": "gpt-4o-mini",
      "BatchSize": 2,
      "RequestIntervalMs": 2000
    },
    {
      "Type": "cohere",
      "Enabled": true,
      "Model": "command",
      "BatchSize": 3,
      "RequestIntervalMs": 1500
    }
  ]
}
```

El reporte incluirá evaluaciones de **ambos agentes** para comparación.

### 2. Rate Limiting Agresivo (APIs con Cuota Baja)

```json
{
  "Agents": [
    {
      "Type": "openai",
      "Enabled": true,
      "BatchSize": 1,              // 1 archivo a la vez
      "RequestIntervalMs": 5000,   // 5 segundos entre archivos
      "MaxConcurrency": 1,         // 1 llamada concurrente
      "MaxRetries": 10,            // 10 reintentos
      "BackoffInitialMs": 2000,    // 2s inicial
      "CircuitBreakerMax": 5,      // Abrir tras 5 fallos
      "CircuitBreakerWait": 60     // Esperar 60s antes de reintentar
    }
  ]
}
```

### 3. Evaluación Selectiva (Solo Archivos Modificados)

Modificar el workflow para evaluar solo cambios del PR:

```yaml
- name: Get changed files
  id: changed-files
  uses: tj-actions/changed-files@v40
  with:
    files: |
      **/*.go
      **/*.py

- name: Create filtered config
  run: |
    echo "${{ steps.changed-files.outputs.all_changed_files }}" > changed_files.txt
    # Modificar TARGET_DIR para evaluar solo archivos cambiados

- name: Run AI Review (changed files only)
  env:
    OPENAI_API_KEY: ${{ secrets.OPENAI_API_KEY }}
  run: ./ai-agent-linux
```

### 4. Integración con SonarQube

```json
{
  "RunSonar": true,
  "SonarHostURL": "https://sonarcloud.io",
  "SonarProjectKey": "your-org_your-repo",
  "SonarToken": ""
}
```

```bash
# Variables de entorno
SONAR_TOKEN=your-token
SONAR_HOST_URL=https://sonarcloud.io
SONAR_PROJECT_KEY=your-project
```

El reporte incluirá secciones de **AI Analysis** y **SonarQube Results**.

### 5. Notificaciones a Microsoft Teams

```json
{
  "SendTeamsNotification": true,
  "TeamsWebhookURL": ""
}
```

```bash
# Variable de entorno
TEAMS_WEBHOOK_URL=https://outlook.office.com/webhook/...
```

Cada ejecución enviará:
```
🤖 AI Agent report generado para branch dev. PR: #123
View Report: https://github.com/owner/repo/pull/123
```

### 6. Modo Mock para Desarrollo

```bash
# En local o CI para testing sin consumir API
export USE_MOCK_MOTOR_AI=true
./ai-agent-linux

# O en config:
{
  "UseMockMotorAI": true,
  "Agents": [
    {
      "Type": "openai",
      "UseMockMotorAI": true  // Override por agente
    }
  ]
}
```

---

## Troubleshooting

### Problema 1: Error 429 "Too Many Requests"

**Síntomas:**
```
Error evaluando: status code: 429, status: 429 Too Many Requests
[Metrics] RateLimits(429): 15 | Retries: 45
```

**Solución:**
```json
{
  "BatchSize": 1,              // Reducir a 1 archivo por lote
  "RequestIntervalMs": 5000,   // Aumentar pausa a 5 segundos
  "MaxRetries": 10,            // Más reintentos
  "BackoffInitialMs": 2000,    // Backoff inicial mayor
  "CircuitBreakerMax": 3       // Abrir circuito más rápido
}
```

### Problema 2: Circuit Breaker Abierto

**Síntomas:**
```
[Metrics] CircuitBreaker trips: 5
Error: circuit breaker is open, waiting 30s before retry
```

**Causa:** Múltiples fallos consecutivos (429, 5xx errors)

**Solución:**
1. Verificar API key válida
2. Verificar cuota disponible en el proveedor
3. Aumentar `CircuitBreakerWait` a 60s
4. Reducir `BatchSize` y aumentar `RequestIntervalMs`

### Problema 3: GitHub Token sin Permisos

**Síntomas:**
```
Error creando PR: Resource not accessible by integration
403 Forbidden
```

**Solución:**
```yaml
# En workflow, agregar permisos:
permissions:
  contents: write
  pull-requests: write
  issues: write

# O usar PAT con scopes: repo, workflow
```

### Problema 4: Binario no Ejecutable

**Síntomas:**
```
bash: ./ai-agent-linux: Permission denied
```

**Solución:**
```bash
chmod +x ai-agent-linux
```

### Problema 5: Configuración no Encontrada

**Síntomas:**
```
Error cargando configuración: error leyendo config default: 
open config/config_AIAgent.json: no such file or directory
```

**Solución:**
```bash
# Verificar estructura:
ls -la config/config_AIAgent.json

# O especificar ruta completa en código (futuro):
./ai-agent-linux --config=/path/to/config_AIAgent.json
```

### Problema 6: Timeout en Evaluación

**Síntomas:**
```
context deadline exceeded
timeout after 300s
```

**Solución:**
```go
// En workflow, aumentar timeout:
timeout-minutes: 30  # Default: 360 (6 horas)

// En código (futuro enhancement):
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
```

### Problema 7: Métricas Muestran Fallos Altos

**Síntomas:**
```
[Metrics] Success: 5 | Failures: 20 | Retries: 60
```

**Debug:**
```bash
# Revisar logs detallados
./ai-agent-linux 2>&1 | tee agent-debug.log

# Verificar errores específicos
grep -i error agent-debug.log
grep -i "429\|rate limit\|quota" agent-debug.log

# Probar con mock mode
export USE_MOCK_MOTOR_AI=true
./ai-agent-linux
```

---

## Mejores Prácticas

### 1. Seguridad de Credenciales

✅ **HACER:**
- Usar GitHub Secrets para API keys
- Nunca commitear claves en `config_AIAgent.json`
- Rotar claves periódicamente (cada 90 días)
- Usar permisos mínimos necesarios (least privilege)

❌ **NO HACER:**
- Hardcodear API keys en código
- Compartir tokens en Slack/Teams
- Usar mismo token para dev y prod
- Commitear archivos `.env` con claves

### 2. Configuración de Rate Limits

Para diferentes niveles de API:

**Tier Free (OpenAI):**
```json
{
  "BatchSize": 1,
  "RequestIntervalMs": 5000,
  "MaxRetries": 10,
  "CircuitBreakerMax": 3
}
```

**Tier Paid (OpenAI Plus):**
```json
{
  "BatchSize": 3,
  "RequestIntervalMs": 1000,
  "MaxRetries": 5,
  "CircuitBreakerMax": 5
}
```

**Enterprise:**
```json
{
  "BatchSize": 10,
  "RequestIntervalMs": 500,
  "MaxRetries": 3,
  "CircuitBreakerMax": 10
}
```

### 3. Estrategia de Branches

```
main (production)
  ↑
  │ PR (requiere AI review score > 80)
  │
dev (staging)
  ↑
  │ PR (requiere AI review score > 70)
  │
feature-branches (desarrollo)
  (AI review informativo, no bloqueante)
```

### 4. Monitoreo y Alertas

**Métricas a Monitorear:**
- Success rate (debe ser > 90%)
- Retry count (debe ser < 10% de intentos)
- Rate limit errors (debe ser < 5%)
- Circuit breaker trips (debe ser 0 en producción)
- Latencia promedio (debe ser < 30s por archivo)

**Alertas Recomendadas:**
```yaml
# En Teams/Slack
- Si SuccessRate < 80%: "⚠️ AI Agent degraded performance"
- Si RateLimitCount > 10: "🚨 API quota near limit"
- Si CircuitBreakerTrips > 3: "🔥 Circuit breaker tripping frequently"
```

### 5. Optimización de Costos

**Estrategia 1: Evaluación Selectiva**
```bash
# Solo archivos modificados en el PR
# Ahorro: 70-90% de llamadas API
```

**Estrategia 2: Caching de Resultados**
```bash
# No reevaluar archivos sin cambios (futuro enhancement)
# Ahorro: 50-80% de llamadas API
```

**Estrategia 3: Modelos Económicos**
```json
{
  "Model": "gpt-4o-mini",  // Más barato que gpt-4
  "MaxTokens": 1200        // Limitar respuesta
}
```

**Estrategia 4: Múltiples Proveedores**
```json
{
  "Agents": [
    {"Type": "cohere", "Enabled": true},   // Más barato
    {"Type": "openai", "Enabled": false}   // Backup
  ]
}
```

### 6. Testing en CI/CD

```yaml
# Stage 1: Validar configuración
- name: Validate Config
  run: |
    jq empty config/config_AIAgent.json
    test -f ai-agent-linux
    test -x ai-agent-linux

# Stage 2: Modo mock (sin costo)
- name: Smoke Test (Mock)
  env:
    USE_MOCK_MOTOR_AI: true
  run: ./ai-agent-linux

# Stage 3: Modo real (en PRs importantes)
- name: Real AI Review
  if: github.event.pull_request.base.ref == 'main'
  env:
    OPENAI_API_KEY: ${{ secrets.OPENAI_API_KEY }}
  run: ./ai-agent-linux
```

---

## Ejemplos de Uso

### Ejemplo 1: Proyecto Go Simple

**Estructura:**
```
my-go-project/
├── .github/workflows/ai-agent.yml
├── config/config_AIAgent.json
├── main.go
├── handlers/
│   ├── user.go
│   └── auth.go
└── models/
    └── user.go
```

**Configuración Mínima:**
```json
{
  "Agents": [
    {
      "Type": "openai",
      "Enabled": true,
      "Model": "gpt-4o-mini",
      "BatchSize": 2,
      "RequestIntervalMs": 2000
    }
  ],
  "TargetDir": "./",
  "GitHubRepo": "myorg/my-go-project",
  "BaseBranch": "main"
}
```

**Resultado:**
```markdown
## handlers/user.go
**Score:** 85/100

### Factores no cumplidos
- Logs no estructurados (usar zap)

### Recomendaciones
- Añadir context.Context para timeout control
- Implementar error wrapping con fmt.Errorf
```

### Ejemplo 2: Monorepo con Go + Python

**Estructura:**
```
monorepo/
├── backend/       # Go
│   ├── api/
│   └── services/
├── scripts/       # Python
│   ├── deploy.py
│   └── migrate.py
└── config/config_AIAgent.json
```

**Configuración:**
```json
{
  "Agents": [
    {
      "Type": "openai",
      "Enabled": true,
      "Model": "gpt-4o-mini"
    }
  ],
  "TargetDir": "./",  // Evalúa todo el monorepo
  "GitHubRepo": "myorg/monorepo",
  "BaseBranch": "dev"
}
```

El agente evaluará **automáticamente** archivos `.go` y `.py` encontrados.

### Ejemplo 3: Integración con CI/CD Multi-stage

```yaml
name: Multi-stage Pipeline

on: [push, pull_request]

jobs:
  # Stage 1: Unit tests
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Run tests
        run: go test ./...
  
  # Stage 2: AI Review (paralelo con tests)
  ai-review:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Download AI Agent
        run: |
          wget https://github.com/rcrala/ai-agent-go/releases/latest/download/ai-agent-linux
          chmod +x ai-agent-linux
      - name: AI Review
        env:
          OPENAI_API_KEY: ${{ secrets.OPENAI_API_KEY }}
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
          GITHUB_REPO: ${{ github.repository }}
        run: ./ai-agent-linux
      - name: Upload report
        uses: actions/upload-artifact@v4
        with:
          name: ai-report
          path: ARQUITECTURE_COMPLIANCE.md
  
  # Stage 3: SonarQube (paralelo)
  sonar:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: SonarQube Scan
        uses: sonarsource/sonarcloud-github-action@master
        env:
          SONAR_TOKEN: ${{ secrets.SONAR_TOKEN }}
  
  # Stage 4: Deploy (requiere todos los anteriores)
  deploy:
    needs: [test, ai-review, sonar]
    runs-on: ubuntu-latest
    if: github.ref == 'refs/heads/main'
    steps:
      - name: Deploy to production
        run: ./deploy.sh
```

### Ejemplo 4: Evaluación de PR con Comentario

```yaml
name: PR Review with Comment

on:
  pull_request:
    types: [opened, synchronize]

jobs:
  review:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      
      - name: Download AI Agent
        run: |
          wget https://github.com/rcrala/ai-agent-go/releases/latest/download/ai-agent-linux
          chmod +x ai-agent-linux
      
      - name: Run AI Review
        env:
          OPENAI_API_KEY: ${{ secrets.OPENAI_API_KEY }}
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
          GITHUB_REPO: ${{ github.repository }}
        run: ./ai-agent-linux
      
      - name: Extract Scores
        id: scores
        run: |
          scores=$(grep -oP 'Score:\*\*\s*\K\d+' ARQUITECTURE_COMPLIANCE.md | tr '\n' ',' | sed 's/,$//')
          avg=$(echo $scores | tr ',' '\n' | awk '{s+=$1; c++} END {print int(s/c)}')
          echo "average=$avg" >> $GITHUB_OUTPUT
          echo "all=$scores" >> $GITHUB_OUTPUT
      
      - name: Comment on PR
        uses: actions/github-script@v7
        with:
          script: |
            const fs = require('fs');
            const report = fs.readFileSync('ARQUITECTURE_COMPLIANCE.md', 'utf8');
            const avgScore = ${{ steps.scores.outputs.average }};
            
            let emoji = '✅';
            if (avgScore < 70) emoji = '❌';
            else if (avgScore < 80) emoji = '⚠️';
            
            const comment = `## ${emoji} AI Agent Review - Score: ${avgScore}/100
            
            <details>
            <summary>📊 Full Report</summary>
            
            \`\`\`
            ${report}
            \`\`\`
            
            </details>
            
            ${avgScore < 70 ? '⚠️ **Score below 70 - Please address issues before merging**' : ''}
            `;
            
            github.rest.issues.createComment({
              issue_number: context.issue.number,
              owner: context.repo.owner,
              repo: context.repo.repo,
              body: comment
            });
```

### Ejemplo 5: Modo Desarrollo Local

```bash
#!/bin/bash
# local-review.sh - Script para evaluación local

# 1. Configurar entorno
export USE_MOCK_MOTOR_AI=false
export OPENAI_API_KEY="sk-proj-..."
export GITHUB_TOKEN="ghp_..."
export GITHUB_REPO="myorg/myrepo"
export BASE_BRANCH="dev"
export TARGET_DIR="./src"  # Solo evaluar src/

# 2. Ejecutar
./ai-agent-linux

# 3. Mostrar resultado
echo "===== EVALUATION COMPLETE ====="
head -n 50 ARQUITECTURE_COMPLIANCE.md

# 4. Abrir en navegador (opcional)
# markdown ARQUITECTURE_COMPLIANCE.md > report.html
# open report.html
```

---

## Referencias Técnicas

### Endpoints de API

| Proveedor | Endpoint | Docs |
|-----------|----------|------|
| OpenAI | `https://api.openai.com/v1/chat/completions` | https://platform.openai.com/docs |
| Cohere | `https://api.cohere.ai/v1/generate` | https://docs.cohere.com |
| Anthropic | `https://api.anthropic.com/v1/messages` | https://docs.anthropic.com |
| Google Gemini | `https://generativelanguage.googleapis.com/v1beta/models` | https://ai.google.dev/docs |

### Métricas Exportadas

```go
type EvaluationMetrics struct {
    TotalAttempts       int    // Total de intentos de evaluación
    SuccessCount        int    // Evaluaciones exitosas
    FailureCount        int    // Evaluaciones fallidas
    RetryCount          int    // Total de reintentos
    RateLimitCount      int    // Errores 429
    TotalLatencyMs      int64  // Latencia total acumulada
    CircuitBreakerTrips int    // Veces que abrió el circuit breaker
}
```

### Estructura del Reporte

```markdown
# Architecture Compliance Report

## Execution Metadata
- Start: 2025-10-16T01:42:54Z
- End: 2025-10-16T01:45:22Z
- Commit: e179ee2
- Agents: openai (gpt-4o-mini)

---

## cmd/ai-agent/main.go
**Score:** 85/100

### Factores no cumplidos
- [Lista de factores]

### Problemas de concurrencia
- [Lista de problemas]

### Recomendaciones de refactorización
[Texto]

### Evaluación por función
| Función | Claridad | Complejidad | Riesgo | Sugerencias |
|---------|----------|-------------|--------|-------------|
| main    | Alta     | Media       | Bajo   | [...]       |
```

### Compatibilidad

| Entorno | Soporte | Notas |
|---------|---------|-------|
| Ubuntu 20.04+ | ✅ | Compilado para linux/amd64 |
| Debian 10+ | ✅ | Compatible |
| macOS | ⚠️ | Requiere recompilación: `GOOS=darwin go build` |
| Windows | ⚠️ | Requiere recompilación: `GOOS=windows go build -o ai-agent.exe` |
| Docker/K8s | ✅ | Usar imagen base con binario |
| GitHub Actions | ✅ | Ubuntu runner |
| GitLab CI | ✅ | Docker runner |
| Jenkins | ✅ | Linux agent |

---

## Roadmap y Mejoras Futuras

### En Desarrollo
- [ ] CLI subcommands (`ai-agent validate-config`, `ai-agent test-agent`)
- [ ] Structured JSON logging (zap/zerolog)
- [ ] Selective file evaluation (solo archivos modificados)
- [ ] Caching de resultados (no reevaluar sin cambios)

### Planificado
- [ ] Webhooks para notificaciones personalizadas
- [ ] Export de métricas a Prometheus/Grafana
- [ ] Dashboard web para visualización de reportes históricos
- [ ] Integración con Jira (crear issues para deuda técnica)
- [ ] Soporte para más lenguajes (Java, TypeScript, Rust)

### Considerado
- [ ] Plugin de VSCode para evaluación en tiempo real
- [ ] API REST para evaluaciones on-demand
- [ ] Base de datos para histórico de scores
- [ ] Machine Learning para predicción de problemas

---

## Soporte y Contribuciones

### Repositorio Principal
https://github.com/rcrala/ai-agent-go

### Reportar Issues
https://github.com/rcrala/ai-agent-go/issues

### Contribuir
```bash
# 1. Fork del repositorio
# 2. Crear branch: git checkout -b feature/mi-mejora
# 3. Commit: git commit -am 'Agregar nueva funcionalidad'
# 4. Push: git push origin feature/mi-mejora
# 5. Crear Pull Request
```

### Contacto
- **Email:** soporte@aiagent.dev (ejemplo)
- **Slack:** #ai-agent-support (ejemplo)
- **Documentation:** https://docs.aiagent.dev (ejemplo)

---

## Licencia

[Especificar licencia del proyecto]

---

**Última Actualización:** Octubre 16, 2025  
**Versión del Documento:** 1.0  
**Autor:** GitHub Copilot & rcrala
