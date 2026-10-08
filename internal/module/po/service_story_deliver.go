// =============================================================================
// 文件: internal/module/po/service_story_deliver.go
// 模块: PO 工作台
// 类型: service
// 职责: 独立研发需求发起交付的加载、权限与提交。
// =============================================================================

package po

import (
	"context"
	"fmt"
	"strings"

	"workbench/internal/model"
)

func (s *Service) loadDeliverStory(ctx context.Context, actor *model.User, storyID uint) (*deliverStoryRow, error) {
	if storyID == 0 {
		return nil, errStoryNotFound
	}
	account := ""
	if actor != nil {
		account = strings.TrimSpace(actor.Account)
	}
	if account == "" {
		return nil, errHomeActionForbidden
	}
	row, err := s.repo.FindDeliverStory(ctx, storyID)
	if err != nil {
		return nil, err
	}
	if actor.IsSuperAdmin || row.AssignedTo == account || row.ProductReqM == account {
		return row, nil
	}
	return nil, errHomeActionForbidden
}

// GetStoryDeliverMeta 获取独立研发需求发起交付表单。展示号是纯编号，不带 US。
func (s *Service) GetStoryDeliverMeta(ctx context.Context, actor *model.User, storyID uint) (*DemandDeliverMetaResp, error) {
	row, err := s.loadDeliverStory(ctx, actor, storyID)
	if err != nil {
		return nil, err
	}
	severeBugs, openBugs, err := s.repo.CheckStoryDeliverBlockers(ctx, storyID)
	if err != nil {
		return nil, fmt.Errorf("检查交付阻塞缺陷失败: %w", err)
	}
	windowID, windowName, releaseDate, err := s.repo.FindStoryLinkedWindow(ctx, storyID)
	if err != nil {
		return nil, err
	}
	windows, err := s.repo.ListUpcomingDeliverWindows(ctx)
	if err != nil {
		return nil, err
	}
	users, err := s.repo.ListInsideUsersForDeliver(ctx)
	if err != nil {
		return nil, err
	}
	return storyDeliverMeta(storyID, row, severeBugs, openBugs, windowID, windowName, releaseDate, windows, users), nil
}

func storyDeliverMeta(storyID uint, row *deliverStoryRow, severeBugs, openBugs int, windowID uint, windowName, releaseDate string, windows []DeliverWindowOption, users []ClarifyOption) *DemandDeliverMetaResp {
	ready := row.WindowBound == 1 && row.Delivered == 0
	testOk := severeBugs == 0
	blockReason := ""
	switch {
	case row.Delivered == 1:
		blockReason = "研发需求已交付"
	case row.WindowBound == 0:
		blockReason = "尚未绑定上线窗口，暂不能发起交付"
	case !testOk:
		blockReason = fmt.Sprintf("存在 %d 个严重缺陷未关闭，测试阻塞暂不能发起交付", severeBugs)
	}
	deliverDate := row.DeliverDate
	if deliverDate == "" {
		deliverDate = releaseDate
	}
	verifier := row.Verifier
	if verifier == "" {
		verifier = row.AssignedTo
	}
	return &DemandDeliverMetaResp{
		Success: true, DemandID: storyID, DisplayID: fmt.Sprintf("%d", storyID), Title: row.Title,
		Status: row.Status, ZentaoStatus: row.Status, WindowID: windowID, WindowName: windowName,
		ReleaseDate: releaseDate, DeliverDate: deliverDate, IsCarReview: defaultString(row.IsCarReview, "0"),
		IsGrayVerifyPlan: "0", VerifyDate: defaultString(row.VerifyDate, "1"), VerifyPlan: row.VerifyPlan,
		Verifier: verifier, Windows: windows, Users: users,
		Precheck: DeliverPrecheck{
			Rows: []DeliverPrecheckRow{
				{Label: "上线窗口", Value: boolStr(row.WindowBound == 1, "已绑定", "未绑定"), OK: row.WindowBound == 1},
				{Label: "交付状态", Value: boolStr(row.Delivered == 0, "可发起", "已交付"), OK: row.Delivered == 0},
				{Label: "测试阻塞", Value: testBlockerValue(severeBugs, openBugs), OK: testOk},
			},
			CanSubmit: ready && testOk, BlockReason: blockReason,
		},
	}
}

// DeliverStory 提交独立研发需求发起交付。
func (s *Service) DeliverStory(ctx context.Context, actor *model.User, req DemandDeliverReq) error {
	row, err := s.loadDeliverStory(ctx, actor, req.ID)
	if err != nil {
		return err
	}
	if row.Delivered == 1 || row.WindowBound == 0 || (req.Mode != "" && req.Mode != "create") {
		return errHomeActionConflict
	}
	severeBugs, _, err := s.repo.CheckStoryDeliverBlockers(ctx, req.ID)
	if err != nil {
		return fmt.Errorf("检查交付阻塞缺陷失败: %w", err)
	}
	if severeBugs > 0 {
		return fmt.Errorf("%w：当前有 %d 个未关闭", errDeliverBlocked, severeBugs)
	}
	comment := strings.TrimSpace(req.Comment)
	if comment == "" {
		comment = "工作台发起交付"
	}
	if req.WindowID > 0 {
		if err := s.repo.RequireDeliverWindow(ctx, req.WindowID); err != nil {
			return err
		}
	}
	return deliverStoryViaZentao(ctx, s.ztAPI, deliverDemandViaZentaoReq{
		DemandID: int64(req.ID), DeliverDate: req.DeliverDate, IsCarReview: req.IsCarReview,
		IsGrayVerifyPlan: req.IsGrayVerifyPlan, VerifyDate: req.VerifyDate, VerifyPlan: req.VerifyPlan,
		VeriFier: req.Verifier, Comment: comment,
	})
}
