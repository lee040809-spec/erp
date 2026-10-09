package config

import (
	"os"
	"strconv"
)

type Config struct {
	ServerPort          int
	MySQLHost           string
	MySQLUser           string
	MySQLPassword       string
	MySQLDatabase       string
	MySQLPort           int
	MaxLoginAttempts    int
	LockDurationMinutes int
	JWTSecret           string
	JWTExpiresHours     int
	JWTIssuer           string
	SMTPHost            string
	SMTPPort            string
	SMTPUsername        string
	SMTPPassword        string
	FromEmail           string
}

func Load() *Config {
	return &Config{
		ServerPort:          getEnvInt("PORT", 8080),
		MySQLHost:           getEnv("MYSQL_HOST", "localhost"),
		MySQLUser:           getEnv("MYSQL_USER", "root"),
		MySQLPassword:       getEnv("MYSQL_PASSWORD", ""),
		MySQLDatabase:       getEnv("MYSQL_DATABASE", "erp"),
		MySQLPort:           getEnvInt("MYSQL_PORT", 3306),
		MaxLoginAttempts:    getEnvInt("MAX_LOGIN_ATTEMPTS", 5),
		LockDurationMinutes: getEnvInt("LOCK_DURATION_MINUTES", 120),
		JWTSecret:           getEnv("JWT_SECRET", "yhs-erp-jwt-secret-key-2026"),
		JWTExpiresHours:     getEnvInt("JWT_EXPIRES_HOURS", 24),
		JWTIssuer:           getEnv("JWT_ISSUER", "yhs-erp"),
		SMTPHost:            getEnv("SMTP_HOST", ""),
		SMTPPort:            getEnv("SMTP_PORT", "587"),
		SMTPUsername:        getEnv("SMTP_USERNAME", ""),
		SMTPPassword:        getEnv("SMTP_PASSWORD", ""),
		FromEmail:           getEnv("FROM_EMAIL", ""),
	}
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func getEnvInt(key string, defaultVal int) int {
	if val := os.Getenv(key); val != "" {
		if i, err := strconv.Atoi(val); err == nil {
			return i
		}
	}
	return defaultVal
}
