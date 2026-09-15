package schedule

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestListInsideUsersForScheduling_PrioritizesActorDepartment(t *testing.T) {
	db, mock := setupMockDB(t)
	repo := NewRepo(db)

	mock.ExpectQuery(`(?s)SELECT d\.path.*FROM zt_user.*LEFT JOIN zt_dept.*WHERE u\.account = \?`).
		WithArgs("alice").
		WillReturnRows(sqlmock.NewRows([]string{"path"}).AddRow(",1,5,18,"))
	mock.ExpectQuery(`(?s)SELECT u\.account, u\.realname, u\.pinyin, COALESCE\(d\.name, ''\) AS dept.*ORDER BY CASE WHEN d\.path LIKE \? THEN 0 ELSE 1 END, u\.account DESC`).
		WithArgs(",1,5,%").
		WillReturnRows(sqlmock.NewRows([]string{"account", "realname", "pinyin", "dept"}).
			AddRow("009900", "本部门用户", "", "金融科技总部").
			AddRow("009800", "本部门用户2", "", "金融科技总部").
			AddRow("010000", "其他部门用户", "", "其他部门"))

	users, err := repo.ListInsideUsersForScheduling(context.Background(), "alice")
	if err != nil {
		t.Fatalf("ListInsideUsersForScheduling returned error: %v", err)
	}
	if len(users) != 3 {
		t.Fatalf("got %d users, want 3", len(users))
	}
	if users[0].Account != "009900" || users[1].Account != "009800" || users[2].Account != "010000" {
		t.Fatalf("unexpected order: %#v", users)
	}
	if users[0].Dept != "金融科技总部" {
		t.Fatalf("department = %q, want 金融科技总部", users[0].Dept)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet SQL expectations: %v", err)
	}
}
