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
	AdminUser           string
	AdminPassword       string
	MaxLoginAttempts    int
	LockDurationMinutes int
}

func Load() *Config {
	port, _ := strconv.Atoi(os.Getenv("PORT"))
	mysqlPort, _ := strconv.Atoi(os.Getenv("MYSQL_PORT"))
	maxAttempts, _ := strconv.Atoi(os.Getenv("MAX_LOGIN_ATTEMPTS"))
	lockDuration, _ := strconv.Atoi(os.Getenv("LOCK_DURATION_MINUTES"))

	if maxAttempts == 0 {
		maxAttempts = 5
	}
	if lockDuration == 0 {
		lockDuration = 30
	}

	return &Config{
		ServerPort:          port,
		MySQLHost:           os.Getenv("MYSQL_HOST"),
		MySQLUser:           os.Getenv("MYSQL_USER"),
		MySQLPassword:       os.Getenv("MYSQL_PASSWORD"),
		MySQLDatabase:       os.Getenv("MYSQL_DATABASE"),
		MySQLPort:           mysqlPort,
		AdminUser:           os.Getenv("ADMIN_USER"),
		AdminPassword:       os.Getenv("ADMIN_PASSWORD"),
		MaxLoginAttempts:    maxAttempts,
		LockDurationMinutes: lockDuration,
	}
}