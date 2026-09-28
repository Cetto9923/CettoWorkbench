// =============================================================================
// 文件: internal/module/po/form_deliver.go
// 模块: PO 工作台
// 类型: form
// 职责: 发起交付表单读写结构体与校验规则（对齐禅道 demand/deliver）。
// =============================================================================

package po

import (
	"strings"
	"time"
)

// DeliverPrecheckRow 交付前置检查单行状态。
type DeliverPrecheckRow struct {
	Label string `json:"label"`
	Value string `json:"value"`
	OK    bool   `json:"ok"`
}

// DeliverPrecheck 交付前置检查结果。
type DeliverPrecheck struct {
	Rows        []DeliverPrecheckRow `json:"rows"`
	CanSubmit   bool                 `json:"canSubmit"`
	BlockReason string               `json:"blockReason"`
}

// DeliverWindowOption 上线窗口下拉选项。
type DeliverWindowOption struct {
	ID          uint   `json:"id"`
	Name        string `json:"name"`
	ReleaseDate string `json:"releaseDate"`
}

// DemandDeliverMetaResp 发起交付表单初始化数据（GET /demands/:id/deliver）。
type DemandDeliverMetaResp struct {
	Success          bool                  `json:"success"`
	DemandID         uint                  `json:"demandId"`
	DisplayID        string                `json:"displayId"`
	Title            string                `json:"title"`
	Status           string                `json:"status"`
	ZentaoStatus     string                `json:"zentaoStatus"`
	WindowID         uint                  `json:"windowId"`
	WindowName       string                `json:"windowName"`
	ReleaseDate      string                `json:"releaseDate"`
	DeliverDate      string                `json:"deliverDate"`
	IsCarReview      string                `json:"isCarReview"`
	IsGrayVerifyPlan string                `json:"isGrayVerifyPlan"`
	VerifyDate       string                `json:"verifyDate"`
	VerifyPlan       string                `json:"verifyPlan"`
	Verifier         string                `json:"verifier"`
	VerifierName     string                `json:"verifierName"`
	Accepter         string                `json:"accepter"`
	AccepterName     string                `json:"accepterName"`
	Windows          []DeliverWindowOption `json:"windows"`
	Users            []ClarifyOption       `json:"users"`
	Precheck         DeliverPrecheck       `json:"precheck"`
}

// DemandDeliverReq 发起交付提交请求体（POST /demands/:id/deliver）。
type DemandDeliverReq struct {
	ID               uint   `json:"id"`
	Mode             string `json:"mode"`
	WindowID         uint   `json:"windowId"`
	DeliverDate      string `json:"deliverDate"`
	IsCarReview      string `json:"isCarReview"`
	IsGrayVerifyPlan string `json:"isGrayVerifyPlan"`
	VerifyDate       string `json:"verifyDate"`
	VerifyPlan       string `json:"verifyPlan"`
	Verifier         string `json:"verifier"`
	Comment          string `json:"comment"`
}

// Validate 校验发起交付请求参数。
func (r *DemandDeliverReq) Validate() []FieldError {
	var errs []FieldError
	if r.ID == 0 {
		errs = append(errs, FieldError{Field: "id", Message: "需求 ID 无效"})
	}
	r.Mode = strings.TrimSpace(r.Mode)
	if r.Mode == "" {
		r.Mode = "create"
	}
	if r.Mode != "create" && r.Mode != "edit" && r.Mode != "cancel" {
		errs = append(errs, FieldError{Field: "mode", Message: "不支持的交付操作"})
	}
	r.DeliverDate = strings.TrimSpace(r.DeliverDate)
	r.IsCarReview = strings.TrimSpace(r.IsCarReview)
	r.IsGrayVerifyPlan = strings.TrimSpace(r.IsGrayVerifyPlan)
	r.VerifyDate = strings.TrimSpace(r.VerifyDate)
	r.VerifyPlan = strings.TrimSpace(r.VerifyPlan)
	r.Verifier = strings.TrimSpace(r.Verifier)
	r.Comment = strings.TrimSpace(r.Comment)

	if r.Mode != "cancel" {
		if len(r.DeliverDate) != 10 || r.DeliverDate[4] != '-' || r.DeliverDate[7] != '-' {
			errs = append(errs, FieldError{Field: "deliverDate", Message: "上线时间格式无效"})
		} else if r.DeliverDate < time.Now().Format("2006-01-02") {
			errs = append(errs, FieldError{Field: "deliverDate", Message: "上线时间不能早于今天"})
		}
		if r.IsGrayVerifyPlan == "" {
			errs = append(errs, FieldError{Field: "isGrayVerifyPlan", Message: "灰度验证计划不能为空"})
		}
		if r.VerifyDate == "" {
			errs = append(errs, FieldError{Field: "verifyDate", Message: "生产验证时间不能为空"})
		}
		if r.VerifyPlan == "" {
			errs = append(errs, FieldError{Field: "verifyPlan", Message: "生产验证计划不能为空"})
		}
		if r.Verifier == "" {
			errs = append(errs, FieldError{Field: "verifier", Message: "生产验证责任人不能为空"})
		}
	}
	return errs
}
