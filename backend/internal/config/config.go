package config

import (
	"net"
	"net/url"
	"os"
	"strconv"
)

type Config struct {
	ServerPort string
	DBHost     string
	DBPort     int
	DBName     string
	DBUser     string
	DBPassword string
	DBSSLMode  string

	AuthUsername        string
	AuthPasswordHash   string
	JWTSecret          string
	TurnstileSecretKey string
}

func (c *Config) AuthEnabled() bool {
	return c.AuthUsername != "" && c.AuthPasswordHash != "" && c.JWTSecret != ""
}

func Load() *Config {
	return &Config{
		ServerPort: getEnv("SERVER_PORT", "8080"),
		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnvInt("DB_PORT", 5432),
		DBName:     getEnv("DB_NAME", "budgetapp"),
		DBUser:     getEnv("DB_USER", "budget"),
		DBPassword: getEnv("DB_PASSWORD", "budget_local_dev"),
		DBSSLMode:  getEnv("DB_SSLMODE", "disable"),

		AuthUsername:        getEnv("AUTH_USERNAME", ""),
		AuthPasswordHash:   getEnv("AUTH_PASSWORD_HASH", ""),
		JWTSecret:          getEnv("JWT_SECRET", ""),
		TurnstileSecretKey: getEnv("TURNSTILE_SECRET_KEY", ""),
	}
}

// DatabaseURL builds the connection DSN. It goes through net/url rather than
// fmt.Sprintf so that a password containing URL-significant characters
// (@ / ? : #) is percent-encoded instead of corrupting the URL structure.
func (c *Config) DatabaseURL() string {
	u := &url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(c.DBUser, c.DBPassword),
		Host:   net.JoinHostPort(c.DBHost, strconv.Itoa(c.DBPort)),
		Path:   "/" + c.DBName,
	}
	u.RawQuery = url.Values{"sslmode": {c.DBSSLMode}}.Encode()
	return u.String()
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return fallback
}
