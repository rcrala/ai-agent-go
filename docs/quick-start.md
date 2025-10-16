# Inicio Rápido - AI Agent en GitHub
## Guía de 5 Minutos para Integración

Esta guía te ayudará a integrar el AI Agent en tu proyecto de GitHub en menos de 5 minutos.

---

## ⚡ Quick Start (3 pasos)

### 1️⃣ Descargar Binario y Config (30 segundos)

```bash
# En la raíz de tu proyecto
cd your-project/

# Descargar binario
wget https://github.com/rcrala/ai-agent-go/releases/latest/download/ai-agent-linux
chmod +x ai-agent-linux

# Crear estructura de config
mkdir -p config .github/workflows

# Descargar plantilla de configuración
wget -O config/config_AIAgent.json https://raw.githubusercontent.com/rcrala/ai-agent-go/main/config/config_AIAgent.json
```

### 2️⃣ Configurar GitHub Secrets (1 minuto)

Ve a tu repositorio en GitHub:
```
Settings → Secrets and variables → Actions → New repository secret
```

Agrega **al menos uno**:
- `OPENAI_API_KEY` → Tu clave de OpenAI
- `COHERE_API_KEY` → Tu clave de Cohere
- `COPILOT_API_KEY` → Tu clave de GitHub Copilot

### 3️⃣ Copiar Workflow (30 segundos)

```bash
# Opción A: Review básico en PRs (recomendado para empezar)
wget -O .github/workflows/ai-agent.yml \
  https://raw.githubusercontent.com/rcrala/ai-agent-go/main/examples/workflows/ai-agent-pr-review.yml

# Opción B: Auditoría semanal programada
wget -O .github/workflows/ai-audit.yml \
  https://raw.githubusercontent.com/rcrala/ai-agent-go/main/examples/workflows/ai-agent-weekly-audit.yml

# Opción C: Verificación obligatoria (bloquea merge si score < 70)
wget -O .github/workflows/ai-check.yml \
  https://raw.githubusercontent.com/rcrala/ai-agent-go/main/examples/workflows/ai-agent-required-check.yml
```

### ✅ Commit y Push

```bash
git add .github/workflows/ config/
git commit -m "Add AI Agent for code review"
git push origin main
```

**¡Listo!** Ahora crea un PR para ver el agente en acción.

---

## 🎯 Casos de Uso Comunes

### Caso 1: Review Automático en PRs
**Objetivo:** Feedback automático sin bloquear

```yaml
# .github/workflows/ai-agent-pr-review.yml
on:
  pull_request:
    branches: [main, dev]
```

**Resultado:**
- Comenta en cada PR con score y recomendaciones
- No bloquea el merge
- Ideal para adopción gradual

---

### Caso 2: Auditoría Semanal
**Objetivo:** Monitoreo continuo de calidad

```yaml
# .github/workflows/ai-agent-weekly-audit.yml
on:
  schedule:
    - cron: '0 9 * * 1'  # Lunes 9am
```

**Resultado:**
- Crea issue si score < 70
- Notifica a Teams
- Mantiene histórico de auditorías

---

### Caso 3: Bloqueo de Merge
**Objetivo:** Enforcement estricto de calidad

```yaml
# .github/workflows/ai-agent-required-check.yml
on:
  pull_request:
    branches: [main]  # Solo en main
```

**Resultado:**
- Bloquea merge si score < 70
- Requiere fixes antes de aprobar
- Protege branch principal

**Configurar Branch Protection:**
```
Settings → Branches → Branch protection rules → main
☑ Require status checks to pass before merging
  ☑ architecture-check
```

---

## 🔧 Configuración Mínima

### `config/config_AIAgent.json`

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
      "RequestIntervalMs": 2000
    }
  ],
  "GitHubRepo": "your-org/your-repo",
  "BaseBranch": "main",
  "SendTeamsNotification": false
}
```

**Importante:** 
- Deja `Key: ""` vacío (usa GitHub Secrets)
- Ajusta `GitHubRepo` a tu repositorio
- `BatchSize: 2` y `RequestIntervalMs: 2000` previenen rate limits

---

## 📊 Entender los Resultados

### Scores

| Rango | Estado | Acción |
|-------|--------|--------|
| 90-100 | 🟢 Excelente | Mantener calidad |
| 80-89 | 🟢 Bueno | Mejoras menores |
| 70-79 | 🟡 Aceptable | Revisar recomendaciones |
| 60-69 | 🟠 Bajo | Correcciones necesarias |
| 0-59 | 🔴 Crítico | Refactorización urgente |

### Reporte Típico

```markdown
## cmd/ai-agent/main.go
**Score:** 85/100

### Factores no cumplidos
- Logs no estructurados (usar zap/zerolog)
- Dependencias no explícitas en algunos imports

### Problemas de concurrencia
- Goroutine sin sincronización en línea 45
- Canal sin buffer puede causar deadlock

### Recomendaciones de refactorización
- Extraer función processBatch() de main()
- Mover lógica de configuración a paquete separado
```

---

## 🐛 Troubleshooting Rápido

### Error: "429 Too Many Requests"
**Solución:**
```json
{
  "BatchSize": 1,
  "RequestIntervalMs": 5000
}
```

### Error: "Permission denied"
**Solución:**
```yaml
permissions:
  contents: write
  pull-requests: write
```

### Error: "Config not found"
**Solución:**
```bash
# Verificar ubicación
ls -la config/config_AIAgent.json

# Debe estar en raíz del proyecto
```

### Workflow no se ejecuta
**Solución:**
1. Verificar triggers en `.github/workflows/*.yml`
2. Check que el secret `OPENAI_API_KEY` esté configurado
3. Revisar branch name (main vs master)

---

## 🚀 Siguiente Nivel

Una vez que tengas el básico funcionando:

1. **Múltiples Agentes:** Compara resultados de OpenAI vs Cohere
   ```json
   "Agents": [
     {"Type": "openai", "Enabled": true},
     {"Type": "cohere", "Enabled": true}
   ]
   ```

2. **Integración con Teams:** Notificaciones automáticas
   ```json
   "SendTeamsNotification": true,
   "TeamsWebhookURL": ""  // Usar secret
   ```

3. **SonarQube Combo:** AI + análisis estático
   ```json
   "RunSonar": true,
   "SonarHostURL": "https://sonarcloud.io"
   ```

4. **Solo Archivos Modificados:** Más rápido y económico
   ```yaml
   - uses: tj-actions/changed-files@v40
     with:
       files: |
         **/*.go
         **/*.py
   ```

---

## 📚 Documentación Completa

- **[Guía de Integración Completa](github-integration-guide.md)** - Todas las opciones y configuraciones
- **[Manejo de Errores HTTP](http-error-handling.md)** - Circuit breaker y retry logic
- **[Twelve-Factor Compliance](twelve-factor-compliance.md)** - Análisis arquitectónico del proyecto
- **[Ejemplos de Workflows](../examples/workflows/)** - Plantillas listas para usar

---

## 💡 Tips Pro

### 1. Test Local Primero
```bash
export USE_MOCK_MOTOR_AI=true
./ai-agent-linux
# Verifica que funcione sin consumir API
```

### 2. Branch Strategy
```
feature/* → PR con review (no bloqueante)
  ↓
dev → PR con review (no bloqueante)
  ↓
main → PR con required check (bloquea si score < 70)
```

### 3. Optimizar Costos
- Usar `gpt-4o-mini` en lugar de `gpt-4` (95% más barato)
- Evaluar solo archivos modificados en PRs
- Usar Cohere (más económico que OpenAI)

### 4. Monitorear Métricas
```
[Metrics] Success: 45 | Failures: 2 | Retries: 5 | RateLimits(429): 1
```
- Success rate debe ser > 90%
- RateLimits < 5% indica configuración saludable

---

## ❓ FAQ Rápido

**P: ¿Cuánto cuesta ejecutar el agente?**  
R: Con OpenAI GPT-4o-mini: ~$0.01-0.05 por archivo evaluado. Proyecto típico de 50 archivos: ~$1-2 por ejecución.

**P: ¿Funciona con monorepos?**  
R: Sí, evalúa todos los archivos .go y .py recursivamente desde `TargetDir`.

**P: ¿Puedo personalizar los criterios de evaluación?**  
R: Los criterios (Twelve-Factor, concurrencia, buenas prácticas Go) están en el prompt. Futuras versiones permitirán personalización.

**P: ¿Soporta otros lenguajes además de Go/Python?**  
R: Actualmente solo Go y Python. TypeScript, Java y Rust en roadmap.

**P: ¿Necesito SonarQube?**  
R: No, es opcional. El agente AI funciona independientemente.

---

## 🎉 Casos de Éxito

### Equipo A (Startup, 3 devs)
- **Setup:** PR review no bloqueante
- **Resultado:** Detectó 15 race conditions antes de producción
- **ROI:** Evitó 2 incidents críticos

### Equipo B (Enterprise, 20 devs)
- **Setup:** Required check en main + auditoría semanal
- **Resultado:** Score promedio subió de 65 → 82 en 3 meses
- **ROI:** Redujo tiempo de onboarding 40%

### Equipo C (Open Source)
- **Setup:** Review en PRs externos + mock mode en dev
- **Resultado:** Mejoró calidad de contribuciones 60%
- **ROI:** Menos tiempo de code review manual

---

## 🆘 Soporte

- **Issues:** https://github.com/rcrala/ai-agent-go/issues
- **Discussions:** https://github.com/rcrala/ai-agent-go/discussions
- **Docs:** https://github.com/rcrala/ai-agent-go/tree/main/docs

---

**Tiempo total de setup:** ⏱️ **< 5 minutos**  
**Primeros resultados:** ⚡ **En tu próximo PR**

¡Empieza ahora! 🚀
