# Engineering Specification Index

This document provides a single canonical directory of the engineering specifications and architectural boundaries governing the Workbench codebase.

> [!IMPORTANT]
> The single canonical engineering truth source is `AGENTS.md` at repository root. Tool adapters (Gemini, Claude, Cursor, Codex) may summarize or load it, but MUST NOT redefine or weaken its boundaries.

## 1. Hierarchy of Engineering Truth

```
AGENTS.md (Root Constitution)
  ├── docs/engineering/spec-index.md (This Directory)
  ├── docs/engineering/architecture.md (Layers, Services, Repos, Boundaries)
  ├── docs/engineering/database.md (SQL Standards, Schema Ownership, Conventions)
  ├── docs/engineering/frontend.md (Write Protocol, Reusable UI, DOM Contracts)
  ├── docs/engineering/testing.md (Test Classification, DB Isolation, Fixtures)
  ├── docs/engineering/quality.md (Quality Gates, Baselines, Self-Locking)
  ├── docs/engineering/module-index.md (Module Inventory & Responsibility Map)
  └── docs/engineering/debt.md (Explicit Technical Debt Register)
```

## 2. Specification Map and Machine Gates

| Domain | Specification File | Governed MUST Rules | Enforcing Gate / Command |
|---|---|---|---|
| **Root Constitution** | `AGENTS.md` | MUST 1 ~ 12 | `make check`, `bash scripts/test-quality-gates.sh` |
| **Architecture** | `docs/engineering/architecture.md` | MUST 1 (Layers), MUST 2 (Module Shape), MUST 3 (Object Auth) | `scripts/check-architecture.sh` |
| **Database** | `docs/engineering/database.md` | MUST 5 (Queries), MUST 6 (Schema Ownership) | `scripts/check-patterns.sh` (`SQL_WILDCARD`) |
| **Frontend** | `docs/engineering/frontend.md` | MUST 4 (Write Protocol), MUST 9 (Shared UI) | `make check-frontend-test` |
| **Testing** | `docs/engineering/testing.md` | MUST 10 (Test Hierarchy & Isolation) | `go test ./...`, `tests/integration/db_safety.go` |
| **Quality & Secrets** | `docs/engineering/quality.md` | MUST 7 (Secrets), MUST 8 (500-Line Limit), MUST 12 (Delivery Truth) | `scripts/check-file-length.sh`, `scripts/check-secrets.sh`, `scripts/check-go-vet.sh` |
| **Module Inventory** | `docs/engineering/module-index.md` | Responsibility Boundaries | Source inspection & import boundaries |
| **Debt Register** | `docs/engineering/debt.md` | Phase 2 Governance & Debt Recording | Quality baseline tracking (`scripts/quality-baseline/`) |

## 3. Agent Onboarding & Verification

All AI agents must follow [agent-onboarding.md](agent-onboarding.md) on session start:
- Inspect repository state (`git rev-parse`, safe branch).
- Read applicable specifications and record actual files loaded.
- Execute machine gates before claiming any completion.
