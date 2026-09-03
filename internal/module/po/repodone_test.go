// =============================================================================
// 文件: internal/module/po/repodone_test.go
// 模块: PO 工作台
// 类型: action
// 职责: 验证我的已办正式动作注册口径。
// 依赖: 无
// =============================================================================

package po

import "testing"

func TestFormalDoneActionsExcludeNoise(t *testing.T) {
	for _, key := range []string{"user:login", "demand:edit", "task:comment"} {
		if _, ok := formalDoneActions[key]; ok {
			t.Fatalf("noise action %q must not enter formal done", key)
		}
	}
	for _, key := range []string{"demand:clarify", "task:finished", "bug:resolved"} {
		if _, ok := formalDoneActions[key]; !ok {
			t.Fatalf("formal action %q missing", key)
		}
	}
}
