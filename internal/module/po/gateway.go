// =============================================================================
// 文件: internal/module/po/gateway.go
// 模块: PO 工作台
// 类型: action
// 职责: 出站适配层：封装业需评审 / 撤回评审 / 发起交付的禅道 REST 调用。
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

// reviewDemandViaZentaoReq 禅道 POST /demand/:id/review 入参。
type reviewDemandViaZentaoReq struct {
	DemandID int64
	Result   string // pass / refuse
	Comment  string
}

// reviewDemandViaZentao 以当前登录账号（ctx）调用禅道评审接口。
func reviewDemandViaZentao(ctx context.Context, client *zentao.Client, req reviewDemandViaZentaoReq) error {
	if client == nil {
		return fmt.Errorf("禅道 API 未配置")
	}
	if req.DemandID <= 0 {
		return fmt.Errorf("需求 ID 无效")
	}
	result := strings.TrimSpace(req.Result)
	if result != "pass" && result != "refuse" {
		return fmt.Errorf("评审结果无效")
	}
	payload := map[string]any{
		"result": result,
	}
	if comment := strings.TrimSpace(req.Comment); comment != "" {
		payload["comment"] = comment
	}
	path := fmt.Sprintf("/demand/%d/review", req.DemandID)
	return client.Do(ctx, http.MethodPost, path, payload, nil)
}

// withdrawDemandReviewViaZentaoReq 禅道 POST /demand/:id/withdrawReview 入参。
type withdrawDemandReviewViaZentaoReq struct {
	DemandID int64
	Comment  string
}

// withdrawDemandReviewViaZentao 以当前登录账号（ctx）调用禅道撤回评审接口。
// comment 始终下发；未传时置为空字符串。
func withdrawDemandReviewViaZentao(ctx context.Context, client *zentao.Client, req withdrawDemandReviewViaZentaoReq) error {
	if client == nil {
		return fmt.Errorf("禅道 API 未配置")
	}
	if req.DemandID <= 0 {
		return fmt.Errorf("需求 ID 无效")
	}
	payload := map[string]any{
		"comment": strings.TrimSpace(req.Comment),
	}
	path := fmt.Sprintf("/demand/%d/withdrawReview", req.DemandID)
	return client.Do(ctx, http.MethodPost, path, payload, nil)
}

// deliverDemandViaZentaoReq 禅道 POST /demand/:id/deliver 入参。
type deliverDemandViaZentaoReq struct {
	DemandID         int64
	DeliverDate      string
	IsGrayVerifyPlan string
	VerifyDate       string
	VerifyPlan       string
	VeriFier         string
	IsCarReview      string
}

// deliverDemandViaZentao 以当前登录账号（ctx）调用禅道发起交付接口。
func deliverDemandViaZentao(ctx context.Context, client *zentao.Client, req deliverDemandViaZentaoReq) error {
	if client == nil {
		return fmt.Errorf("禅道 API 未配置")
	}
	if req.DemandID <= 0 {
		return fmt.Errorf("需求 ID 无效")
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
	}
	path := fmt.Sprintf("/demand/%d/deliver", req.DemandID)
	return client.Do(ctx, http.MethodPost, path, payload, nil)
}
