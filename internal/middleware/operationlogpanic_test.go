// =============================================================================
// 文件: internal/middleware/operationlogpanic_test.go
// 模块: 操作日志
// 类型: test
// 职责: 验证后台写日志的 panic 不会越过协程边界。
// =============================================================================
package middleware

import (
	"testing"
	"workbench/internal/model"
)

func TestOperationLogPanicIsolated(t *testing.T) {
	done := make(chan error, 1)
	go func() { done <- saveOperationLogSafely(nil, model.OperationLog{}) }()
	if err := <-done; err == nil {
		t.Fatal("panic must return an error")
	}
}
