// =============================================================================
// 文件: internal/module/po/service_deliver.go
// 模块: PO 工作台
// 类型: service
// 职责: 发起交付前置检查、表单数据初始化、参数校验与全流程流转编排。
// =============================================================================

package po

import (
	"context"
	"fmt"
	"strings"

	"workbench/internal/model"
)

// GetDemandDeliverMeta 获取发起交付表单初始化数据及前置检查结果。
func (s *Service) GetDemandDeliverMeta(ctx context.Context, actor *model.User, demandID uint) (*DemandDeliverMetaResp, error) {
	if demandID == 0 {
		return nil, errHomeActionNotFound
	}
	account := ""
	if actor != nil {
		account = strings.TrimSpace(actor.Account)
	}
	if account == "" {
		return nil, errHomeActionForbidden
	}

	row, err := s.repo.FindDeliverDemand(ctx, demandID)
	if err != nil {
		return nil, err
	}

	// 权限检查：必须命中 RoleOwner
	homeRow := &homeActionDemandRow{
		ID:            row.ID,
		Status:        row.Status,
		Deleted:       row.Deleted,
		AssignedTo:    row.AssignedTo,
		DistributedBy: row.DistributedBy,
		CreatedBy:     row.CreatedBy,
		SubmitedBy:    row.SubmitedBy,
		SubmitBy:      row.SubmitBy,
		QD:            row.QD,
		RD:            row.RD,
		BRA:           row.BRA,
		Accepter:      row.Accepter,
		VeriFier:      row.VeriFier,
		Product:       row.Product,
	}
	if !s.repo.homeActionAuthorized(homeRow, account, "deliver") {
		return nil, errHomeActionForbidden
	}

	// 前置检查
	severeBugs, openBugs, err := s.repo.CheckDeliverBlockers(ctx, demandID)
	if err != nil {
		return nil, fmt.Errorf("检查交付阻塞缺陷失败: %w", err)
	}
	acceptOk := row.Status == "acceptanced" || row.Status == "waitdeliver" || row.VerifyFinish != ""
	mgrOk := true // 主管部门审批（当前禅道库中无阻断 approval）
	testOk := severeBugs == 0

	precheckRows := []DeliverPrecheckRow{
		{Label: "业务验收", Value: boolStr(acceptOk, "通过", "未通过"), OK: acceptOk},
		{Label: "主管部门审批", Value: boolStr(mgrOk, "通过", "未完成"), OK: mgrOk},
		{Label: "测试阻塞", Value: testBlockerValue(severeBugs, openBugs), OK: testOk},
	}
	canSubmit := acceptOk && mgrOk && testOk
	blockReason := ""
	if !acceptOk {
		blockReason = "业务验收未通过，暂不能发起交付"
	} else if !mgrOk {
		blockReason = "主管部门审批未完成，暂不能发起交付"
	} else if !testOk {
		blockReason = fmt.Sprintf("存在 %d 个严重缺陷未关闭，测试阻塞暂不能发起交付", severeBugs)
	}

	// 查询已关联上线窗口与候选窗口
	windowID, windowName, releaseDate, _ := s.repo.FindDemandLinkedWindow(ctx, demandID)
	windows, _ := s.repo.ListUpcomingDeliverWindows(ctx)
	users, _ := s.repo.ListInsideUsersForDeliver(ctx)

	deliverDate := row.DeliverDate
	if deliverDate == "" && releaseDate != "" {
		deliverDate = releaseDate
	}
	verifier := row.VeriFier
	if verifier == "" {
		verifier = row.Accepter
	}

	resp := &DemandDeliverMetaResp{
		Success:          true,
		DemandID:         demandID,
		DisplayID:        fmt.Sprintf("US%d", demandID),
		Title:            row.Name,
		Status:           row.Status,
		ZentaoStatus:     row.Status,
		WindowID:         windowID,
		WindowName:       windowName,
		ReleaseDate:      releaseDate,
		DeliverDate:      deliverDate,
		IsCarReview:      defaultString(row.IsCarReview, "0"),
		IsGrayVerifyPlan: defaultString(row.IsGrayVerifyPlan, "0"),
		VerifyDate:       defaultString(row.VerifyDate, "1"),
		VerifyPlan:       row.VerifyPlan,
		Verifier:         verifier,
		Accepter:         row.Accepter,
		Windows:          windows,
		Users:            users,
		Precheck: DeliverPrecheck{
			Rows:        precheckRows,
			CanSubmit:   canSubmit,
			BlockReason: blockReason,
		},
	}

	return resp, nil
}

// DeliverDemand 处理发起交付提交请求（完整校验、状态流转、写库同步）。
func (s *Service) DeliverDemand(ctx context.Context, actor *model.User, req DemandDeliverReq) error {
	if req.ID == 0 {
		return errHomeActionNotFound
	}
	account := ""
	if actor != nil {
		account = strings.TrimSpace(actor.Account)
	}
	if account == "" {
		return errHomeActionForbidden
	}

	row, err := s.repo.FindDeliverDemand(ctx, req.ID)
	if err != nil {
		return err
	}

	// 权限检查：必须为 RoleOwner
	homeRow := &homeActionDemandRow{
		ID:            row.ID,
		Status:        row.Status,
		Deleted:       row.Deleted,
		AssignedTo:    row.AssignedTo,
		DistributedBy: row.DistributedBy,
		CreatedBy:     row.CreatedBy,
		SubmitedBy:    row.SubmitedBy,
		SubmitBy:      row.SubmitBy,
		QD:            row.QD,
		RD:            row.RD,
		BRA:           row.BRA,
		Accepter:      row.Accepter,
		VeriFier:      row.VeriFier,
		Product:       row.Product,
	}
	if !s.repo.homeActionAuthorized(homeRow, account, "deliver") {
		return errHomeActionForbidden
	}

	// 状态门拦截
	if row.Status != "acceptanced" && row.Status != "waitdeliver" {
		return errHomeActionConflict
	}

	// 阻塞检查：严重缺陷未清时阻断发起交付
	severeBugs, _, err := s.repo.CheckDeliverBlockers(ctx, req.ID)
	if err != nil {
		return fmt.Errorf("检查交付阻塞缺陷失败: %w", err)
	}
	if severeBugs > 0 {
		return fmt.Errorf("存在 %d 个严重缺陷未关闭，禁止发起交付", severeBugs)
	}

	comment := strings.TrimSpace(req.Comment)
	if comment == "" {
		comment = "工作台发起交付"
	}

	params := DeliverWriteParams{
		WindowID:         req.WindowID,
		DeliverDate:      req.DeliverDate,
		IsCarReview:      req.IsCarReview,
		IsGrayVerifyPlan: req.IsGrayVerifyPlan,
		VerifyDate:       req.VerifyDate,
		VerifyPlan:       req.VerifyPlan,
		Verifier:         req.Verifier,
		Actor:            account,
		Comment:          comment,
	}

	return s.repo.UpdateDemandDeliverFull(ctx, req.ID, params)
}

func boolStr(ok bool, t, f string) string {
	if ok {
		return t
	}
	return f
}

func testBlockerValue(severeBugs, openBugs int) string {
	if severeBugs > 0 {
		return fmt.Sprintf("%d 个严重缺陷未闭环", severeBugs)
	}
	if openBugs > 0 {
		return fmt.Sprintf("无严重缺陷（%d 个普通缺陷）", openBugs)
	}
	return "无阻塞"
}

func defaultString(v, fallback string) string {
	if strings.TrimSpace(v) == "" {
		return fallback
	}
	return strings.TrimSpace(v)
}
