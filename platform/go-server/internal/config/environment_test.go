package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDevelopmentEnvironmentFile(t *testing.T) {
	t.Chdir(t.TempDir())
	t.Setenv("GO_ENV", "local")
	t.Setenv("ERP_CONFIG_TEST_FILE", "")
	_ = os.Unsetenv("ERP_CONFIG_TEST_FILE")
	t.Setenv("ERP_CONFIG_TEST_OVERRIDE", "system")
	if err := os.WriteFile(".env", []byte("ERP_CONFIG_TEST_FILE=from_file\nERP_CONFIG_TEST_OVERRIDE=from_file\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := Initialize(); err != nil {
		t.Fatal(err)
	}
	if os.Getenv("ERP_CONFIG_TEST_FILE") != "from_file" || os.Getenv("ERP_CONFIG_TEST_OVERRIDE") != "system" {
		t.Fatal("file loading or explicit environment precedence failed")
	}
}

func TestProductionDoesNotReadDevelopmentFile(t *testing.T) {
	if embeddedProduction != "" {
		t.Skip("default-build missing-production guard only")
	}
	t.Chdir(t.TempDir())
	t.Setenv("GO_ENV", "production")
	if err := os.WriteFile(".env", []byte("MYSQL_PASSWORD=should_not_be_loaded\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := Initialize(); err == nil {
		t.Fatal("production silently fell back to development file")
	}
}

func TestConfigurationErrorsDoNotExposeValues(t *testing.T) {
	t.Chdir(t.TempDir())
	t.Setenv("GO_ENV", "local")
	secret := "private-value-not-for-logs"
	if err := os.WriteFile(filepath.Join(".env"), []byte("INVALID@"+secret+"=x"), 0600); err != nil {
		t.Fatal(err)
	}
	_, err := Initialize()
	if err == nil || strings.Contains(err.Error(), secret) {
		t.Fatal("parse error must omit file contents")
	}
}

func TestProductionValidation(t *testing.T) {
	values := map[string]string{"GO_ENV": "production", "PORT": "28080", "MYSQL_HOST": "localhost", "MYSQL_PORT": "3306", "MYSQL_USER": "app", "MYSQL_PASSWORD": "fixture-only", "MYSQL_DATABASE": "fixture", "JWT_SECRET": strings.Repeat("x", 32), "DRAWING_STORAGE_DIR": "/var/lib/erp/drawings"}
	if err := ValidateProduction(values); err != nil {
		t.Fatal(err)
	}
	values["DRAWING_STORAGE_DIR"] = "F:/data"
	if ValidateProduction(values) == nil {
		t.Fatal("Windows path accepted for Linux release")
	}
	values["DRAWING_STORAGE_DIR"] = "/var/lib/erp/drawings"
	values["SMTP_HOST"] = "smtp.example.invalid"
	if ValidateProduction(values) == nil {
		t.Fatal("partial SMTP settings accepted")
	}
}

func TestEmbeddedDefaultsAndPartialProductionOverride(t *testing.T) {
	t.Chdir(t.TempDir())
	const defaults = "MYSQL_USER=embedded_user\nMYSQL_PASSWORD=fixture_only\nSMTP_HOST=embedded_mail\n"
	values, err := readConfiguration(".env.production", defaults)
	if err != nil || values["MYSQL_USER"] != "embedded_user" {
		t.Fatal("embedded defaults require no external production file")
	}
	if err := os.WriteFile(".env.production", []byte("MYSQL_USER=external_user\nSMTP_HOST=\n"), 0600); err != nil {
		t.Fatal(err)
	}
	values, err = readConfiguration(".env.production", defaults)
	if err != nil || values["MYSQL_USER"] != "external_user" || values["MYSQL_PASSWORD"] != "fixture_only" || values["SMTP_HOST"] != "" {
		t.Fatal("external values must override only specified embedded defaults, including explicit empty values")
	}
}
