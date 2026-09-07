// =============================================================================
// 文件: internal/module/po/sidebar_badges.go
// 模块: PO 工作台
// 类型: action
// 职责: 侧栏菜单角标（我的待办 / 我的已办 / 通知中心 实时计数）。
//       一次请求一个 user 三个 count；总耗时 3 次 count 查询，单连接内串行。
//       不在循环里查 DB（无 N+1），不构造新框架，落在已有 Repo 风格上。
// 依赖: 现有 CountOpenTodos / CountRecentDone / CountUnreadNotices
// =============================================================================

package po

import (
	"context"
	"strings"
)

// SidebarBadges 侧栏角标数据。
type SidebarBadges struct {
	Todos  int `json:"todos"`
	Done   int `json:"done"`
	Notice int `json:"notice"`
}

// SidebarBadges 取当前 actor 三个角标计数。
// 失败时任何子项返 0，绝不阻塞页面渲染。
func (s *Service) SidebarBadges(ctx context.Context, actor *SidebarActor) (SidebarBadges, error) {
	out := SidebarBadges{}
	if s == nil || s.repo == nil || actor == nil {
		return out, nil
	}
	account := strings.TrimSpace(actor.Account)
	if account == "" {
		return out, nil
	}

	if n, err := s.repo.CountOpenTodos(ctx, account); err == nil {
		out.Todos = int(n)
	}

	if n, err := s.repo.CountRecentDone(ctx, account); err == nil {
		out.Done = int(n)
	}

	if n, err := s.repo.CountUnreadNotices(ctx, account); err == nil {
		out.Notice = int(n)
	}

	return out, nil
}

// SidebarActor 仅取字段，避免在 po 包内反向 import user 包。
// 调用方（render 包 bootstrap）传入的闭包负责把 *model.User 适配成此结构。
type SidebarActor struct {
	Account string
	ID      int64
}
