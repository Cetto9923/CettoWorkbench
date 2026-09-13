// =============================================================================
// 文件: internal/module/po/repodone_approval_context_test.go
// 模块: PO 工作台
// 类型: test
// 职责: 回归审批对象（章程 / 建设指引 / 计划变更）的禅道跳转链接与上下文加载：
//       1. 章程 / 建设指引链接必须用所属 projectID，缺失上下文时不生成链接；
//       2. 上下文查询失败必须报错，不能被当成"对象不存在"静默降级。
// 依赖: github.com/DATA-DOG/go-sqlmock, gorm.io/gorm
// =============================================================================

package po

import (
	"errors"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"workbench/internal/config"
	"workbench/internal/pkg/zentao"
)

// 复现用户上报的 "项目建设指引 28 → 禅道首页" 链接形态：
// charter / buildguideline 的禅道 view 入口按项目打开，不能把对象 ID 当 projectID。
func TestApprovalObjectURLNeverEmitsZeroIDHomepageLink(t *testing.T) {
	zentao.SetConfig(config.ZentaoConfig{URL: "http://127.0.0.1:8080"})

	cases := []struct {
		objectType string
		objectID   uint
		projectID  uint
		wantID     string
	}{
		{"buildguideline", 28, 0, ""},
		{"charter", 261, 239, "projectID=239"},
		{"planchange", 21, 0, "id=21"},
	}
	for _, tc := range cases {
		url := objectViewURLWithProject(tc.objectType, tc.objectID, tc.projectID)
		if tc.wantID == "" {
			if url != "" {
				t.Fatalf("%s without project must not emit URL, got %q", tc.objectType, url)
			}
			continue
		}
		if !strings.Contains(url, "m="+tc.objectType) || !strings.Contains(url, "f=view") {
			t.Fatalf("%s url = %q, want m=%s&f=view", tc.objectType, url, tc.objectType)
		}
		if !strings.Contains(url, tc.wantID) {
			t.Fatalf("%s url = %q, want %s", tc.objectType, url, tc.wantID)
		}
		if strings.Contains(url, "projectID=0") || strings.Contains(url, "id=0&") || strings.HasSuffix(url, "id=0") {
			t.Fatalf("%s url must not carry a zero id/projectID, got %q", tc.objectType, url)
		}
	}
}

// 上下文查询失败必须向上返回 error。静默降级会把 DB 故障显示成
// "项目建设指引 28" 这种占位标题，用户无法区分"没有上下文"和"查库挂了"。
func TestFetchObjectContextsPropagatesApprovalQueryFailure(t *testing.T) {
	gormDB, mock := newPoolEnrichMockDB(t)
	repo := NewRepo(gormDB, gormDB)

	wantErr := errors.New("zt_projectbuildguide unavailable")
	mock.ExpectQuery(`FROM zt_projectbuildguide AS bg`).
		WithArgs(int64(28)).
		WillReturnError(wantErr)

	_, err := repo.fetchObjectContexts(t.Context(),
		[]doneActionDBRow{{ID: 900, ObjectType: "buildguideline", ObjectID: 28}})
	if err == nil {
		t.Fatal("expected buildguideline context failure to propagate, got nil error")
	}
	if !strings.Contains(err.Error(), "buildguideline") {
		t.Fatalf("error must name the failing approval object type, got %v", err)
	}
	if e := mock.ExpectationsWereMet(); e != nil {
		t.Fatalf("unmet sql expectations: %v", e)
	}
}

// 查询成功但对象行不存在（本地库 zt_projectbuildguide 只有 2 行，审批却引用到 id=28）
// 是合法的"无上下文"状态：不报错、留空标题由调用方回退，URL 仍用对象自身 ID。
func TestFetchObjectContextsMissingApprovalRowIsNotAnError(t *testing.T) {
	gormDB, mock := newPoolEnrichMockDB(t)
	repo := NewRepo(gormDB, gormDB)

	mock.ExpectQuery(`FROM zt_projectbuildguide AS bg`).
		WithArgs(int64(28)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "projectID", "project_name"}))

	ctxs, err := repo.fetchObjectContexts(t.Context(),
		[]doneActionDBRow{{ID: 901, ObjectType: "buildguideline", ObjectID: 28}})
	if err != nil {
		t.Fatalf("missing row must not be reported as an error, got %v", err)
	}
	if _, ok := ctxs["buildguideline:28"]; ok {
		t.Fatal("missing approval row must not be materialized as an empty context entry")
	}
	if e := mock.ExpectationsWereMet(); e != nil {
		t.Fatalf("unmet sql expectations: %v", e)
	}
}
