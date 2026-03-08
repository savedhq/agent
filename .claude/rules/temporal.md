---
paths:
  - "temporal/**"
  - "infra/**/temporal*/**"
---

# Temporal

## Clusters
- **temporal-public**: customized server at `temporal-server/`, for **agents only**
  - Infra (beta): `infra/flux/apps/saved-beta/temporal-public/`
  - Local: `infra/local/temporal-public/`
- **temporal-private**: stock open-source Temporal, for **workers only**
  - Local: `infra/local/temporal-private/`

## Access Rules
- Agent connects to temporal-public only — do not configure other services to use temporal-public
- Worker connects to temporal-private only
- Backend has access to both clusters

## Workflows & Activities Structure
Mirror layout: agent and worker each have their own `workflows/` and `activities/` directories with parallel-but-distinct logic:
- Agent: `agent/internal/temporal/workflows/`, `agent/internal/temporal/activities/`
- Worker: `backend/internal/temporal/workflows/`, `backend/internal/temporal/activities/`
