---
paths:
  - "agent/**"
---

# Agent

- Open source, publicly downloadable client
- Connects **only** to temporal-public (custom Temporal server at `temporal-server/`)
- No direct access to DB, Vault, or S3 — must call the backend API for all sensitive operations
- Stores client-sensitive configuration locally in `agent/config.yaml`
- Entry point: `agent/cmd/main.go`
- Runs on the client's system — has access to the client's local resources

## Agent Workflows
- Temporal code: `agent/internal/temporal/workflows/`, `agent/internal/temporal/activities/`
- Responsible only for its own backup jobs (e.g. `git.go` ↔ `provider_git.go`)
- Supported providers: all worker providers + `script`
