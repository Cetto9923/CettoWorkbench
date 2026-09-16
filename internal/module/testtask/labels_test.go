// =============================================================================
// 文件: internal/module/testtask/labels_test.go
// 模块: 提测办理
// 类型: action
// 职责: BuildContextResp 多参装配与 BuildSystemItems 主系统置顶的单测。
// 依赖: 无
// =============================================================================

package testtask

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestBuildContextRespIncludesQDAndUsers(t *testing.T) {
	row := DemandContextRow{
		ID:             42,
		Name:           "示例需求",
		Status:         "developing",
		QD:             "tester01",
		MainSystemName: "核心系统",
		EstimateLaunch: "2026-09-20",
	}
	displayMap := map[string]string{
		"tester01": "测试员(tester01)",
	}
	users := []UserOption{
		{Account: "tester01", Realname: "测试员"},
		{Account: "dev01", Realname: "开发员"},
	}

	got := BuildContextResp(row, displayMap, "admin", "管理员", nil, users)
	if got == nil {
		t.Fatal("BuildContextResp returned nil")
	}
	if got.QD != "tester01" {
		t.Fatalf("QD: got %q, want tester01", got.QD)
	}
	if got.QDName != "测试员(tester01)" {
		t.Fatalf("QDName: got %q, want 测试员(tester01)", got.QDName)
	}
	if len(got.Users) != 2 {
		t.Fatalf("Users len: got %d, want 2", len(got.Users))
	}
	if got.Users[0].Account != "tester01" || got.Users[0].Realname != "测试员" {
		t.Fatalf("Users[0]: %+v", got.Users[0])
	}
	if got.Users == nil {
		t.Fatal("Users must be non-nil empty slice when provided")
	}
}

func TestBuildContextRespSystemsAlwaysNonNil(t *testing.T) {
	row := DemandContextRow{ID: 1, Name: "n", Status: "developing"}
	got := BuildContextResp(row, nil, "", "", nil, nil)
	if got == nil {
		t.Fatal("BuildContextResp returned nil")
	}
	if got.Systems == nil {
		t.Fatal("Systems must be empty slice, not nil")
	}
	if got.Users == nil {
		t.Fatal("Users must be empty slice, not nil")
	}
	if got.HandlerName != "—" {
		t.Fatalf("HandlerName: got %q, want —", got.HandlerName)
	}
	if got.QDName != "—" {
		t.Fatalf("QDName: got %q, want —", got.QDName)
	}
}

func TestBuildSystemItemsMainPinned(t *testing.T) {
	products := []productRow{
		{ID: 7, Name: "配系统B"},
		{ID: 9, Name: "配系统C"},
	}
	got := BuildSystemItems(products, 9, "主系统C")
	if len(got) != 2 {
		t.Fatalf("len: got %d, want 2", len(got))
	}
	if got[0].ID != 9 || !got[0].IsMain {
		t.Fatalf("first item must be main: %+v", got[0])
	}
	if got[1].ID != 7 || got[1].IsMain {
		t.Fatalf("second item must be non-main: %+v", got[1])
	}
}

func TestBuildSystemItemsAppendsMissingMain(t *testing.T) {
	products := []productRow{{ID: 7, Name: "配系统B"}}
	got := BuildSystemItems(products, 9, "主系统C")
	if len(got) != 2 {
		t.Fatalf("len: got %d, want 2", len(got))
	}
	if got[0].ID != 9 || !got[0].IsMain || got[0].Name != "主系统C" {
		t.Fatalf("first item: %+v", got[0])
	}
	if got[1].ID != 7 {
		t.Fatalf("second item: %+v", got[1])
	}
}

func TestBuildSystemItemsEmpty(t *testing.T) {
	got := BuildSystemItems(nil, 0, "")
	if len(got) != 0 {
		t.Fatalf("len: got %d, want 0", len(got))
	}
}

func TestBuildExecutionOptionsStagefilterLeafNoClosed(t *testing.T) {
	rows := []executionRow{
		// 主项目 P1 / 主执行 E1（scrum；leaf 保留，stagefilter 不适用）
		{ID: 11, Name: "S1", ProjectID: 1, Parent: 0, Grade: 1, Attribute: "request", Status: "wait", Type: "sprint", ProjectName: "P1", ProjectModel: "scrum"},
		// F1 是 Design/Dev 的父（被 leaf 过滤：parentSet 含 12）
		{ID: 12, Name: "F1", ProjectID: 1, Parent: 0, Grade: 1, Attribute: "request", Status: "wait", Type: "stage", ProjectName: "P1", ProjectModel: "waterfall"},
		// Design 属性 request（waterfall → stagefilter 过滤）
		{ID: 13, Name: "Design", ProjectID: 1, Parent: 12, Grade: 2, Attribute: "request", Status: "wait", Type: "stage", ProjectName: "P1", ProjectModel: "waterfall"},
		// Dev 属性 dev（waterfall 保留；且为叶子）
		{ID: 14, Name: "Dev", ProjectID: 1, Parent: 12, Grade: 2, Attribute: "dev", Status: "wait", Type: "stage", ProjectName: "P1", ProjectModel: "waterfall"},
		// Done 已关闭（noclosed 过滤）
		{ID: 15, Name: "Done", ProjectID: 1, Parent: 0, Grade: 1, Attribute: "request", Status: "done", Type: "stage", ProjectName: "P1", ProjectModel: "waterfallplus"},
	}
	got := BuildExecutionOptions(rows, true)
	// 过滤后只剩 S1（scrum leaf）与 Dev（waterfall dev leaf）；F1 是 Design/Dev 的父被 leaf 剔除。
	if len(got) != 2 {
		t.Fatalf("len: got %d, want 2 (%+v)", len(got), got)
	}
	if got[0].Value != "1-11" || got[0].Label != "P1/S1" {
		t.Fatalf("[0]: %+v", got[0])
	}
	if got[1].Value != "1-14" || got[1].Label != "P1/Dev" {
		t.Fatalf("[1]: %+v", got[1])
	}
}

func TestBuildExecutionOptionsEmpty(t *testing.T) {
	got := BuildExecutionOptions(nil, true)
	if len(got) != 0 {
		t.Fatalf("len: got %d, want 0", len(got))
	}
	if got == nil {
		t.Fatal("must be empty slice, not nil")
	}
}

func TestValidateCreateBuildsReq(t *testing.T) {
	good := CreateBuildsReq{Builds: []CreateBuildItem{{
		ProductID: 1, ProjectID: 2, ExecutionID: 3, Name: "v1", Date: "2026-09-20",
	}}}
	if errs := good.Validate(); len(errs) != 0 {
		t.Fatalf("good req: %+v", errs)
	}
	bad := CreateBuildsReq{Builds: []CreateBuildItem{{ProductID: 1}}}
	errs := bad.Validate()
	if len(errs) == 0 {
		t.Fatal("bad req should produce errors")
	}
	empty := CreateBuildsReq{}
	if errs := empty.Validate(); len(errs) == 0 {
		t.Fatal("empty req should produce error")
	}
}

func TestUserOptionLabelDeduplication(t *testing.T) {
	cases := []struct {
		account  string
		realname string
		want     string
	}{
		{"000014", "丁喜莱", "丁喜莱(000014)"},
		{"000014", "丁喜莱(000014)", "丁喜莱(000014)"},
		{"000014", "", "000014"},
	}
	for _, tc := range cases {
		db, mock := setupMockTesttaskDB(t)
		mock.ExpectQuery(`(?s)SELECT account, realname, pinyin\s+FROM zt_user\s+WHERE deleted = '0'\s+AND type = 'inside'`).
			WillReturnRows(sqlmock.NewRows([]string{"account", "realname", "pinyin"}).AddRow(tc.account, tc.realname, "dxl"))
		options, err := NewRepo(db).ListInsideUsers(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		if len(options) != 1 || options[0].Account != tc.account || options[0].Label != tc.want || options[0].Pinyin != "dxl" {
			t.Fatalf("ListInsideUsers(%q, %q) = %+v, want label %q", tc.account, tc.realname, options, tc.want)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatal(err)
		}
	}
}
