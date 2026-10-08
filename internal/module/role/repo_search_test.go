// =============================================================================
// 文件: internal/module/role/repo_search_test.go
// 模块: 角色管理
// 类型: test
// 职责: 验证角色别名搜索的分页与总数使用相同条件。
// =============================================================================
package role

import (
	"context"
	"database/sql/driver"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func newRoleTestDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	t.Helper()
	conn, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	db, err := gorm.Open(mysql.New(mysql.Config{Conn: conn, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	return db, mock
}

func TestRoleSearchAliasesAndPagination(t *testing.T) {
	for _, keyword := range []string{"产品经理", "产品负责人", "PO", "经理", "负责人", "po", "产品", "管理员", ""} {
		t.Run(keyword, func(t *testing.T) {
			db, mock := newRoleTestDB(t)
			where := "WHERE deleted = 0"
			args := []driver.Value{}
			switch keyword {
			case "":
			case "管理员":
				where += " AND name LIKE ?"
				args = append(args, "%"+keyword+"%")
			default:
				where += " AND ((name LIKE ? OR code = ? OR name IN (?,?,?,?)))"
				args = append(args, "%"+keyword+"%", "po", "产品经理", "产品负责人", "PO", "产品负责人 (PO)")
			}
			where += " AND `zt_roles`.`deletedAt` IS NULL"
			mock.ExpectQuery(regexp.QuoteMeta("SELECT count(*) FROM `zt_roles` " + where)).WithArgs(args...).
				WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))
			itemArgs := append(append([]driver.Value{}, args...), 1, 1)
			mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM `zt_roles` " + where + " ORDER BY sortOrder ASC, id ASC LIMIT ? OFFSET ?")).
				WithArgs(itemArgs...).WillReturnRows(sqlmock.NewRows([]string{"id", "code", "name"}).AddRow(2, "po", "产品负责人"))
			rows, total, err := NewRepo(db).FindAll(context.Background(), RepoFindAllReq{Keyword: keyword, Page: 2, PageSize: 1})
			if err != nil || total != 2 || len(rows) != 1 {
				t.Fatalf("rows=%v total=%d err=%v", rows, total, err)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}
