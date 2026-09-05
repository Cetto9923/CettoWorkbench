package config

import (
	"strings"
	"testing"
)

const reviewValidConfig = `app:
  env: dev
database:
  host: localhost
  port: 3306
  user: synthetic
  dbname: synthetic
`

func TestConfigInvalidInputDoesNotEchoValues(t *testing.T) {
	const canary = "CANARY_CONFIG_BAD_INPUT"
	for _, key := range []string{"WORKBENCH_DATABASE_PORT", "WORKBENCH_APP_ENV"} {
		t.Run(key, func(t *testing.T) {
			t.Setenv(key, canary)
			_, err := LoadFromPath(createTestConfigFile(t, reviewValidConfig))
			if err == nil {
				t.Fatal("invalid input accepted")
			}
			if strings.Contains(err.Error(), canary) {
				t.Fatal("invalid configuration value leaked into error")
			}
		})
	}
}

func TestOptionalReadonlyConfigDoesNotBlockStartup(t *testing.T) {
	// Bootstrap deliberately degrades if the optional replica cannot connect.
	_, err := LoadFromPath(createTestConfigFile(t, reviewValidConfig+"databaseReadonly:\n  host: localhost\n"))
	if err != nil {
		t.Fatalf("optional replica rejected before bootstrap fallback: %v", err)
	}
}
