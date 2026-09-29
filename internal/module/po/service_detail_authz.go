// =============================================================================
// 文件: internal/module/po/service_detail_authz.go
// 模块: PO 工作台
// 类型: service
// 职责: 业务需求统一详情的对象级授权（F01）与 API 出口富文本净化（F02）。
//       授权决策在此集中，Repo 仅做候选行投影。
// =============================================================================

package po

import (
	"context"
	"errors"
	"strings"

	"gorm.io/gorm"

	"workbench/internal/model"
	"workbench/internal/module/demandauthz"
	"workbench/internal/pkg/errorx"
	"workbench/internal/pkg/perm"
)

// GetDemandDetailAuthZ 输入校验 + 对象级授权 + not-found 区分的最小入口辅助。
//
// F01：actor 缺失/空视为未登录；demandID == 0 视为无效入参；存储错误
// 原样向上抛；ErrRecordNotFound 转换为 errorx.NotFound。授权失败统一
// 返回 errorx.Forbidden；超级管理员（IsSuperAdmin）跳过对象级检查。
func (s *DetailService) GetDemandDetailAuthZ(ctx context.Context, actor *model.User, demandID uint) (*DemandDetailRow, error) {
	if actor == nil || strings.TrimSpace(actor.Account) == "" {
		return nil, errorx.New(errorx.ErrCodeInvalidParam, "缺少用户身份")
	}
	if demandID == 0 {
		return nil, errorx.New(errorx.ErrCodeInvalidParam, "需求 ID 无效")
	}
	return s.loadDemandIfVisible(ctx, actor, demandID)
}

// loadDemandIfVisible 是 F01 的核心授权辅助：先取主记录（区分不存在与存在
// 不可见），再做对象级权限判断。授权失败统一返回 errorx.Forbidden；
// 不存在返回 errorx.NotFound；存储错误原样向上抛。
//
// 可见性矩阵（F01）= 对象级关系（读写共用同一口径，见 demandauthz.Evaluate）
// OR 需求查询权限（perm.ScheduleList）只读放行：
//
//	超级管理员（actor.IsSuperAdmin == true）：全部可见、可写。
//	组织角色 PMO（zt_roles.code = 'pmo'）：全部可见、可写。
//	个人干系人关系（命中至少一条）：
//	  - PO/BR：d.BRA == actor.Account
//	  - 研发负责人：d.RD == actor.Account
//	  - 业务提出人：d.originator == actor.Account
//	  - 当前责任人：d.assignedTo == actor.Account
//	  - 测试/验收人：d.QD == actor.Account || d.accepter == actor.Account
//	  - 评审人：d.reviewer == actor.Account
//	  - 创建人：d.createdBy == actor.Account
//	  - 闭环人：d.closedBy == actor.Account
//	  - 编辑人：d.editedBy == actor.Account
//	  - 反馈接收人：d.feedbackedBy == actor.Account
//	  - 澄清 PM：zt_demandclarify.PM == actor.Account
//	组织角色（团队长）：需求任一干系人属于该团队长管辖部门（zt_dept.manager，
//	补缺表优先）或其下级部门。
//	以上均未命中时，perm.ScheduleList 仅放行「只读查看」，不授予任何写权限。
//
// 父需求沿用同一矩阵；子需求列表各自独立走 loadDemandIfVisible，避免父可见
// 子不可见时的正文泄露（详见 service_detail.go 父/子聚合）。
func (s *DetailService) loadDemandIfVisible(ctx context.Context, actor *model.User, demandID uint) (*DemandDetailRow, error) {
	row, err := s.repo.FindDemandDetailByID(ctx, demandID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errorx.New(errorx.ErrCodeNotFound, "需求不存在")
		}
		return nil, err
	}

	// 对象级关系判断：超管 / PMO / 干系人 / 团队长管辖。
	// 与排期工作台写授权（demandauthz.Evaluate）共用同一份 SQL 与顺序。
	access, err := s.repo.authzRepo().Evaluate(ctx, actor, demandID)
	if err != nil {
		return nil, err
	}
	if access != demandauthz.AccessNone {
		return row, nil
	}

	// 需求查询权限仍可放行只读查看；该权限不含写权限（见 RequireDemandWrite）。
	if perm.HasAnyGranted(ctx, perm.ScheduleList) {
		return row, nil
	}

	return nil, errorx.New(errorx.ErrCodeForbidden, "无权查看该业务需求")
}
