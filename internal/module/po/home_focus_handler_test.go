// =============================================================================
// 文件: internal/module/po/home_focus_handler_test.go
// 模块: PO 工作台
// 类型: action
// 职责: currentHandlerDemandWhere* 阶段办理人谓词的单元测试（验收 A-2 提测口径）。
// 依赖: 标准库 strings / testing
// =============================================================================

package po

import (
	"strings"
	"testing"
)

// 验收 A-2 口径，逐条对应报告场景：
//
//	BRA = 本人                              → 命中（不因列表「当前负责人」是别人而被误列）
//	BRA 空 + assignedTo = 本人 + 已到提测日   → 命中（US63451 之前永远进不了「待我处理」）
//	BRA 空 + assignedTo = 本人 + 未设/未来    → 不命中
//
// 整段 SQL 文本断言：assignedTo 兜底必须整体嵌在提测日条件内，BRA 分支不受日期限制；
// 「未设/未来不命中」由该嵌套结构保证，故随文本一并锁定。
func TestDevelopingHandlerSQLFollowsReportRule(t *testing.T) {
	const want = "(status = 'developing' AND (BRA = ? OR ((BRA IS NULL OR BRA = '') AND assignedTo = ? AND " +
		"developFinish IS NOT NULL AND CAST(developFinish AS CHAR) NOT LIKE '0000-00-00%' AND developFinish <= CURDATE())))"
	if developingHandlerSQL != want {
		t.Fatalf("developing handler clause mismatch:\n got %s\nwant %s", developingHandlerSQL, want)
	}
	// 提测日沿用仓库既有零日期安全 helper，不新造日期方言。
	if !strings.Contains(developingHandlerSQL, dateSetBeforeTodaySQL("developFinish")) {
		t.Fatalf("developing clause must reuse dateSetBeforeTodaySQL: %s", developingHandlerSQL)
	}
	if strings.Contains(developingHandlerSQL, "0000-00-00'") || strings.Contains(developingHandlerSQL, "!= '0000-00-00'") {
		t.Fatalf("developing clause must not compare zero-date literals: %s", developingHandlerSQL)
	}
}

// 三个谓词入口必须带同一条提测子句，且占位符数量与参数一一对应（错位会把账号串到别的阶段）。
func TestDevelopingHandlerSQLWiredIntoEveryMyActionWhere(t *testing.T) {
	atomicSQL, atomicArgs := currentHandlerDemandWhere("alice")
	emptySQL, emptyArgs := currentHandlerDemandWhereWithReviews("alice", []int{})
	fullSQL, fullArgs := currentHandlerDemandWhereWithReviews("alice", []int{101, 102})

	for _, c := range []struct {
		name     string
		clause   string
		args     []interface{}
		skipIdx  int
		wantArgs int
	}{
		{name: "currentHandlerDemandWhere", clause: atomicSQL, args: atomicArgs, skipIdx: -1, wantArgs: 16},
		{name: "withReviews(empty)", clause: emptySQL, args: emptyArgs, skipIdx: -1, wantArgs: 15},
		{name: "withReviews(ids)", clause: fullSQL, args: fullArgs, skipIdx: 1, wantArgs: 16},
	} {
		if !strings.Contains(c.clause, developingHandlerSQL) {
			t.Fatalf("%s missing developing handler clause: %s", c.name, c.clause)
		}
		if len(c.args) != c.wantArgs {
			t.Fatalf("%s args = %d, want %d", c.name, len(c.args), c.wantArgs)
		}
		verifyHandlerWhereArgs(t, c.name, c.args, c.skipIdx)
	}
}

// verifyHandlerWhereArgs 校验除 skipIdx 外每个占位符都绑定同一个账号——数量对但顺序错位，
// 会把账号悄悄串到别的阶段。
func verifyHandlerWhereArgs(t *testing.T, name string, args []interface{}, skipIdx int) {
	t.Helper()
	for i, a := range args {
		if i == skipIdx {
			if _, ok := a.([]int); !ok {
				t.Fatalf("%s arg[%d] = %T, want []int reviewDemandIDs", name, i, a)
			}
			continue
		}
		if a != "alice" {
			t.Fatalf("%s arg[%d] = %v, want alice", name, i, a)
		}
	}
}
