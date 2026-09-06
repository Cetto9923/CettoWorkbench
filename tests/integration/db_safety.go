// =============================================================================
// 文件: tests/integration/db_safety.go
// 模块: 测试基础设施安全
// 类型: test-helper
// 职责: 在发起任何物理连接前，静态白名单校验测试 DSN 与执行环境，
//       防止测试用例误连开发库、测试非隔离库或生产库导致数据破坏。
// =============================================================================

package integration

import (
	"errors"
	"fmt"
	"net"
	"os"
	"strings"

	mysqldriver "github.com/go-sql-driver/mysql"
)

var (
	// ErrMissingDSN DSN 为空
	ErrMissingDSN = errors.New("WB_TEST_MYSQL_DSN is empty")
	// ErrUnsafeDatabase 库名非安全测试库
	ErrUnsafeDatabase = errors.New("database name is not an isolated test database")
	// ErrDisallowedHost 主机非本地回环地址
	ErrDisallowedHost = errors.New("test database host must be loopback (127.0.0.1 or localhost)")
	// ErrMissingDDLOptIn 缺少破坏性 DDL 显式授权
	ErrMissingDDLOptIn = errors.New("destructive DDL/seed requires WB_TEST_ALLOW_DDL=1 opt-in")
)

// ValidateTestDSN 静态验证测试 DSN 的安全边界，不发起任何物理网络连接。
func ValidateTestDSN(dsn string, requireDDL bool) error {
	trimmed := strings.TrimSpace(dsn)
	if trimmed == "" {
		return ErrMissingDSN
	}

	cfg, err := mysqldriver.ParseDSN(trimmed)
	if err != nil {
		return fmt.Errorf("invalid mysql dsn: %w", err)
	}

	dbName := strings.ToLower(strings.TrimSpace(cfg.DBName))
	if dbName == "" {
		return fmt.Errorf("%w: empty database name", ErrUnsafeDatabase)
	}

	// 1. 黑名单检查：绝不允许生产/禅道库
	blacklist := []string{"zentaopms", "prod", "production", "master", "uat", "release"}
	for _, b := range blacklist {
		if strings.Contains(dbName, b) {
			return fmt.Errorf("%w: database %q matches forbidden keyword %q", ErrUnsafeDatabase, cfg.DBName, b)
		}
	}

	// 2. 白名单检查：必须显式包含 _test 或为 test 结尾
	if !strings.Contains(dbName, "_test") && !strings.HasSuffix(dbName, "test") {
		return fmt.Errorf("%w: database %q must contain '_test'", ErrUnsafeDatabase, cfg.DBName)
	}

	// 3. 网络与主机检查：仅允许本地回环地址
	host := cfg.Addr
	if cfg.Net == "tcp" && host != "" {
		h, _, splitErr := net.SplitHostPort(host)
		if splitErr == nil {
			host = h
		}
		hostLower := strings.ToLower(strings.TrimSpace(host))
		if hostLower != "127.0.0.1" && hostLower != "localhost" && hostLower != "::1" {
			return fmt.Errorf("%w: target host %q is not loopback", ErrDisallowedHost, host)
		}
	}

	// 4. 破坏性操作 (DDL / Truncate / Seed) 显式授权校验
	if requireDDL {
		isCI := strings.EqualFold(os.Getenv("CI"), "true") || strings.EqualFold(os.Getenv("GITHUB_ACTIONS"), "true")
		hasOptIn := os.Getenv("WB_TEST_ALLOW_DDL") == "1"
		if !isCI && !hasOptIn {
			return ErrMissingDDLOptIn
		}
	}

	return nil
}
