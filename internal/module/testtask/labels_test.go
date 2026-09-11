// =============================================================================
// 文件: internal/module/testtask/labels_test.go
// 模块: 提测办理
// 类型: action
// 职责: BuildContextResp 装配 qd / users 等字段的单测。
// 依赖: 无
// =============================================================================

package testtask

import "testing"

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
