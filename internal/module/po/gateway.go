// =============================================================================
// 文件: internal/module/po/gateway.go
// 模块: PO 工作台
// 类型: action
// 职责: 出站适配层：封装业需发起评审 / 评审 / 撤回评审 / 发起交付 / 验收的禅道 REST 调用。
// 依赖: internal/pkg/zentao
// =============================================================================

package po

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"workbench/internal/pkg/zentao"
)

// deliverDemandViaZentaoReq 禅道 POST /demand/:id/deliver 入参。
type deliverDemandViaZentaoReq struct {
	DemandID         int64
	DeliverDate      string
	IsGrayVerifyPlan string
	VerifyDate       string
	VerifyPlan       string
	VeriFier         string
	IsCarReview      string
	Comment          string
}

func postZentaoDeliver(ctx context.Context, client *zentao.Client, path, invalidMsg string, req deliverDemandViaZentaoReq) error {
	if client == nil {
		return fmt.Errorf("禅道 API 未配置")
	}
	if req.DemandID <= 0 {
		return fmt.Errorf("%s", invalidMsg)
	}
	isCar := strings.TrimSpace(req.IsCarReview)
	if isCar == "" {
		isCar = "0"
	}
	payload := map[string]any{
		"deliverDate":      strings.TrimSpace(req.DeliverDate),
		"isGrayVerifyPlan": strings.TrimSpace(req.IsGrayVerifyPlan),
		"verifyDate":       strings.TrimSpace(req.VerifyDate),
		"verifyPlan":       strings.TrimSpace(req.VerifyPlan),
		"veriFier":         strings.TrimSpace(req.VeriFier),
		"isCarReview":      isCar,
		"comment":          strings.TrimSpace(req.Comment),
	}
	return client.Do(ctx, http.MethodPost, path, payload, nil)
}

// deliverDemandViaZentao 以当前登录账号（ctx）调用禅道发起交付接口。
func deliverDemandViaZentao(ctx context.Context, client *zentao.Client, req deliverDemandViaZentaoReq) error {
	return postZentaoDeliver(ctx, client, fmt.Sprintf("/demand/%d/deliver", req.DemandID), "需求 ID 无效", req)
}

// deliverStoryViaZentao 以当前登录账号调用禅道发起独立研发需求交付。
func deliverStoryViaZentao(ctx context.Context, client *zentao.Client, req deliverDemandViaZentaoReq) error {
	return postZentaoDeliver(ctx, client, fmt.Sprintf("/stories/%d/deliver", req.DemandID), "研发需求 ID 无效", req)
}

// acceptDemandViaZentaoReq 禅道 POST /demand/:id/acceptance 入参。
type acceptDemandViaZentaoReq struct {
	DemandID   int64
	Acceptance string // yes / no
	AssignedTo string
	Comment    string
}

// acceptDemandViaZentao 以当前登录账号（ctx）调用禅道验收接口。
func acceptDemandViaZentao(ctx context.Context, client *zentao.Client, req acceptDemandViaZentaoReq) error {
	if client == nil {
		return fmt.Errorf("禅道 API 未配置")
	}
	if req.DemandID <= 0 {
		return fmt.Errorf("需求 ID 无效")
	}
	acceptance := strings.TrimSpace(req.Acceptance)
	if acceptance != "yes" && acceptance != "no" {
		return fmt.Errorf("验收结果无效")
	}
	assignedTo := strings.TrimSpace(req.AssignedTo)
	if assignedTo == "" {
		return fmt.Errorf("指派给不能为空")
	}
	payload := map[string]any{
		"acceptance": acceptance,
		"assignedTo": assignedTo,
		"comment":    strings.TrimSpace(req.Comment),
	}
	path := fmt.Sprintf("/demand/%d/acceptance", req.DemandID)
	return client.Do(ctx, http.MethodPost, path, payload, nil)
}
