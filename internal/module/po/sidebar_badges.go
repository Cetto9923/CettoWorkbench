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
	"time"
	"workbench/internal/model"
	"workbench/internal/pkg/render"
)

// SidebarBadges 侧栏角标数据。
type SidebarBadges = render.SidebarBadges

// SidebarBadges 取当前 actor 三个角标计数。
// 角标最多等待 250ms；失败不冒充真实零值。
func (s *Service) SidebarBadges(ctx context.Context, actor *model.User) (SidebarBadges, error) {
	out := SidebarBadges{}
	if s == nil || s.repo == nil || actor == nil {
		return out, nil
	}
	account := strings.TrimSpace(actor.Account)
	if account == "" {
		return out, nil
	}

	ctx, cancel := context.WithTimeout(ctx, 250*time.Millisecond)
	defer cancel()
	for _, item := range []struct {
		count  func(context.Context, string) (int64, error)
		target *int
	}{
		{s.repo.CountOpenTodos, &out.Todos}, {s.repo.CountRecentDone, &out.Done}, {s.repo.CountUnreadNotices, &out.Notice},
	} {
		n, err := item.count(ctx, account)
		if err != nil {
			out.Unavailable = true
			return out, err
		}
		*item.target = int(n)
	}

	return out, nil
}
