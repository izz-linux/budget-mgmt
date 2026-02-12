.PHONY: dev dev-backend dev-frontend build build-backend build-frontend docker-build docker-up docker-down k8s-deploy k8s-delete test

# ---- Development ----

dev: docker-up
	@echo "Starting development environment..."
	@echo "Backend:  http://localhost:8080"
	@echo "Frontend: http://localhost:3000"

dev-backend:
	cd backend && go run -buildvcs=false ./cmd/server

dev-frontend:
	cd frontend && npm run dev

# ---- Build ----

build: build-backend build-frontend

build-backend:
	cd backend && CGO_ENABLED=0 go build -buildvcs=false -ldflags="-s -w" -o ../bin/budget-api ./cmd/server

build-frontend:
	cd frontend && npm run build

# ---- Docker ----

docker-build:
	docker build -t budget-api:latest ./backend
	docker build -t budget-ui:latest ./frontend

docker-up:
	docker-compose up -d

docker-down:
	docker-compose down

docker-logs:
	docker-compose logs -f

# ---- Kubernetes ----

k8s-deploy: docker-build
	kubectl apply -f k8s/namespace.yaml
	kubectl apply -f k8s/postgres/
	kubectl apply -f k8s/backend/
	kubectl apply -f k8s/frontend/
	@echo ""
	@echo "Deployed to namespace: budget-app"
	@echo "Access via: http://localhost:30080 (NodePort)"
	@echo "Or add 'budget.local' to /etc/hosts and use Ingress"

k8s-delete:
	kubectl delete namespace budget-app --ignore-not-found

k8s-status:
	kubectl -n budget-app get pods,svc,ingress

# ---- Test ----

test:
	cd backend && go test -buildvcs=false ./...

# ---- Clean ----

clean:
	rm -rf bin/
	rm -rf frontend/dist/
	docker-compose down -v
