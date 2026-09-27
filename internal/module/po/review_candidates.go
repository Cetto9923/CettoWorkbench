// =============================================================================
// 文件: internal/module/po/review_candidates.go
// 模块: PO 工作台
// 类型: action
// 职责: 提交业务评审弹框的评审人候选读取。
// 依赖: internal/model
//       internal/pkg/errorx
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
// 授权口径与 SubmitDemandReview 一致（同 Main）：暂存/驳回状态下，仅创建人可操作。
// 候选人口径同 Main（bug #39365）：只取业需所属需求池的 businessReviewer，对齐禅道 demand-submit。
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
		return nil, errorx.New(errorx.ErrCodeConflict, "仅暂存或已驳回的需求可发起评审")
	}
	account := strings.TrimSpace(actor.Account)
	if strings.TrimSpace(demand.CreatedBy) != account {
		return nil, errorx.New(errorx.ErrCodeForbidden, "只有创建人可以发起评审")
	}
	poolReviewers, err := s.repo.FindDemandPoolBusinessReviewer(ctx, demandID)
	if err != nil {
		return nil, err
	}
	allUsers, err := s.repo.FindCandidateUsers(ctx, account)
	if err != nil {
		return nil, err
	}
	users := filterPoolBusinessReviewers(allUsers, splitReviewerAccounts(poolReviewers))
	return &DemandReviewCandidatesResp{
		DemandID: demandID,
		Selected: splitReviewerAccounts(demand.Reviewer),
		Users:    users,
	}, nil
}

// filterPoolBusinessReviewers 按需求池业务评审人账号顺序筛出候选人；已删除账号不在 users 中，自然剔除。
func filterPoolBusinessReviewers(users []ClarifyOption, poolAccounts []string) []ClarifyOption {
	byAccount := make(map[string]ClarifyOption, len(users))
	for _, u := range users {
		byAccount[u.Value] = u
	}
	out := make([]ClarifyOption, 0, len(poolAccounts))
	for _, account := range poolAccounts {
		if u, ok := byAccount[account]; ok {
			out = append(out, u)
		}
	}
	return out
}
