# Auth Implementation Plan: Single-user + Cloudflare Turnstile + JWT

## Overview
Add login-gated access to the budget web app with:
- Single-user credentials configured via environment variables
- Cloudflare Turnstile captcha on the login form
- JWT stored in an httpOnly cookie for session management

---

## Backend Changes (Go)

### 1. Config — add auth env vars
**File**: `backend/internal/config/config.go`
- Add fields: `AuthUsername`, `AuthPasswordHash`, `JWTSecret`, `TurnstileSecretKey`
- Env vars: `AUTH_USERNAME`, `AUTH_PASSWORD_HASH`, `JWT_SECRET`, `TURNSTILE_SECRET_KEY`
- No defaults for auth fields (app starts without auth if not configured, or we require them)

### 2. Auth package — JWT + password + Turnstile verification
**New file**: `backend/internal/auth/auth.go`
- `VerifyPassword(hash, password string) error` — bcrypt compare
- `CreateToken(secret string, username string, expiry time.Duration) (string, time.Time, error)` — generate JWT with `exp` and `sub` claims
- `ValidateToken(secret string, tokenStr string) (string, error)` — parse/validate JWT, return username
- `VerifyTurnstile(secretKey, token, remoteIP string) error` — POST to `https://challenges.cloudflare.com/turnstile/v0/siteverify`, check `success` field

### 3. Auth middleware
**New file**: `backend/internal/auth/middleware.go`
- `RequireAuth(jwtSecret string) func(http.Handler) http.Handler` — Chi middleware
- Reads JWT from `auth_token` httpOnly cookie
- Validates token; returns 401 JSON error if invalid/missing/expired
- If auth is not configured (no `AUTH_USERNAME` set), middleware passes through (allows graceful no-auth mode)

### 4. Auth handler — login/logout/status endpoints
**New file**: `backend/internal/handlers/auth.go`
- `POST /api/v1/auth/login` — accepts `{ username, password, turnstileToken }`, verifies Turnstile, verifies password, sets httpOnly cookie with JWT (24h expiry)
- `POST /api/v1/auth/logout` — clears the auth cookie
- `GET /api/v1/auth/status` — returns `{ authenticated: true/false }` (checks cookie validity)

### 5. Router — wire up auth
**File**: `backend/internal/router/router.go`
- Accept `*config.Config` parameter (in addition to `*pgxpool.Pool`)
- Register auth routes OUTSIDE the protected group (public)
- Keep health check public
- Wrap all existing `/api/v1/*` data routes with `auth.RequireAuth` middleware
- Add `SameSite=Lax`, `Secure` (in prod), `HttpOnly`, `Path=/` on the cookie

### 6. Update main.go
**File**: `backend/cmd/server/main.go`
- Pass config to `router.New(pool, cfg)`

### 7. Go dependencies
- `github.com/golang-jwt/jwt/v5` — JWT library
- `golang.org/x/crypto` already in go.mod (for bcrypt)

---

## Frontend Changes (React/TypeScript)

### 8. Add Turnstile script to index.html
**File**: `frontend/index.html`
- Add `<script src="https://challenges.cloudflare.com/turnstile/v0/api.js" async defer></script>`

### 9. Auth store
**New file**: `frontend/src/stores/authStore.ts`
- Zustand store with: `isAuthenticated`, `isLoading`, `checkAuth()`, `login()`, `logout()`
- `checkAuth()` calls `GET /api/v1/auth/status` on app load
- `login()` calls `POST /api/v1/auth/login`
- `logout()` calls `POST /api/v1/auth/logout`

### 10. Login page
**New file**: `frontend/src/components/auth/LoginPage.tsx`
- Dark-themed login form matching existing design (CSS vars from globals.css)
- Username + password fields
- Cloudflare Turnstile widget (reads site key from `VITE_TURNSTILE_SITE_KEY` build-time env)
- Error display for invalid credentials or failed captcha
- Calls `authStore.login()` on submit

**New file**: `frontend/src/components/auth/LoginPage.css`
- Styles for the login page

### 11. Update API client
**File**: `frontend/src/api/client.ts`
- Add `credentials: 'include'` to all fetch calls (needed for cookies cross-origin in dev)
- On 401 response, redirect to `/login` (or trigger auth store)

### 12. Update App.tsx — protected routing
**File**: `frontend/src/App.tsx`
- Add `/login` route
- Wrap all existing routes in a `<ProtectedRoute>` component that checks `authStore.isAuthenticated`
- If not authenticated, redirect to `/login`
- Show loading spinner while `checkAuth()` is in progress

### 13. Add logout to AppShell
**File**: `frontend/src/components/layout/AppShell.tsx`
- Add a logout button/link in the header or sidebar
- Calls `authStore.logout()` then redirects to `/login`

---

## Deployment / Config Changes

### 14. Docker Compose
**File**: `docker-compose.yaml`
- Add env vars to backend service: `AUTH_USERNAME`, `AUTH_PASSWORD_HASH`, `JWT_SECRET`, `TURNSTILE_SECRET_KEY`
- Add build arg to frontend: `VITE_TURNSTILE_SITE_KEY`

### 15. K8s manifests
**File**: `k8s/backend/configmap.yaml` — add `AUTH_USERNAME`
- Auth secrets (`AUTH_PASSWORD_HASH`, `JWT_SECRET`, `TURNSTILE_SECRET_KEY`) should go in a K8s Secret
**New file**: `k8s/backend/auth-secret.yaml` — template for auth secrets
**File**: `k8s/backend/deployment.yaml` — reference the new secret

### 16. Vite config
**File**: `frontend/vite.config.ts` — no changes needed (env vars with `VITE_` prefix are auto-exposed)

---

## File Summary

| Action | File |
|--------|------|
| Edit | `backend/internal/config/config.go` |
| Create | `backend/internal/auth/auth.go` |
| Create | `backend/internal/auth/middleware.go` |
| Create | `backend/internal/handlers/auth.go` |
| Edit | `backend/internal/router/router.go` |
| Edit | `backend/cmd/server/main.go` |
| Edit | `backend/go.mod` (add jwt dep) |
| Edit | `frontend/index.html` |
| Create | `frontend/src/stores/authStore.ts` |
| Create | `frontend/src/components/auth/LoginPage.tsx` |
| Create | `frontend/src/components/auth/LoginPage.css` |
| Edit | `frontend/src/api/client.ts` |
| Edit | `frontend/src/App.tsx` |
| Edit | `frontend/src/components/layout/AppShell.tsx` |
| Edit | `docker-compose.yaml` |
| Edit | `k8s/backend/configmap.yaml` |
| Create | `k8s/backend/auth-secret.yaml` |
| Edit | `k8s/backend/deployment.yaml` |

---

## Env Vars Summary

| Variable | Where | Example |
|----------|-------|---------|
| `AUTH_USERNAME` | Backend | `admin` |
| `AUTH_PASSWORD_HASH` | Backend | `$2a$12$...` (bcrypt hash) |
| `JWT_SECRET` | Backend | random 32+ char string |
| `TURNSTILE_SECRET_KEY` | Backend | from Cloudflare dashboard |
| `VITE_TURNSTILE_SITE_KEY` | Frontend (build-time) | from Cloudflare dashboard |
