// =============================================================================
// 文件: internal/module/po/service_clarify_guard_test.go
// 模块: PO 工作台
// 类型: test
// 职责: 验证需求澄清的状态门与禅道 isClickable 条件一致。
// =============================================================================

package po

import (
	"context"
	"testing"

	"workbench/internal/model"
	"workbench/internal/pkg/errorx"

	"github.com/DATA-DOG/go-sqlmock"
)

// expectDemandForClarify 造一行 FindDemandForClarify 的查询结果。
func expectDemandForClarify(t *testing.T, mock sqlmock.Sqlmock, id uint, status, hang string, isParent int) {
	t.Helper()
	mock.ExpectQuery(`(?s)SELECT.*FROM zt_demand.*WHERE id = \?`).
		WithArgs(id, 1).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "name", "status", "category", "BRA", "QD", "RD", "desc", "clarifyDesc",
			"mainSystem", "scaleEstimation", "isNewProduct", "isRelatedAccounts", "isNewFunction",
			"isOtherImportantOrder", "multiLegalPersonLogo", "deleted", "hang", "isParent",
		}).AddRow(
			id, "测试业需", status, "feature", "alice", "qd", "rd", "描述", "",
			"", "3", "0", "0", "0", "0", "0", "0", hang, isParent,
		))
}

// isClarifyGateErr 判断是否被澄清状态门拦下。
func isClarifyGateErr(err error) bool {
	bizErr, ok := errorx.IsBizError(err)
	return ok && bizErr.Code == errorx.ErrCodeConflict && bizErr.Msg == "当前需求状态不可澄清"
}

func TestClarifyDemand_StatusGateMatchesZentao(t *testing.T) {
	cases := []struct {
		name     string
		status   string
		hang     string
		isParent int
		want     bool
	}{
		{name: "active", status: "active", want: true},
		{name: "clarified", status: "clarified", want: true},
		{name: "developing", status: "developing", want: true},
		{name: "testing", status: "testing", want: true},
		{name: "waitacceptance", status: "waitacceptance", want: true},
		{name: "acceptanced", status: "acceptanced", want: true},
		{name: "draft rejected", status: "draft", want: false},
		{name: "wait rejected", status: "wait", want: false},
		{name: "refuse rejected", status: "refuse", want: false},
		{name: "waitdeliver rejected", status: "waitdeliver", want: false},
		{name: "released rejected", status: "released", want: false},
		{name: "closed rejected", status: "closed", want: false},
		{name: "suspended rejected", status: "active", hang: "1", want: false},
		{name: "parent rejected", status: "active", isParent: 1, want: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s, mock := newServiceForDeliverTest(t)
			expectDemandForClarify(t, mock, 1002, tc.status, tc.hang, tc.isParent)
			err := s.ClarifyDemand(context.Background(), &model.User{Account: "003030"}, DemandClarifySubmitReq{ID: 1002})
			if tc.want {
				// 状态通过后会继续走表单校验或禅道，这里只关心没有返回状态门错误。
				if isClarifyGateErr(err) {
					t.Fatalf("状态 %s 不该被状态门拒绝", tc.status)
				}
				return
			}
			if !isClarifyGateErr(err) {
				t.Fatalf("状态 %s 期望被状态门拒绝, got %v", tc.status, err)
			}
		})
	}
}

func TestGetDemandClarifyForm_StatusGate(t *testing.T) {
	s, mock := newServiceForDeliverTest(t)
	expectDemandForClarify(t, mock, 1002, "released", "0", 0)
	if _, err := s.GetDemandClarifyForm(context.Background(), &model.User{Account: "003030"}, 1002); !isClarifyGateErr(err) {
		t.Fatalf("期望已发布需求被状态门拒绝, got %v", err)
	}
}
