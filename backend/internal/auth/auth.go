package auth

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

const CookieName = "auth_token"

func VerifyPassword(hash, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}

func CreateToken(secret, username string, expiry time.Duration) (string, time.Time, error) {
	exp := time.Now().Add(expiry)
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": username,
		"exp": jwt.NewNumericDate(exp),
		"iat": jwt.NewNumericDate(time.Now()),
	})
	signed, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", time.Time{}, err
	}
	return signed, exp, nil
}

func ValidateToken(secret, tokenStr string) (string, error) {
	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(secret), nil
	})
	if err != nil {
		return "", err
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return "", fmt.Errorf("invalid token")
	}
	sub, _ := claims.GetSubject()
	if sub == "" {
		return "", fmt.Errorf("missing subject")
	}
	return sub, nil
}

type turnstileResponse struct {
	Success bool `json:"success"`
}

func VerifyTurnstile(secretKey, token, remoteIP string) error {
	resp, err := http.PostForm("https://challenges.cloudflare.com/turnstile/v0/siteverify", url.Values{
		"secret":   {secretKey},
		"response": {token},
		"remoteip": {remoteIP},
	})
	if err != nil {
		return fmt.Errorf("turnstile request failed: %w", err)
	}
	defer resp.Body.Close()

	var result turnstileResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("turnstile response decode failed: %w", err)
	}
	if !result.Success {
		return fmt.Errorf("turnstile verification failed")
	}
	return nil
}
