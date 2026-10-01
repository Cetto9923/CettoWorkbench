// =============================================================================
// 文件: internal/module/po/repotodo.go
// 模块: PO 工作台
// 类型: action
// 职责: 待办共用辅助（CountOpenTodos、关系/截止日/优先级标签、账号展示名映射）。
//       列表查询见 QueryTodoUnified / todo_query_*.go。
// 依赖: 无
// =============================================================================

package po

import (
	"context"
	"strings"
)

// CountOpenTodos 返回当前账号待办总数（不取 items）。
func (r *Repo) CountOpenTodos(ctx context.Context, account string) (int64, error) {
	if r == nil || r.db == nil || strings.TrimSpace(account) == "" {
		return 0, nil
	}
	res, err := r.QueryTodoUnified(ctx, account, TodoListReq{Page: 1, PageSize: 0})
	if err != nil {
		return 0, err
	}
	return res.Total, nil
}

func todoActionLabel(status string) string {
	switch status {
	case "draft", "wait", "refuse":
		return "受理"
	case "active":
		return "澄清"
	case "clarified":
		return "排期"
	case "developing", "testing":
		return "跟进"
	case "waitacceptance":
		return "验收"
	case "acceptanced":
		return "发起交付"
	case "waitdeliver":
		return "跟进发布"
	default:
		return "查看"
	}
}
