# Budget Management Application

A full-stack budget tracking and optimization application for managing bills, income sources, and pay period assignments.

## Tech Stack

- **Backend**: Go 1.24 with Chi router, PostgreSQL (pgx driver)
- **Frontend**: React 19 + TypeScript, Vite, TanStack Query, Zustand, Recharts
- **Database**: PostgreSQL 16
- **Containerization**: Docker, Docker Compose, Kubernetes

## Project Structure

```
budget-mgmt/
├── backend/                 # Go API server
│   ├── cmd/server/          # Application entry point
│   └── internal/            # Business logic, handlers, models
├── frontend/                # React SPA
│   └── src/
│       ├── api/             # API client
│       ├── components/      # React components
│       ├── hooks/           # Custom hooks
│       ├── stores/          # Zustand state stores
│       └── types/           # TypeScript definitions
├── k8s/                     # Kubernetes manifests
│   ├── namespace.yaml
│   ├── postgres/
│   ├── backend/
│   └── frontend/
├── docker-compose.yaml
└── Makefile
```

## Quick Start

### Option 1: Docker Compose (Recommended for Development)

```bash
# Start all services (postgres, backend, frontend)
make dev

# View logs
make docker-logs

# Stop services
make docker-down
```

Access the application at http://localhost:3000

### Option 2: Local Development (Hot Reload)

**Prerequisites:**
- Go 1.24+
- Node.js 22+
- PostgreSQL 16+ (running locally or via Docker)

```bash
# Start database only
docker compose up -d postgres

# Run backend with hot reload (in terminal 1)
make dev-backend

# Run frontend with Vite (in terminal 2)
make dev-frontend
```

- Frontend: http://localhost:5173
- Backend API: http://localhost:8080

### Option 3: Build and Run Binaries

```bash
# Build both backend and frontend
make build

# Run the backend binary (requires PostgreSQL)
./bin/budget-api

# Serve frontend/dist with your preferred static server
```

## Makefile Commands

| Command | Description |
|---------|-------------|
| `make dev` | Start full Docker development environment |
| `make dev-backend` | Run backend locally with hot reload |
| `make dev-frontend` | Run frontend with Vite dev server |
| `make build` | Build backend binary and frontend bundle |
| `make build-backend` | Compile Go binary to `bin/budget-api` |
| `make build-frontend` | Build React app to `frontend/dist/` |
| `make docker-build` | Build Docker images |
| `make docker-up` | Start Docker Compose services |
| `make docker-down` | Stop and remove containers |
| `make docker-logs` | Stream container logs |
| `make test` | Run Go tests with race detection |
| `make clean` | Remove build artifacts and volumes |
| `make k8s-deploy` | Deploy to Kubernetes |
| `make k8s-delete` | Delete Kubernetes deployment |
| `make k8s-status` | Check Kubernetes resource status |

## Kubernetes Deployment

### Prerequisites

- Kubernetes cluster (EKS, GKE, AKS, or self-managed)
- `kubectl` configured with cluster access
- Container registry accessible to the cluster (e.g., Docker Hub, ECR, GCR, ACR)

### Deploy

```bash
# Build and push Docker images to your registry
export REGISTRY=your-registry.com/your-org
docker build -t $REGISTRY/budget-api:latest ./backend
docker build -t $REGISTRY/budget-ui:latest ./frontend
docker push $REGISTRY/budget-api:latest
docker push $REGISTRY/budget-ui:latest

# Update image references in k8s manifests (or use kustomize/helm)
# k8s/backend/deployment.yaml and k8s/frontend/deployment.yaml

# Deploy all resources
make k8s-deploy
```

### Access

```bash
# Check deployment status
make k8s-status

# Frontend is exposed via NodePort on port 30080
# Access via any cluster node IP: http://<node-ip>:30080

# Or use kubectl port-forward for local access
kubectl -n budget-app port-forward svc/frontend 3000:80
# Then access at http://localhost:3000
```

### Delete

```bash
make k8s-delete
```

### Kubernetes Resources

All resources are deployed to the `budget-app` namespace:

| Component | Type | Access |
|-----------|------|--------|
| PostgreSQL | Deployment + PVC | ClusterIP (internal) |
| Backend API | Deployment | ClusterIP (internal) |
| Frontend | Deployment | NodePort 30080 |

## Environment Variables

### Backend Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `SERVER_PORT` | `8080` | API server port |
| `DB_HOST` | `localhost` | PostgreSQL hostname |
| `DB_PORT` | `5432` | PostgreSQL port |
| `DB_NAME` | `budgetapp` | Database name |
| `DB_USER` | `budget` | Database user |
| `DB_PASSWORD` | `budget_local_dev` | Database password |
| `DB_SSLMODE` | `disable` | PostgreSQL SSL mode |

### Docker Compose Defaults

The `docker-compose.yaml` includes development defaults. For production, override with environment files or secrets.

### Kubernetes Configuration

Kubernetes uses ConfigMaps and Secrets in `k8s/postgres/` and `k8s/backend/`:
- `postgres-config`: Database name and user
- `postgres-secret`: Database password
- `backend-config`: Connection settings

**Important:** Update secrets before production deployment.

## API Endpoints

Base URL: `/api/v1`

| Endpoint | Methods | Description |
|----------|---------|-------------|
| `/health` | GET | Health check |
| `/bills` | GET, POST | List/create bills |
| `/bills/{id}` | GET, PUT, DELETE | Bill operations |
| `/bills/reorder` | PATCH | Reorder bills |
| `/income-sources` | GET, POST | List/create income sources |
| `/income-sources/{id}` | GET, PUT, DELETE | Income source operations |
| `/pay-periods` | GET | List pay periods |
| `/pay-periods/generate` | POST | Generate pay periods |
| `/pay-periods/{id}` | PUT | Update pay period |
| `/assignments` | GET, POST | List/create bill assignments |
| `/assignments/{id}` | PUT, DELETE | Assignment operations |
| `/assignments/{id}/status` | PATCH | Update assignment status |
| `/budget-grid` | GET | Get budget grid view data |
| `/import/xlsx` | POST | Upload Excel file |
| `/import/xlsx/confirm` | POST | Confirm import |
| `/import/history` | GET | Get import history |
| `/optimizer/suggest` | POST | Get optimization suggestions |
| `/optimizer/surplus` | GET | Detect surplus funds |
| `/dashboard/summary` | GET | Dashboard summary data |

## Database Schema

The application uses the following tables:
- `bills` - Recurring bills and expenses
- `credit_cards` - Credit cards linked to bills
- `income_sources` - Income sources with pay schedules
- `pay_periods` - Individual paycheck dates
- `bill_assignments` - Maps bills to pay periods
- `import_history` - Excel import tracking
- `app_settings` - Application settings

Migrations run automatically on backend startup.

## Development Notes

### Frontend Proxy

During development, the Vite dev server proxies `/api` requests to `http://localhost:8080`.

### CORS

Backend allows requests from `localhost` and `127.0.0.1` on any port during development.

### File Uploads

Excel imports support files up to 10MB.

## Troubleshooting

### Database Connection Issues

```bash
# Check PostgreSQL is running
docker compose ps postgres

# View PostgreSQL logs
docker compose logs postgres

# Test connection
psql -h localhost -U budget -d budgetapp
```

### Kubernetes Pods Not Starting

```bash
# Check pod status and events
kubectl -n budget-app describe pods

# View pod logs
kubectl -n budget-app logs -l app=backend
kubectl -n budget-app logs -l app=postgres
```

### Frontend Can't Reach Backend

```bash
# Docker Compose: Ensure containers are on same network
docker compose ps

# Kubernetes: Check service endpoints
kubectl -n budget-app get endpoints
```

## License

This project is licensed under the [MIT License](LICENSE).
