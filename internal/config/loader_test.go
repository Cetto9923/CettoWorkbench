package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func createTestConfigFile(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.test.yaml")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("failed to write test config file: %v", err)
	}
	return path
}

// TestConfig_EnvOverridesFile 验证环境变量优先覆盖配置文件中的对应配置项。
func TestConfig_EnvOverridesFile(t *testing.T) {
	yamlContent := `
app:
  name: synthetic-app-A
  env: dev
database:
  host: 192.168.1.10
  port: 3306
  user: user-A
  password: pwd-A
  dbname: db-A
session:
  cookieName: sid_A
  lifetimeHours: 2
  cookieSecure: false
`
	configPath := createTestConfigFile(t, yamlContent)

	t.Setenv("WORKBENCH_APP_NAME", "synthetic-app-B")
	t.Setenv("WORKBENCH_DATABASE_HOST", "10.0.0.99")
	t.Setenv("WORKBENCH_DATABASE_PORT", "3307")
	t.Setenv("WORKBENCH_SESSION_LIFETIMEHOURS", "8")

	cfg, err := LoadFromPath(configPath)
	if err != nil {
		t.Fatalf("LoadFromPath failed: %v", err)
	}

	if cfg.App.Name != "synthetic-app-B" {
		t.Fatalf("App.Name = %q, want %q", cfg.App.Name, "synthetic-app-B")
	}
	if cfg.Database.Host != "10.0.0.99" {
		t.Fatalf("Database.Host = %q, want %q", cfg.Database.Host, "10.0.0.99")
	}
	if cfg.Database.Port != 3307 {
		t.Fatalf("Database.Port = %d, want %d", cfg.Database.Port, 3307)
	}
	if cfg.Session.LifetimeHours != 8 {
		t.Fatalf("Session.LifetimeHours = %d, want %d", cfg.Session.LifetimeHours, 8)
	}
}

// TestConfig_ModesAgree 验证 prod 与 local/dev 环境下的配置验证与安全约束一致性。
func TestConfig_ModesAgree(t *testing.T) {
	// Case A: prod 正确配置
	prodValid := `
app:
  name: prod-app
  env: prod
database:
  host: 10.0.0.1
  port: 3306
  user: prod-user
  password: prod-password
  dbname: prod-db
session:
  cookieName: sid_prod
  lifetimeHours: 2
  cookieSecure: true
`
	pathValid := createTestConfigFile(t, prodValid)
	cfg, err := LoadFromPath(pathValid)
	if err != nil {
		t.Fatalf("valid prod config should pass: %v", err)
	}
	if cfg.App.Env != "prod" || !cfg.Session.CookieSecure {
		t.Fatalf("unexpected prod config state: env=%q, cookieSecure=%v", cfg.App.Env, cfg.Session.CookieSecure)
	}

	// Case B: prod 下 cookieSecure 为 false 必须被拒绝
	prodInsecureCookie := `
app:
  name: prod-app
  env: prod
database:
  host: 10.0.0.1
  port: 3306
  user: prod-user
  password: prod-password
  dbname: prod-db
session:
  cookieName: sid_prod
  lifetimeHours: 2
  cookieSecure: false
`
	pathInsecureCookie := createTestConfigFile(t, prodInsecureCookie)
	_, err = LoadFromPath(pathInsecureCookie)
	if err == nil {
		t.Fatal("prod with cookieSecure=false must be rejected")
	}
	if !strings.Contains(err.Error(), "cookieSecure") {
		t.Fatalf("error should mention cookieSecure, got: %v", err)
	}

	// Case C: local 下 cookieSecure 为 false 允许通过
	localValid := `
app:
  name: local-app
  env: local
database:
  host: 127.0.0.1
  port: 3306
  user: root
  password: ""
  dbname: test_db
session:
  cookieName: sid_local
  lifetimeHours: 2
  cookieSecure: false
`
	pathLocal := createTestConfigFile(t, localValid)
	cfgLocal, err := LoadFromPath(pathLocal)
	if err != nil {
		t.Fatalf("valid local config should pass: %v", err)
	}
	if cfgLocal.Session.CookieSecure != false {
		t.Fatalf("local cookieSecure should be false, got: %v", cfgLocal.Session.CookieSecure)
	}
}

// TestConfig_MissingRequired_InvalidValue 验证缺必填项或非法值时拒绝启动且不泄露秘密。
func TestConfig_MissingRequired_InvalidValue(t *testing.T) {
	secretCanary := "CANARY_CONFIDENTIAL_TEST_PASSWORD_999"

	cases := []struct {
		name        string
		yaml        string
		errContains string
	}{
		{
			name: "missing database.host",
			yaml: `
app:
  env: dev
database:
  port: 3306
  user: root
  password: ` + secretCanary + `
  dbname: test
session:
  cookieName: sid
  lifetimeHours: 2
`,
			errContains: "database.host",
		},
		{
			name: "invalid database.port",
			yaml: `
app:
  env: dev
database:
  host: 127.0.0.1
  port: 99999
  user: root
  password: ` + secretCanary + `
  dbname: test
session:
  cookieName: sid
  lifetimeHours: 2
`,
			errContains: "database.port",
		},
		{
			name: "missing app.env",
			yaml: `
app:
  name: test
database:
  host: 127.0.0.1
  port: 3306
  user: root
  password: ` + secretCanary + `
  dbname: test
session:
  cookieName: sid
  lifetimeHours: 2
`,
			errContains: "app.env",
		},
		{
			name: "invalid app.env",
			yaml: `
app:
  env: invalid_environment
database:
  host: 127.0.0.1
  port: 3306
  user: root
  password: ` + secretCanary + `
  dbname: test
session:
  cookieName: sid
  lifetimeHours: 2
`,
			errContains: "invalid app.env",
		},
		{
			name: "prod empty password",
			yaml: `
app:
  env: prod
database:
  host: 10.0.0.1
  port: 3306
  user: root
  password: ""
  dbname: test
session:
  cookieName: sid
  lifetimeHours: 2
  cookieSecure: true
`,
			errContains: "database.password",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			path := createTestConfigFile(t, tc.yaml)
			_, err := LoadFromPath(path)
			if err == nil {
				t.Fatalf("%s: expected validation error, got nil", tc.name)
			}
			errStr := err.Error()
			if !strings.Contains(errStr, tc.errContains) {
				t.Fatalf("%s: error %q should contain %q", tc.name, errStr, tc.errContains)
			}
			// 确保任何错误都不包含合成秘密明文
			if strings.Contains(errStr, secretCanary) {
				t.Fatalf("%s: error leaked secret canary: %q", tc.name, errStr)
			}
		})
	}
}
