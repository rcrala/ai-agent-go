# Copilot Instructions for ai-agent-go

## Project Overview
- **ai-agent-go** is a Go-based code review agent that evaluates codebases for compliance with the Twelve-Factor App methodology and other best practices, leveraging pluggable AI agents (OpenAI, Copilot, etc.) and GitHub APIs.
- The agent is designed to run as a CLI tool and as a GitHub Action, providing automated code analysis and reporting via Teams and GitHub PRs.

## Architecture & Key Components
- **cmd/ai-agent/main.go**: Entry point. Loads config, initializes clients, runs all enabled AI agents (via a generic interface), SonarQube analysis, and sends notifications.
- **internal/ai/agent.go**: Defines the `CodeEvaluator` interface, agent factory, config loading, and generic file evaluation logic.
- **internal/ai/openai_agent.go**: Implements the OpenAI agent as a `CodeEvaluator`.
- **internal/ai/copilot_agent.go**: Implements the Copilot agent as a `CodeEvaluator`.
- **internal/github/**: GitHub API client for branch, PR, and file operations. See `gh_client.go` for REST API usage and custom error handling.
- **internal/teams/**: Sends notifications to Microsoft Teams via webhook (`webhook.go`).
- **config/config_AIAgent.json**: Main configuration file. Now supports a collection of agents with their own settings.

## Developer Workflows
- **Build**: `go build -o ai-agent-linux ./cmd/ai-agent` (see `.github/workflows/twelve-factor.yml`)
- **Run Locally**: Set required env vars for each agent (`OPENAI_API_KEY`, `COPILOT_API_KEY`, `GITHUB_TOKEN`, etc.), then run the built binary.
- **GitHub Action**: Triggered on PRs and pushes to `main`/`dev`. Requires secrets for each agent you want to use: `OPENAI_API_KEY`, `COPILOT_API_KEY`, `TEAMS_WEBHOOK_URL`, `GITHUB_TOKEN`.
- **Config**: Edit `config/config_AIAgent.json` to add, enable, or configure agents in the `Agents` array. Each agent can have its own type, key, model, and parameters.

## Project-Specific Patterns & Conventions
- **Pluggable Agents**: All AI agents implement the `CodeEvaluator` interface. Add new agents by implementing this interface and registering them in the factory in `agent.go`.
- **Agent Key Management**: Each agent's API key can be set in the config or overridden by environment variable (e.g., `OPENAI_API_KEY`, `COPILOT_API_KEY`).
- **Concurrency**: File evaluation is batched and parallelized (see `EvaluateFilesGeneric` in `agent.go`).
- **Error Handling**: Errors are logged and reported, but do not always halt execution (see `main.go`).
- **Extensibility**: Add new agent logic in a new file, implement `CodeEvaluator`, and update the factory.
- **Notifications**: Teams integration is optional and controlled by config.
- **Spanish Naming**: Many variables, comments, and config fields use Spanish for clarity to the team.

## Integration Points
- **OpenAI**: Uses `github.com/sashabaranov/go-openai` for LLM-based code evaluation (see `openai_agent.go`).
- **Copilot**: Placeholder agent for GitHub Copilot-based evaluation (see `copilot_agent.go`).
- **GitHub**: Uses REST API for PR/branch/file operations (see `gh_client.go`).
- **Teams**: Sends results to Teams via webhook if enabled.
- **SonarQube**: Optional; controlled by config.


## Copilot Agent Usage

To enable and use the Copilot agent:

1. In `config/config_AIAgent.json`, add or update an entry in the `Agents` array:

	 ```json
	 {
		 "Type": "copilot",
		 "Enabled": true,
		 "Key": "", // or leave blank to use env var
		 "Model": "copilot-2025",
		 "MaxTokens": 1200,
		 "Temperature": 0.0,
		 "BatchSize": 3
	 }
	 ```

2. Set the environment variable `COPILOT_API_KEY` with your Copilot API key (or set the `Key` field in config).

3. Run the agent as usual (locally or via GitHub Action). The system will automatically use all enabled agents, including Copilot, with no code changes required in `main.go`.

4. Review results in the generated report, PR, or Teams notification as configured.

---

- To add a new agent, create a new file (e.g., `myagent_agent.go`), implement the `CodeEvaluator` interface, and register it in the factory in `agent.go`.

## Mock mode for testing
You can run the agent without calling external LLM services by enabling mock mode.

- Per-agent mock (OpenAI only): set `OPENAI_MOCK=true` in the environment.
- Global mock for all agents: set `USE_MOCK_MOTOR_AI=true` in the environment. This overrides per-agent mock flags and forces mock responses across agents.

Example (PowerShell):
```powershell
$env:USE_MOCK_MOTOR_AI = 'true'
go run ./cmd/ai-agent
```
- To run only AI analysis, enable the desired agents in config and set `RunSonar: false`.

---

For more, see `README.md` and comments in each key file. When in doubt, follow the structure and patterns in `internal/ai/agent.go` and `cmd/ai-agent/main.go`.
