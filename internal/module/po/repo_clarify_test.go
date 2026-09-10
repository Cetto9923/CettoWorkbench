package po

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestFindCandidateProductsPrioritizesActorParticipation(t *testing.T) {
	db, mock := setupMockDB(t)
	repo := NewRepo(db, nil)

	mock.ExpectQuery("SELECT id, name, PO,[\\s\\S]*AS participating FROM `zt_product`[\\s\\S]*ORDER BY participating DESC, `order` DESC, id DESC").
		WithArgs("alice", "alice", "alice", "alice", "alice").
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "PO", "participating"}).
			AddRow(2, "我参与的产品", "alice", true).
			AddRow(1, "其他产品", "bob", false))

	products, err := repo.FindCandidateProducts(context.Background(), "alice")
	if err != nil {
		t.Fatalf("FindCandidateProducts returned error: %v", err)
	}
	if len(products) != 2 || !products[0].Participating || products[1].Participating {
		t.Fatalf("participation ordering/flag = %#v, want participating product first", products)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet SQL expectations: %v", err)
	}
}

func TestFindFrequentProducts(t *testing.T) {
	db, mock := setupMockDB(t)
	repo := NewRepo(db, nil)

	mock.ExpectQuery("SELECT p.id, p.name, p.PO, COUNT\\(dc.id\\) AS clarify_count[\\s\\S]*LIMIT \\?").
		WithArgs("alice", "alice", "alice", 8).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "PO", "clarify_count"}).
			AddRow(35, "手机银行", "alice", 12).
			AddRow(18, "核心业务系统", "bob", 5))

	prods, err := repo.FindFrequentProducts(context.Background(), "alice", 8)
	if err != nil {
		t.Fatalf("FindFrequentProducts error: %v", err)
	}
	if len(prods) != 2 || prods[0].ID != 35 || prods[0].ClarifyCount != 12 {
		t.Fatalf("unexpected frequent products: %#v", prods)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet SQL expectations: %v", err)
	}
}

func TestFindProductMembers(t *testing.T) {
	db, mock := setupMockDB(t)
	repo := NewRepo(db, nil)

	mock.ExpectQuery("SELECT id, PO, QD, RD, feedback, ticket, createdBy, whitelist FROM `zt_product` WHERE id IN \\(\\?\\) AND deleted = '0'").
		WithArgs(35).
		WillReturnRows(sqlmock.NewRows([]string{"id", "PO", "QD", "RD", "feedback", "ticket", "createdBy", "whitelist"}).
			AddRow(35, "alice", "bob", "charlie", "", "", "david", "eve,frank"))

	mock.ExpectQuery("SELECT DISTINCT product, PM FROM `zt_demandclarify` WHERE product IN \\(\\?\\) AND PM != ''").
		WithArgs("35").
		WillReturnRows(sqlmock.NewRows([]string{"product", "PM"}).
			AddRow("35", "grace"))

	mock.ExpectQuery("SELECT account, realname FROM `zt_user` WHERE account IN[\\s\\S]*").
		WillReturnRows(sqlmock.NewRows([]string{"account", "realname"}).
			AddRow("alice", "爱丽丝").
			AddRow("bob", "鲍勃").
			AddRow("charlie", "查理").
			AddRow("eve", "伊芙").
			AddRow("frank", "弗兰克").
			AddRow("grace", "格蕾丝").
			AddRow("david", "大卫"))

	members, err := repo.FindProductMembers(context.Background(), []int64{35})
	if err != nil {
		t.Fatalf("FindProductMembers error: %v", err)
	}
	list := members["35"]
	if len(list) < 5 {
		t.Fatalf("expected at least 5 members, got: %#v", list)
	}
	if list[0].Account != "alice" || list[0].Role != "产品负责人 (PO)" {
		t.Fatalf("expected PO first, got: %#v", list[0])
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet SQL expectations: %v", err)
	}
}

func TestFindCandidateUsers_PrimaryDeptFirstAndAccountDesc(t *testing.T) {
	db, mock := setupMockDB(t)
	repo := NewRepo(db, nil)

	// 1. 查找操作人员工所在部门
	mock.ExpectQuery("SELECT `dept` FROM `zt_user` WHERE account = \\? AND deleted = '0' LIMIT \\?").
		WithArgs("002000", 1).
		WillReturnRows(sqlmock.NewRows([]string{"dept"}).AddRow(18))

	// 2. 查找操作人员工部门的 path (总行=1, 金融科技总部=5, 软件开发一处=18)
	mock.ExpectQuery("SELECT id, parent, path, grade FROM `zt_dept` WHERE id = \\? LIMIT \\?").
		WithArgs(18, 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "parent", "path", "grade"}).
			AddRow(18, 5, ",1,5,18,", 3))

	// 3. 查找一级部门 5 (金融科技总部) 的详情与 path
	mock.ExpectQuery("SELECT id, path FROM `zt_dept` WHERE id = \\? LIMIT \\?").
		WithArgs(5, 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "path"}).
			AddRow(5, ",1,5,"))

	// 4. 查找一级部门及其所有子部门 ID
	mock.ExpectQuery("SELECT `id` FROM `zt_dept` WHERE id = \\? OR path LIKE \\?").
		WithArgs(5, ",1,5,%").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).
			AddRow(5).AddRow(18).AddRow(19))

	// 5. 候选人排序查询：本一级部门置顶，工号倒序，并带回所在部门。
	mock.ExpectQuery("SELECT u\\.account, u\\.realname, COALESCE\\(d\\.name, ''\\) AS dept FROM zt_user AS u LEFT JOIN zt_dept AS d ON d\\.id = u\\.dept WHERE u\\.deleted = '0' ORDER BY CASE WHEN u\\.dept IN \\(\\?,\\?,\\?\\) THEN 0 ELSE 1 END, u\\.account DESC").
		WithArgs(5, 18, 19).
		WillReturnRows(sqlmock.NewRows([]string{"account", "realname", "dept"}).
			AddRow("003000", "张三", "金融科技总部").
			AddRow("002000", "李四", "软件开发一处").
			AddRow("001000", "王五", "软件开发二处").
			AddRow("009999", "赵六", "业务部门").
			AddRow("008888", "钱七", ""))

	users, err := repo.FindCandidateUsers(context.Background(), "002000")
	if err != nil {
		t.Fatalf("FindCandidateUsers error: %v", err)
	}

	if len(users) != 5 {
		t.Fatalf("expected 5 candidate users, got %d", len(users))
	}

	expectedOrder := []string{"003000", "002000", "001000", "009999", "008888"}
	for i, exp := range expectedOrder {
		if users[i].Value != exp {
			t.Errorf("user[%d].Value = %q, want %q", i, users[i].Value, exp)
		}
	}

	if users[0].Label != "张三(003000)" {
		t.Errorf("user[0].Label = %q, want %q", users[0].Label, "张三(003000)")
	}
	if users[0].Dept != "金融科技总部" || users[4].Dept != "" {
		t.Errorf("unexpected user departments: first=%q last=%q", users[0].Dept, users[4].Dept)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet SQL expectations: %v", err)
	}
}

func TestFindCandidateUsers_FallbackToAccountDescWhenNoDept(t *testing.T) {
	db, mock := setupMockDB(t)
	repo := NewRepo(db, nil)

	// 操作人员工无部门/空部门
	mock.ExpectQuery("SELECT `dept` FROM `zt_user` WHERE account = \\? AND deleted = '0' LIMIT \\?").
		WithArgs("009999", 1).
		WillReturnRows(sqlmock.NewRows([]string{"dept"}).AddRow(0))

	// 回退到纯工号倒序，仍带回部门字段。
	mock.ExpectQuery("SELECT u\\.account, u\\.realname, COALESCE\\(d\\.name, ''\\) AS dept FROM zt_user AS u LEFT JOIN zt_dept AS d ON d\\.id = u\\.dept WHERE u\\.deleted = '0' ORDER BY u\\.account DESC").
		WillReturnRows(sqlmock.NewRows([]string{"account", "realname", "dept"}).
			AddRow("009999", "赵六", "业务部门").
			AddRow("001000", "张三", ""))

	users, err := repo.FindCandidateUsers(context.Background(), "009999")
	if err != nil {
		t.Fatalf("FindCandidateUsers error: %v", err)
	}

	if len(users) != 2 || users[0].Value != "009999" || users[1].Value != "001000" {
		t.Fatalf("unexpected users ordering: %#v", users)
	}
	if users[0].Dept != "业务部门" || users[1].Dept != "" {
		t.Fatalf("unexpected user departments: %#v", users)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet SQL expectations: %v", err)
	}
}
