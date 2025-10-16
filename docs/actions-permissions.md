# GitHub Actions: Fixing "not permitted to create or approve pull requests"

This document explains two safe ways to allow a GitHub Actions workflow to create or update files and open pull requests from the workflow runner:

1) Grant the workflow explicit permissions (recommended when you control the repo)
2) Use a Personal Access Token (PAT) stored as a secret (recommended when repository/org policies restrict workflow permissions)

Both approaches are shown below with exact YAML snippets and PowerShell commands you can use for local testing.

---

## 1) Grant workflow permissions (recommended)

Add the following `permissions` block near the top of your workflow (top-level under `jobs` or at root of the workflow file). This gives the runner the contents and pull-requests write scopes required to create branches, update files and open PRs.

Example (workflow YAML):

```yaml
name: ai-agent
on:
  push:
    branches: [ main ]

permissions:
  contents: write
  pull-requests: write

jobs:
  ai-agent:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Run AI agent
        run: |
          # build + run the ai-agent binary or run go run
          go run ./cmd/ai-agent
```

Notes:
- You may place the `permissions:` block at the root of the workflow (as above) or inside an individual `job` (if you only want that job to have elevated permissions).
- Some organizations may enforce more restrictive default permissions; repo admins can always change default workflow permissions in repo Settings → Actions → General → Workflow permissions.

---

## 2) Use a Personal Access Token (PAT) via secrets (fallback)

If you cannot grant the workflow the needed permissions (organization policy, limited repo permissions), create a PAT with the required `repo` scopes and store it as a secret (for example `ACTIONS_PAT`). Then set the job to export that PAT into `GITHUB_TOKEN` so existing tools that read `GITHUB_TOKEN` will use the PAT.

Example (workflow YAML using a PAT secret):

```yaml
name: ai-agent
on:
  pull_request:
    types: [opened, synchronize]

jobs:
  ai-agent:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Set PAT as GITHUB_TOKEN
        run: |
          echo "GITHUB_TOKEN=$ACTIONS_PAT" >> $GITHUB_ENV
        env:
          ACTIONS_PAT: ${{ secrets.ACTIONS_PAT }}

      - name: Run AI agent
        run: |
          go run ./cmd/ai-agent
```

Notes:
- Create the PAT in your user settings (https://github.com/settings/tokens) with `repo` scope (and any other scopes you need). For private repos `repo` is required.
- Add the PAT to repository secrets (Settings → Secrets → Actions) as `ACTIONS_PAT` (or any name you prefer) before referencing it in the workflow.
- This approach effectively makes `GITHUB_TOKEN` backed by a PAT for the duration of the job.

---

## Local PowerShell testing

Before running inside Actions, you can test behavior locally with PowerShell by exporting an env var and running the agent in the same session.

If you have a PAT stored in `$env:ACTIONS_PAT` (or a plain PAT string), do:

```powershell
# One-off - set the PAT in the current session
$env:GITHUB_TOKEN = 'ghp_XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX'
# Or if you have it in ACTIONS_PAT
$env:GITHUB_TOKEN = $env:ACTIONS_PAT

# Run the agent (in the same shell) so it picks up GITHUB_TOKEN
go run ./cmd/ai-agent
```

PowerShell note: if you created a helper `setDEV.ps1` that exports variables, dot-source it to export into the current session:

```powershell
. .\setDEV.ps1
# then run
go run ./cmd/ai-agent
```

---

## Troubleshooting checklist

- Confirm the job actually receives the token/permissions:
  - Add a step to print `GITHUB_ACTIONS` and `GITHUB_TOKEN` presence (do NOT echo secrets; just check if set):

```yaml
- name: Debug env
  run: |
    echo "GITHUB_ACTIONS=$GITHUB_ACTIONS"
    if [ -z "$GITHUB_TOKEN" ]; then echo "GITHUB_TOKEN not set"; else echo "GITHUB_TOKEN set"; fi
```

- If you still see `403: GitHub Actions is not permitted to create or approve pull requests.` then:
  - Either your workflow lacks the `permissions:` block or your repo default permissions override it.
  - Or the token in `GITHUB_TOKEN` is not a PAT with repo scope (when using PAT fallback).

- To verify a PAT locally:
  - Use the PAT with curl to test the REST API security boundary quickly:

```powershell
$pat = 'ghp_XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX'
$headers = @{ Authorization = "token $pat"; Accept = "application/vnd.github+json" }
Invoke-RestMethod -Uri "https://api.github.com/user" -Headers $headers
```

If the user info returns, the PAT works and is valid for API calls.

---

## Security notes

- Never print or commit PATs or secrets. Use repository secrets and mask values in Actions.
- Prefer the minimal permissions possible. Granting `contents: write` and `pull-requests: write` to a workflow is a sensitive capability; use PAT fallback with a dedicated technical user when appropriate.

---

If you want, I can add a sample workflow file to `.github/workflows/` in this repository that demonstrates both approaches (commented), and/or add a `--dry-run` flag to the agent so CI runs do not create PRs automatically. Tell me which and I'll implement it.
