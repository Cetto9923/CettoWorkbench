// =============================================================================
// 文件: internal/module/schedule/service_execution.go
// 模块: 排期工作台
// 类型: action
// 职责: 快速创建迭代业务编排（查团队成员并代理禅道创建执行）。
// 依赖: internal/model
//       internal/module/schedule/gateway_execution.go
//       internal/pkg/zentao
// =============================================================================

package schedule

import (
	"context"
	"errors"

	"workbench/internal/model"
	"workbench/internal/pkg/zentao"
)

// CreateExecution 以当前用户身份创建项目下的执行（迭代）。
func (s *Service) CreateExecution(ctx context.Context, actor *model.User, req CreateExecutionReq) (CreateExecutionResp, error) {
	var empty CreateExecutionResp
	if actorAccount(actor) == "" {
		return empty, errors.New("未登录或无法识别当前用户")
	}
	client := s.ztAPI
	if client == nil {
		client = zentao.API()
	}
	if client == nil {
		return empty, errors.New("禅道 API 未配置")
	}

	var teamMembers []string
	if req.ScrumTeam > 0 {
		accounts, err := s.repo.ListTeamgroupMemberAccounts(ctx, req.ScrumTeam)
		if err != nil {
			return empty, err
		}
		teamMembers = accounts
	}

	return createExecution(ctx, client, req, teamMembers)
}
