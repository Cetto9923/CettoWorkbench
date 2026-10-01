// =============================================================================
// 文件: internal/module/po/repo_fallback_test.go
// 模块: PO 工作台
// 职责: 验证未配置只读池时首页查询使用主库，配置时仍保留独立读池。
// =============================================================================
package po

import (
	"context"
	"gorm.io/gorm"
	"testing"
)

func TestHomeQueryWithoutReadPool(t *testing.T) {
	primary, _ := openSQLMock(t)
	query, err := NewRepo(nil, primary).allStageRefQuery(context.Background(), "alice", DemandsReq{})
	if err != nil {
		t.Fatal(err)
	}
	var rows []struct{ ID int }
	stmt := primary.Table("(?) AS stages", query).Session(&gorm.Session{DryRun: true}).Find(&rows).Statement
	if stmt.Error != nil || stmt.SQL.Len() == 0 {
		t.Fatalf("home query unavailable: %v", stmt.Error)
	}
	read, _ := openSQLMock(t)
	repo := NewRepo(read, primary)
	if repo.db != read || repo.writeDB != primary {
		t.Fatal("configured read/write pools changed")
	}
}
