// =============================================================================
// 文件: internal/module/po/repo_badges_stub.go
// 模块: PO 工作台
// 职责: 待办与通知角标临时支持（第 3 批移植完整 Todos/Notice 全链路后由对应 repo 提供）。
// =============================================================================

package po

import "context"

// CountOpenTodos 返回待办未办总数（第 3 批接入完整统一待办查询）。
func (r *Repo) CountOpenTodos(ctx context.Context, account string) (int64, error) {
	return 0, nil
}

// CountUnreadNotices 返回未读通知总数（第 3 批接入通知中心查询）。
func (r *Repo) CountUnreadNotices(ctx context.Context, account string) (int64, error) {
	return 0, nil
}
