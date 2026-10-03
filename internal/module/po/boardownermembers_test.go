// =============================================================================
// 文件: internal/module/po/boardownermembers_test.go
// 模块: 工作看板
// 类型: test
// 职责: 有权限的小组成员即使无任务仍显示，保留任务数和单次查询。
// =============================================================================

package po

import (
	"github.com/DATA-DOG/go-sqlmock"
	"reflect"
	"testing"
)

func TestBoardOwnersIncludeZeroTaskMembersWithoutExtraQueries(t *testing.T) {
	db, mock := setupMockDB(t)
	mock.ExpectQuery(`(?s)SELECT .*COUNT\(\*\) AS count.*zt_task.finishedBy IN`).
		WillReturnRows(sqlmock.NewRows([]string{"account", "count"}).AddRow("busy", 3))
	req := BoardTaskReq{TeamgroupID: 1, members: []string{"idle", "busy", "idle"}, OwnerAccount: "busy", Focus: "overdue"}
	owners, err := NewRepo(db, db).FindBoardTaskOwners(t.Context(), req, map[string]string{"busy": "有任务", "idle": "空闲成员"})
	want := []BoardOwnerOption{{Account: "busy", Display: "有任务", Count: 3}, {Account: "idle", Display: "空闲成员", Count: 0}}
	if err != nil || !reflect.DeepEqual(owners, want) {
		t.Fatalf("owners=%+v, err=%v", owners, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestBoardStoryOwnersDoNotAddUnrelatedMembers(t *testing.T) {
	db, mock := setupMockDB(t)
	mock.ExpectQuery(`(?s)SELECT .*COUNT\(\*\) AS count.*zt_task.story`).
		WillReturnRows(sqlmock.NewRows([]string{"account", "count"}).AddRow("owner", 1))
	owners, err := NewRepo(db, db).FindBoardTaskOwners(t.Context(), BoardTaskReq{StoryID: 7}, nil)
	if err != nil || len(owners) != 1 || owners[0].Account != "owner" {
		t.Fatalf("owners=%+v, err=%v", owners, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
