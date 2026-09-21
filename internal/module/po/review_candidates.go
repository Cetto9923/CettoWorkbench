// =============================================================================
// 文件: internal/module/po/review_candidates.go
// 模块: PO 工作台
// 类型: action
// 职责: 提交业务评审弹框的评审人候选读取。
// =============================================================================

package po

import (
	"context"
	"strings"

	"workbench/internal/model"
	"workbench/internal/pkg/errorx"
)

// DemandReviewCandidatesResp 是提交评审弹框所需的候选人与当前评审人。
// Users 复用澄清表单的禅道用户格式，避免前端维护另一套人员字段。
type DemandReviewCandidatesResp struct {
	DemandID int64           `json:"demandId"`
	Selected []string        `json:"selected"`
	Users    []ClarifyOption `json:"users"`
}

// GetDemandReviewCandidates 查询当前需求可提交的业务评审人。
// 授权口径与 SubmitDemandReview 一致：草稿/驳回状态下，仅创建人、指派人或超级管理员可操作。
func (s *Service) GetDemandReviewCandidates(ctx context.Context, actor *model.User, demandID int64) (*DemandReviewCandidatesResp, error) {
	if actor == nil || strings.TrimSpace(actor.Account) == "" {
		return nil, errorx.New(errorx.ErrCodeForbidden, "请先登录")
	}
	if demandID <= 0 {
		return nil, errorx.New(errorx.ErrCodeInvalidParam, "需求 ID 无效")
	}
	demand, err := s.repo.FindDemandForReview(ctx, demandID)
	if err != nil {
		return nil, err
	}
	if demand == nil || demand.Deleted != "0" {
		return nil, errorx.New(errorx.ErrCodeNotFound, "需求不存在")
	}
	status := strings.TrimSpace(demand.Status)
	if status != "draft" && status != "refuse" {
		return nil, errorx.New(errorx.ErrCodeConflict, "当前状态不允许提交评审")
	}
	account := strings.TrimSpace(actor.Account)
	if !actor.IsSuperAdmin && strings.TrimSpace(demand.CreatedBy) != account && strings.TrimSpace(demand.AssignedTo) != account {
		return nil, errorx.New(errorx.ErrCodeForbidden, "只有创建人或指派人可以选择评审人")
	}
	users, err := s.repo.FindCandidateUsers(ctx, account)
	if err != nil {
		return nil, err
	}
	return &DemandReviewCandidatesResp{
		DemandID: demandID,
		Selected: splitReviewerAccounts(demand.Reviewer),
		Users:    users,
	}, nil
}
