// =============================================================================
// 文件: internal/module/po/serviceparallel_test.go
// 模块: 产品经理工作台
// 类型: test
// 职责: 验证并发 panic 被转为错误且不阻塞其他取数任务。
// =============================================================================
package po

import (
	"errors"
	"sync"
	"testing"
)

func TestParallelQueryPanicIsolated(t *testing.T) {
	var wg sync.WaitGroup
	var panicErr, normalErr error
	result := 0
	runQuerySafely(&wg, &panicErr, func() { panic("query failure") })
	runQuerySafely(&wg, &normalErr, func() { result = 7 })
	wg.Wait()
	if panicErr == nil || normalErr != nil || result != 7 {
		t.Fatalf("panic=%v normal=%v result=%d", panicErr, normalErr, result)
	}
}

func TestParallelQueryRetainsReturnedError(t *testing.T) {
	var wg sync.WaitGroup
	want := errors.New("query error")
	var got error
	runQuerySafely(&wg, &got, func() { got = want })
	wg.Wait()
	if !errors.Is(got, want) {
		t.Fatalf("error lost: %v", got)
	}
}
