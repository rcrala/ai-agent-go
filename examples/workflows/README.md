# Ejemplo de Workflow Completo
## GitHub Actions - AI Agent Integration

Este directorio contiene ejemplos de workflows de GitHub Actions para integrar el AI Agent en tu proyecto.

## 📋 Workflows Incluidos

### 1. `ai-agent-pr-review.yml` - Revisión Automática de PRs
Ejecuta análisis AI en cada Pull Request y comenta resultados.

### 2. `ai-agent-weekly-audit.yml` - Auditoría Semanal
Ejecuta análisis programado y crea issues si score < 70.

### 3. `ai-agent-required-check.yml` - Verificación Obligatoria
Bloquea merge si el score arquitectónico es muy bajo.

### 4. `ai-agent-multi-stage.yml` - Pipeline Multi-etapa
Integra AI review con tests, SonarQube y deployment.

---

## Instalación Rápida

1. **Crear directorio de workflows:**
```bash
mkdir -p .github/workflows
```

2. **Copiar workflow deseado:**
```bash
# Opción recomendada: PR Review
cp examples/workflows/ai-agent-pr-review.yml .github/workflows/

# O todos los workflows
cp examples/workflows/*.yml .github/workflows/
```

3. **Configurar secrets en GitHub:**
```
Settings → Secrets and variables → Actions → New repository secret
```

Agregar:
- `OPENAI_API_KEY` (o el proveedor que uses)
- `GITHUB_TOKEN` (automático o personal)
- `TEAMS_WEBHOOK_URL` (opcional)

4. **Ajustar configuración:**
Editar `config/config_AIAgent.json` según tus necesidades.

5. **Crear PR y probar:**
```bash
git checkout -b test-ai-agent
git commit --allow-empty -m "Test AI Agent"
git push origin test-ai-agent
# Crear PR en GitHub UI
```

---

## Comparación de Workflows

| Workflow | Trigger | Uso | Bloquea Merge | Notificaciones |
|----------|---------|-----|---------------|----------------|
| **PR Review** | En cada PR | Review automático | No | Comentario en PR |
| **Weekly Audit** | Lunes 9am | Auditoría periódica | No | Issue si score < 70 |
| **Required Check** | PR a main | Control de calidad | Sí (score < 70) | Status check |
| **Multi-stage** | Push/PR | CI/CD completo | Sí (si falla stage) | Artifact upload |

---

## Personalización

### Cambiar Branch Base
```yaml
on:
  push:
    branches: [main, develop]  # Cambiar según tu estrategia
```

### Ajustar Score Mínimo
```bash
# En required-check workflow
if [ "$score" -lt 75 ]; then  # Cambiar 70 → 75
```

### Agregar Más Proveedores AI
```json
// config/config_AIAgent.json
{
  "Agents": [
    {"Type": "openai", "Enabled": true},
    {"Type": "cohere", "Enabled": true},
    {"Type": "anthropic", "Enabled": true}
  ]
}
```

### Filtrar Archivos
```yaml
- name: Get changed files
  uses: tj-actions/changed-files@v40
  with:
    files: |
      **/*.go
      **/*.py
      !vendor/**
      !*_test.go
```

---

## Troubleshooting

### Workflow no se Ejecuta
1. Verificar triggers en el YAML
2. Revisar permisos del repositorio
3. Check branch protection rules

### Error de Permisos
```yaml
permissions:
  contents: write
  pull-requests: write
  issues: write
```

### Binario No Descargado
```yaml
- name: Download AI Agent
  run: |
    wget https://github.com/rcrala/ai-agent-go/releases/latest/download/ai-agent-linux
    chmod +x ai-agent-linux
    ./ai-agent-linux --version  # Verificar
```

### Secrets No Disponibles
```yaml
- name: Check Secrets
  run: |
    if [ -z "$OPENAI_API_KEY" ]; then
      echo "❌ OPENAI_API_KEY not set"
      exit 1
    fi
  env:
    OPENAI_API_KEY: ${{ secrets.OPENAI_API_KEY }}
```

---

## Mejores Prácticas

### 1. Usar Cache para Binario
```yaml
- name: Cache AI Agent Binary
  uses: actions/cache@v3
  with:
    path: ai-agent-linux
    key: ai-agent-${{ hashFiles('**/go.sum') }}
```

### 2. Timeout Apropiado
```yaml
jobs:
  ai-review:
    timeout-minutes: 30  # Ajustar según tamaño del repo
```

### 3. Artifact Retention
```yaml
- name: Upload Report
  uses: actions/upload-artifact@v4
  with:
    name: ai-report-${{ github.run_number }}
    retention-days: 30
```

### 4. Conditional Execution
```yaml
# Solo en PRs importantes
if: |
  github.event.pull_request.base.ref == 'main' &&
  github.event.pull_request.draft == false
```

### 5. Matrix Strategy para Múltiples Agentes
```yaml
strategy:
  matrix:
    agent: [openai, cohere, anthropic]
steps:
  - name: Run ${{ matrix.agent }} Review
    env:
      AGENT_TYPE: ${{ matrix.agent }}
```

---

## Recursos Adicionales

- [GitHub Actions Documentation](https://docs.github.com/actions)
- [Workflow Syntax](https://docs.github.com/actions/reference/workflow-syntax-for-github-actions)
- [Actions Marketplace](https://github.com/marketplace?type=actions)
- [AI Agent Main Docs](../../README.md)

---

## Soporte

Issues: https://github.com/rcrala/ai-agent-go/issues  
Discussions: https://github.com/rcrala/ai-agent-go/discussions
