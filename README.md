
# ai-agent-go

Agente de revisión de código en Go con arquitectura extensible de agentes IA (OpenAI, Copilot, etc.).

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
- La clave de cada agente puede configurarse por variable de entorno (`OPENAI_API_KEY`, `COPILOT_API_KEY`, etc.).

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
