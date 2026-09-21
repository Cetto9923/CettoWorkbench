// =============================================================================
// 文件: internal/module/schedule/service_reviewtostory.go
// 模块: 排期工作台
// 类型: action
// 职责: 检查业需转研发需求前是否需完成主管部门审批。
// 依赖: internal/model
//       internal/module/schedule/gateway.go
// =============================================================================

package schedule

import (
	"context"
	"fmt"
	"strings"

	"workbench/internal/model"
)

// CheckReviewToStoryNotice 调用禅道检查主管部门审批提醒。
// 返回非空文案表示需拦截，不允许继续添加研发需求。
func (s *Service) CheckReviewToStoryNotice(ctx context.Context, actor *model.User, demandID uint) (string, error) {
	if actor == nil || strings.TrimSpace(actor.Account) == "" {
		return "", fmt.Errorf("未登录或禅道账号为空")
	}
	return checkReviewToStoryNotice(ctx, s.ztAPI, demandID)
}
