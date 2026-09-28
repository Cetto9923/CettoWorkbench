// =============================================================================
// 文件: internal/module/po/sidebar_badges.go
// 模块: PO 工作台
// 类型: action
// 职责: 侧栏菜单角标（我的待办 / 我的已办 / 通知中心）。
//       一次请求内串行三次 count（待办 / 已办 / 未读通知）。
// 依赖: CountOpenTodos / CountRecentDone / CountUnreadNotices
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

// SidebarActor 侧栏角标所需的最小调用方身份。
type SidebarActor struct {
	Account string
	ID      int64
}
