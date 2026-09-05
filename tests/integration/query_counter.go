//go:build integration

// =============================================================================
// 文件: tests/integration/query_counter.go
// 模块: 性能基线测试
// 类型: test
// 职责: 通过 GORM 回调提供零侵入的 SQL 执行次数统计。
// 依赖: gorm.io/gorm
// =============================================================================

package integration

import (
	"sync/atomic"

	"gorm.io/gorm"
)

// QueryCounter 统计 GORM 会话执行的 SQL 语句总数。
type QueryCounter struct {
	count int64
}

// AttachQueryCounter 为 GORM DB 实例附加查询计数回调。
func AttachQueryCounter(db *gorm.DB) (*gorm.DB, *QueryCounter) {
	qc := &QueryCounter{}
	callback := db.Callback()

	_ = callback.Query().Before("perf:count_before").Register("perf:count_query", func(d *gorm.DB) {
		atomic.AddInt64(&qc.count, 1)
	})
	_ = callback.Row().Before("perf:count_row_before").Register("perf:count_row", func(d *gorm.DB) {
		atomic.AddInt64(&qc.count, 1)
	})
	_ = callback.Raw().Before("perf:count_raw_before").Register("perf:count_raw", func(d *gorm.DB) {
		atomic.AddInt64(&qc.count, 1)
	})

	return db, qc
}

// Reset 重置计数器。
func (qc *QueryCounter) Reset() {
	atomic.StoreInt64(&qc.count, 0)
}

// Count 获取当前统计的 SQL 数量。
func (qc *QueryCounter) Count() int64 {
	return atomic.LoadInt64(&qc.count)
}
