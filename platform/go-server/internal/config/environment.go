package config

import (
	"errors"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

// Initialize is shared by the server and maintenance commands. Production
// builds have embedded defaults; a selected file and explicit process variables
// can override them. Production never falls back to the development .env file.
func Initialize() (string, error) {
	mode := os.Getenv("GO_ENV")
	if mode == "" && embeddedProduction != "" {
		mode = "production"
	}
	file := ".env"
	if mode == "production" {
		file = ".env.production"
	}
	embedded := ""
	if mode == "production" {
		embedded = embeddedProduction
	}
	defaults, err := readConfiguration(file, embedded)
	if err != nil {
		return "", err
	}
	for key, value := range defaults {
		if _, exists := os.LookupEnv(key); !exists {
			if os.Setenv(key, value) != nil {
				return "", errors.New("cannot apply configuration key: " + key)
			}
		}
	}
	if _, exists := os.LookupEnv("GO_ENV"); !exists && mode != "" {
		_ = os.Setenv("GO_ENV", mode)
	}
	source := file
	if mode == "production" && embeddedProduction != "" {
		source = "embedded production defaults (external .env.production and process overrides supported)"
	}
	return source, nil
}

func readConfiguration(file, embedded string) (map[string]string, error) {
	defaults := map[string]string{}
	if embedded != "" {
		var err error
		defaults, err = godotenv.Unmarshal(embedded)
		if err != nil {
			return nil, errors.New("embedded production configuration is invalid")
		}
	}
	external, err := godotenv.Read(file)
	if err != nil && !os.IsNotExist(err) {
		// Dotenv parse errors can contain secret values; never return raw errors.
		return nil, errors.New("cannot read configuration file: " + file)
	}
	if err != nil && file == ".env.production" && embedded == "" {
		return nil, errors.New("production requires .env.production or an embedded production build")
	}
	for key, value := range external {
		defaults[key] = value
	}
	return defaults, nil
}

// ValidateProduction checks only configuration shape; it never connects to a
// database or SMTP server and never includes values in validation errors.
func ValidateProduction(values map[string]string) error {
	for _, key := range []string{"MYSQL_HOST", "MYSQL_PORT", "MYSQL_USER", "MYSQL_PASSWORD", "MYSQL_DATABASE", "JWT_SECRET", "DRAWING_STORAGE_DIR", "PORT"} {
		v := strings.TrimSpace(values[key])
		if v == "" || strings.HasPrefix(v, "replace_") || strings.HasPrefix(v, "your_") {
			return errors.New("production configuration missing: " + key)
		}
	}
	if len(values["JWT_SECRET"]) < 32 {
		return errors.New("JWT_SECRET must contain at least 32 characters")
	}
	if !strings.HasPrefix(values["DRAWING_STORAGE_DIR"], "/") {
		return errors.New("DRAWING_STORAGE_DIR must be an absolute Linux path")
	}
	for _, key := range []string{"PORT", "MYSQL_PORT"} {
		if !validPort(values[key]) {
			return errors.New("invalid port: " + key)
		}
	}
	if values["GO_ENV"] != "production" {
		return errors.New("GO_ENV must be production in the package configuration")
	}
	if values["SMTP_HOST"] != "" {
		for _, key := range []string{"SMTP_USERNAME", "SMTP_PASSWORD", "FROM_EMAIL"} {
			if strings.TrimSpace(values[key]) == "" {
				return errors.New("SMTP configuration incomplete: " + key)
			}
		}
		if !validPort(values["SMTP_PORT"]) {
			return errors.New("invalid port: SMTP_PORT")
		}
	}
	return nil
}
