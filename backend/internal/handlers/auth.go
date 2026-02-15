package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/izz-linux/budget-mgmt/backend/internal/auth"
	"github.com/izz-linux/budget-mgmt/backend/internal/config"
)

const tokenExpiry = 24 * time.Hour

type AuthHandler struct {
	cfg *config.Config
}

func NewAuthHandler(cfg *config.Config) *AuthHandler {
	return &AuthHandler{cfg: cfg}
}

type loginRequest struct {
	Username       string `json:"username"`
	Password       string `json:"password"`
	TurnstileToken string `json:"turnstileToken"`
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"error": map[string]string{"message": "invalid request body"},
		})
		return
	}

	// Verify Turnstile if configured
	if h.cfg.TurnstileSecretKey != "" {
		if req.TurnstileToken == "" {
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"error": map[string]string{"message": "captcha verification required"},
			})
			return
		}
		if err := auth.VerifyTurnstile(h.cfg.TurnstileSecretKey, req.TurnstileToken, r.RemoteAddr); err != nil {
			writeJSON(w, http.StatusForbidden, map[string]any{
				"error": map[string]string{"message": "captcha verification failed"},
			})
			return
		}
	}

	// Verify credentials
	if req.Username != h.cfg.AuthUsername {
		writeJSON(w, http.StatusUnauthorized, map[string]any{
			"error": map[string]string{"message": "invalid credentials"},
		})
		return
	}
	if err := auth.VerifyPassword(h.cfg.AuthPasswordHash, req.Password); err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]any{
			"error": map[string]string{"message": "invalid credentials"},
		})
		return
	}

	// Create JWT
	token, exp, err := auth.CreateToken(h.cfg.JWTSecret, req.Username, tokenExpiry)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"error": map[string]string{"message": "failed to create token"},
		})
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     auth.CookieName,
		Value:    token,
		Path:     "/",
		Expires:  exp,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   r.TLS != nil,
	})

	writeJSON(w, http.StatusOK, map[string]any{
		"data": map[string]any{"authenticated": true},
	})
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     auth.CookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   r.TLS != nil,
	})

	writeJSON(w, http.StatusOK, map[string]any{
		"data": map[string]any{"authenticated": false},
	})
}

func (h *AuthHandler) Status(w http.ResponseWriter, r *http.Request) {
	if !h.cfg.AuthEnabled() {
		writeJSON(w, http.StatusOK, map[string]any{
			"data": map[string]any{"authenticated": true, "authRequired": false},
		})
		return
	}

	cookie, err := r.Cookie(auth.CookieName)
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]any{
			"data": map[string]any{"authenticated": false, "authRequired": true},
		})
		return
	}

	if _, err := auth.ValidateToken(h.cfg.JWTSecret, cookie.Value); err != nil {
		writeJSON(w, http.StatusOK, map[string]any{
			"data": map[string]any{"authenticated": false, "authRequired": true},
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"data": map[string]any{"authenticated": true, "authRequired": true},
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
