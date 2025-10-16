# setDEV.ps1
# PowerShell helper to configure environment variables for local testing of ai-agent-go
# Edit the placeholder values below. This script sets variables only for the current session.
# WARNING: Do NOT commit real tokens to source control. Keep this file local and secure.

# -----------------------------
# Required variables (replace placeholders)
# -----------------------------
# GitHub Personal Access Token (PAT) with repo write permissions
$env:GITHUB_TOKEN = 'ghp_XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX'

# Repository in owner/repo format
$env:GITHUB_REPO = 'rcrala/ai-agent-go'

# Base branch to create PRs against
$env:BASE_BRANCH = 'dev'

# Global mock flag for AI engines; set to 'true' to avoid real LLM calls (mock mode)
$env:USE_MOCK_MOTOR_AI = 'true'

# -----------------------------
# Optional variables (uncomment and set if needed)
# -----------------------------
# OpenAI API key (if you plan to run real OpenAI requests)
# $env:OPENAI_API_KEY = 'sk-...'

# Copilot API key (if using Copilot agent)
# $env:COPILOT_API_KEY = 'copilot-...'

# Teams webhook to send notifications (optional)
# $env:TEAMS_WEBHOOK_URL = 'https://outlook.office.com/webhook/....'

# SonarQube configuration (optional)
# $env:SONAR_HOST_URL = 'https://sonarqube.example.com'
# $env:SONAR_PROJECT_KEY = 'my-project-key'
# $env:SONAR_TOKEN = 'sonar-token'

# Target directory (optional) - relative path where the agent scans files
# $env:TARGET_DIR = './'

# -----------------------------
# Helpful verification commands
# -----------------------------
Write-Host "Environment variables set for this session:" -ForegroundColor Cyan
Write-Host "GITHUB_REPO = $env:GITHUB_REPO"
Write-Host "BASE_BRANCH  = $env:BASE_BRANCH"
Write-Host "USE_MOCK_MOTOR_AI = $env:USE_MOCK_MOTOR_AI"
if ($env:OPENAI_API_KEY) { Write-Host "OPENAI_API_KEY is set" }
if ($env:COPILOT_API_KEY) { Write-Host "COPILOT_API_KEY is set" }

Write-Host "\nQuick checks (optional):" -ForegroundColor Cyan
Write-Host "1) Verify GitHub token is valid (will return user info):"
Write-Host "   Invoke-RestMethod -Headers @{ Authorization = \"token $env:GITHUB_TOKEN\" } -Uri 'https://api.github.com/user'"

Write-Host "2) Build the agent (optional):"
Write-Host "   go build ./cmd/ai-agent"

Write-Host "3) Run the agent (mock AI + real GitHub operations if token present):"
Write-Host "   go run ./cmd/ai-agent"

Write-Host "\nRemember: do not commit this file with real tokens. Use a secrets manager or environment variables for CI." -ForegroundColor Yellow
