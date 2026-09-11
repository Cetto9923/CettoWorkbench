// =============================================================================
// 文件: internal/module/po/service_demand_edit.go
// 模块: PO 工作台
// 类型: service
// 职责: 业务需求草稿/驳回状态的编辑初始化、内容修改与安全删除（严格对象级创建人鉴权）。
// =============================================================================

package po

import (
	"context"
	"fmt"
	"strings"
	"time"

	"workbench/internal/model"
	"workbench/internal/pkg/errorx"
)

// GetDemandEditData 获取业务需求编辑页初始化数据及选项字典。
func (s *Service) GetDemandEditData(ctx context.Context, actor *model.User, demandID int64) (*DemandEditDataResp, error) {
	if actor == nil || strings.TrimSpace(actor.Account) == "" {
		return nil, errorx.New(errorx.ErrCodeForbidden, "请先登录")
	}
	account := strings.TrimSpace(actor.Account)

	demand, err := s.repo.FindDemandForEdit(ctx, demandID)
	if err != nil {
		return nil, err
	}
	if demand == nil || demand.Deleted != "0" {
		return nil, errorx.New(errorx.ErrCodeNotFound, "需求不存在或已被删除")
	}

	st := strings.ToLower(strings.TrimSpace(demand.Status))
	if st != "draft" && st != "refuse" {
		return nil, errorx.New(errorx.ErrCodeConflict, "该需求已进入评审或已通过，不可编辑或删除")
	}

	isCreator := actor.IsSuperAdmin || strings.TrimSpace(demand.CreatedBy) == account
	if !isCreator {
		return nil, errorx.New(errorx.ErrCodeForbidden, "只有需求创建人可以编辑或删除该需求")
	}

	opts, err := s.repo.ListDemandEditOptions(ctx)
	if err != nil {
		opts = &DemandEditOptions{}
	}

	elStr := ""
	if demand.EstimateLaunch != nil {
		elStr = demand.EstimateLaunch.Format("2006-01-02")
	}

	statusLabel := "草稿 / 暂存"
	if st == "refuse" {
		statusLabel = "已驳回"
	}

	code := fmt.Sprintf("US%d", demand.ID)

	return &DemandEditDataResp{
		Success: true,
		Demand: DemandEditFields{
			ID:             demand.ID,
			Code:           code,
			Name:           demand.Name,
			Status:         demand.Status,
			StatusLabel:    statusLabel,
			Category:       demand.Category,
			Pri:            demand.Pri,
			Pool:           demand.Pool,
			PoolName:       demand.PoolName,
			Product:        demand.Product,
			ProductName:    demand.ProductName,
			Reviewer:       demand.Reviewer,
			ReviewerName:   demand.ReviewerName,
			EstimateLaunch: elStr,
			Source:         demand.Source,
			SourceNote:     demand.SourceNote,
			Desc:           demand.Desc,
			VerifyPlan:     demand.VerifyPlan,
			CreatedBy:      demand.CreatedBy,
			CreatedByName:  demand.CreatedByName,
			IsCreator:      isCreator,
		},
		Options:   *opts,
		CanEdit:   true,
		CanDelete: true,
	}, nil
}

// UpdateDemand 更新业务需求内容（草稿/驳回状态下，创建人可操作）。
func (s *Service) UpdateDemand(ctx context.Context, actor *model.User, req UpdateDemandReq) error {
	if actor == nil || strings.TrimSpace(actor.Account) == "" {
		return errorx.New(errorx.ErrCodeForbidden, "请先登录")
	}
	account := strings.TrimSpace(actor.Account)

	demand, err := s.repo.FindDemandForEdit(ctx, req.ID)
	if err != nil {
		return err
	}
	if demand == nil || demand.Deleted != "0" {
		return errorx.New(errorx.ErrCodeNotFound, "需求不存在或已被删除")
	}

	st := strings.ToLower(strings.TrimSpace(demand.Status))
	if st != "draft" && st != "refuse" {
		return errorx.New(errorx.ErrCodeConflict, "当前需求状态不允许编辑")
	}

	isCreator := actor.IsSuperAdmin || strings.TrimSpace(demand.CreatedBy) == account
	if !isCreator {
		return errorx.New(errorx.ErrCodeForbidden, "只有需求创建人可以编辑该需求")
	}

	updates := map[string]any{
		"name":       strings.TrimSpace(req.Name),
		"category":   strings.TrimSpace(req.Category),
		"source":     strings.TrimSpace(req.Source),
		"sourceNote": strings.TrimSpace(req.SourceNote),
		"desc":       req.Desc,
		"verifyPlan": req.VerifyPlan,
	}

	if req.Pri != "" {
		updates["pri"] = strings.TrimSpace(req.Pri)
	}
	if req.Pool > 0 {
		updates["pool"] = req.Pool
	}
	if req.Product != "" {
		updates["product"] = strings.TrimSpace(req.Product)
	}
	if req.Reviewer != "" {
		updates["reviewer"] = strings.TrimSpace(req.Reviewer)
	}

	el := strings.TrimSpace(req.EstimateLaunch)
	if el != "" && el != "—" {
		if t, err := time.Parse("2006-01-02", el); err == nil {
			updates["estimateLaunch"] = t
		}
	} else if el == "" {
		updates["estimateLaunch"] = nil
	}

	if err := s.repo.UpdateDemandDraft(ctx, req.ID, account, updates, req.Comment); err != nil {
		return err
	}

	// 若用户勾选「保存并提交评审」
	if req.SubmitReview {
		rev := strings.TrimSpace(req.Reviewer)
		if rev == "" {
			rev = demand.Reviewer
		}
		c := strings.TrimSpace(req.Comment)
		if c == "" {
			c = "编辑后直接提交业务评审"
		}
		return s.SubmitDemandReview(ctx, actor, SubmitDemandReviewReq{
			ID:       req.ID,
			Reviewer: rev,
			Comment:  c,
		})
	}

	return nil
}

// DeleteDemand 安全删除业务需求（草稿/驳回状态下，创建人可操作）。
func (s *Service) DeleteDemand(ctx context.Context, actor *model.User, req DeleteDemandReq) error {
	if actor == nil || strings.TrimSpace(actor.Account) == "" {
		return errorx.New(errorx.ErrCodeForbidden, "请先登录")
	}
	account := strings.TrimSpace(actor.Account)

	demand, err := s.repo.FindDemandForEdit(ctx, req.ID)
	if err != nil {
		return err
	}
	if demand == nil || demand.Deleted != "0" {
		return errorx.New(errorx.ErrCodeNotFound, "需求不存在或已被删除")
	}

	st := strings.ToLower(strings.TrimSpace(demand.Status))
	if st != "draft" && st != "refuse" {
		return errorx.New(errorx.ErrCodeConflict, "已进入评审或已通过的需求不能删除")
	}

	isCreator := actor.IsSuperAdmin || strings.TrimSpace(demand.CreatedBy) == account
	if !isCreator {
		return errorx.New(errorx.ErrCodeForbidden, "只有需求创建人可以删除该需求")
	}

	c := strings.TrimSpace(req.Comment)
	if c == "" {
		c = "工作台创建人删除需求"
	}

	return s.repo.DeleteDemandDraft(ctx, req.ID, account, c)
}
