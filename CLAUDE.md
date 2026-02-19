# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Budget Management Application - A full-stack budget tracking and optimization app for managing bills, income sources, and pay period assignments. Users can track recurring bills, optimize payment scheduling, and import budget data via Excel.

## Tech Stack

- **Backend:** Go 1.24, Chi v5 router, PostgreSQL 16 (pgx v5), JWT auth
- **Frontend:** React 19, TypeScript, Vite, Zustand (state), TanStack Query (data fetching)
- **Infrastructure:** Docker, Docker Compose, Kubernetes

## Common Commands

```bash
# Development
make dev              # Start full Docker dev environment (postgres, backend, frontend)
make dev-backend      # Run backend locally with hot reload
make dev-frontend     # Run frontend with Vite dev server (port 5173)

# Testing
make test             # Run Go tests with race detection
cd backend && go test ./...                    # All backend tests
cd backend && go test -v ./internal/services/  # Specific package tests

# Building
make build            # Build both backend and frontend
make docker-build     # Build Docker images

# Database
docker compose up -d postgres  # Start just PostgreSQL for local dev
```

## Architecture

```
backend/
├── cmd/server/main.go           # Entry point - loads config, runs migrations, starts server
├── internal/
│   ├── router/router.go         # All API routes defined here
│   ├── handlers/                 # HTTP handlers (one per resource)
│   ├── services/                 # Business logic
│   │   ├── optimizer.go         # Bill assignment optimization algorithm
│   │   ├── period_generator.go  # Pay period generation from income schedules
│   │   ├── xlsx_importer.go     # Excel import processing
│   │   └── surplus_detector.go  # Find periods with surplus funds
│   ├── models/                   # Data structures and response helpers
│   ├── db/                       # Database connection and migrations
│   │   └── migrations/           # SQL migration files (auto-applied on startup)
│   └── config/                   # Environment variable configuration

frontend/
├── src/
│   ├── App.tsx                  # Router setup with protected routes
│   ├── api/client.ts            # Fetch-based API client
│   ├── stores/                  # Zustand stores (auth, budget, ui)
│   ├── components/              # React components by feature
│   └── types/                   # TypeScript interfaces
├── vite.config.ts               # Dev server config with /api proxy to backend
```

## Key Patterns

**Backend handler pattern:**
```go
type BillHandler struct { db DBTX }
func NewBillHandler(db DBTX) *BillHandler { ... }
func (h *BillHandler) List(w http.ResponseWriter, r *http.Request) { ... }
```

**API response format:**
```json
{"data": [...], "error": null}
```

**Frontend API calls:** Use TanStack Query with the API client in `src/api/client.ts`

**Database migrations:** Embedded SQL files in `backend/internal/db/migrations/`, auto-applied on server startup

## API Base Path

All endpoints are under `/api/v1/`. Key resources: `/bills`, `/income-sources`, `/pay-periods`, `/assignments`, `/budget-grid`, `/optimizer`, `/import/xlsx`

## Database

PostgreSQL with these main tables: `bills`, `income_sources`, `pay_periods`, `bill_assignments`, `credit_cards`, `import_history`, `app_settings`

Default local credentials: user=`budget`, password=`budget_local_dev`, db=`budgetapp`

## Development URLs

- Frontend (Docker): http://localhost:3000
- Frontend (Vite dev): http://localhost:5173
- Backend API: http://localhost:8080
