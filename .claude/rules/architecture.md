# Platform Architecture

## Overview
Cloud backup platform wrapping Temporal for job orchestration.

## Temporal Clusters
- **temporal-public**: customized (`temporal-server/`), accessible by agents only
- **temporal-private**: stock open-source Temporal, accessible by workers only
- Backend has access to both clusters

## Job Types
- **Agent job**: initiated by the agent, runs on temporal-public
- **Worker job (cloud job)**: runs on temporal-private via the worker
- **Manual job**: triggered manually (e.g. via sctl or dashboard)

## Auth
- Auth0 with two separate databases:
  - One for **users** (dashboard/sctl access)
  - One for **agents** (agent authentication)

## SDK Generation
- OpenAPI spec lives at `openapi/`
- SDKs are generated from the spec: `sdk-go/`, `sdk-python/`, `sdk-typescript/`
- SDKs are consumed by `dashboard/` and `sctl/`

## Vault Access Model
- Backend writes secrets to Vault
- Worker reads secrets from Vault (in-cluster only)
- Agent has no direct Vault access — must call backend API

## Components
- Backend + Worker are a single Go project in `backend/`
- Backend accesses both Temporal clusters
- Worker entry point: `backend/cmd/worker/main.go`
