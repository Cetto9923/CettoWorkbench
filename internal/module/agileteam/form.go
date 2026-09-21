// =============================================================================
// 文件: internal/module/agileteam/form.go
// 模块: 敏捷小组治理
// 类型: action
// 职责: 列表/详情/调整提交/确认驳回 Req 与响应结构。
// 依赖: 无
// =============================================================================

package agileteam

import (
	"strings"
	"time"
	"unicode/utf8"
)

// FieldError 表单字段错误；表单级错误使用 "_form"。
type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// ListReq 敏捷小组列表筛选。
type ListReq struct {
	Name         string `form:"name"`
	ParentName   string `form:"parentName"`
	Coach        string `form:"coach"`
	PO           string `form:"po"`
	Member       string `form:"member"`
	AdjustStatus string `form:"adjustStatus"` // all / pending / none
	Status       string `form:"status"`       // all / enable / disable
	View         string `form:"view"`         // pmo / lead
	Scope        string `form:"scope"`        // team / dept（仅 lead）
	ScopeID      uint   `form:"scopeId"`
	Page         int    `form:"page"`
	PageSize     int    `form:"pageSize"`
}

func (r *ListReq) Normalize() {
	r.Name = strings.TrimSpace(r.Name)
	r.ParentName = strings.TrimSpace(r.ParentName)
	r.Coach = strings.TrimSpace(r.Coach)
	r.PO = strings.TrimSpace(r.PO)
	r.Member = strings.TrimSpace(r.Member)
	r.AdjustStatus = strings.TrimSpace(r.AdjustStatus)
	r.Status = strings.TrimSpace(r.Status)
	r.View = strings.ToLower(strings.TrimSpace(r.View))
	r.Scope = strings.ToLower(strings.TrimSpace(r.Scope))
	if r.AdjustStatus == "" {
		r.AdjustStatus = "all"
	}
	if r.Status == "" {
		r.Status = "all"
	}
	if r.View != "lead" {
		r.View = "pmo"
	}
	if r.Scope != "dept" {
		r.Scope = "team"
	}
	if r.Page <= 0 {
		r.Page = 1
	}
	if r.PageSize <= 0 {
		r.PageSize = 20
	}
	// 夹逼到 [5, 200]：保留 20/50/100 三个预设档的常用体验，
	// 同时放开自定义入口（前端下拉里的"自定义…"会写入任意整数）。
	if r.PageSize < 5 {
		r.PageSize = 5
	}
	if r.PageSize > 200 {
		r.PageSize = 200
	}
}

// CandidateSearchReq 成员候选人搜索。
type CandidateSearchReq struct {
	Q           string `form:"q"`
	TeamgroupID uint   `form:"teamgroupId"`
}

func (r *CandidateSearchReq) Normalize() {
	r.Q = strings.TrimSpace(r.Q)
}

// CandidateItem 成员候选人。
type CandidateItem struct {
	Account string `json:"account"`
	Name    string `json:"name"`
	Pinyin  string `json:"pinyin,omitempty"`
}

// MemberItem 正式/待加入成员展示行。
type MemberItem struct {
	Account   string  `json:"account"`
	Name      string  `json:"name"`
	Role      string  `json:"role"`
	Hours     float64 `json:"hours"`
	JoinDate  string  `json:"joinDate"`
	Status    string  `json:"status"` // formal / pendingAdd / pendingRemove
	Submitter string  `json:"submitter,omitempty"`
}

// PendingSummary 待确认调整摘要。
type PendingSummary struct {
	AdjustmentID int64    `json:"adjustmentId"`
	AdjustNo     string   `json:"adjustNo"`
	SubmittedBy  string   `json:"submittedBy"`
	SubmittedAt  string   `json:"submittedAt"`
	Status       string   `json:"status"`
	Reason       string   `json:"reason"`
	AddCount     int      `json:"addCount"`
	RemoveCount  int      `json:"removeCount"`
	ChangeCount  int      `json:"changeCount"`
	AddNames     []string `json:"addNames"`
	RemoveNames  []string `json:"removeNames"`
	ChangeNames  []string `json:"changeNames"`
}

// HistoryItem 调整记录时间线。
type HistoryItem struct {
	EventType    string `json:"eventType"`
	Summary      string `json:"summary"`
	Actor        string `json:"actor"`
	ActorName    string `json:"actorName"`
	CreatedAt    string `json:"createdAt"`
	AdjustmentID int64  `json:"adjustmentId,omitempty"`
}

// ParentOption 父级小组下拉。
type ParentOption struct {
	ID   uint   `json:"id"`
	Name string `json:"name"`
}

// ScopeOption 团队管理视图维度选项。
type ScopeOption struct {
	ID   uint   `json:"id"`
	Name string `json:"name"`
}

// ListItem 列表行。
type ListItem struct {
	ID              uint   `json:"id"`
	Name            string `json:"name"`
	ParentID        uint   `json:"parentId"`
	ParentName      string `json:"parentName"`
	Type            string `json:"type"`
	ChildCount      int    `json:"childCount"`
	CoachAccount    string `json:"coachAccount"`
	CoachName       string `json:"coachName"`
	POAccount       string `json:"poAccount"`
	POName          string `json:"poName"`
	FormalCount     int    `json:"formalCount"`
	PendingAdd      int    `json:"pendingAdd"`
	PendingRemove   int    `json:"pendingRemove"`
	Status          string `json:"status"`
	StatusLabel     string `json:"statusLabel"`
	LastAdjustAt    string `json:"lastAdjustAt"`
	PendingAdjustID int64  `json:"pendingAdjustId,omitempty"`
}

// ListResp 列表响应。
type ListResp struct {
	Items        []ListItem    `json:"items"`
	Total        int64         `json:"total"`
	AllCount     int64         `json:"allCount"`
	EnableCount  int64         `json:"enableCount"`
	DisableCount int64         `json:"disableCount"`
	PendingCount int64         `json:"pendingCount"`
	Page         int           `json:"page"`
	PageSize     int           `json:"pageSize"`
	PageCount    int           `json:"pageCount"`
	ScopeOptions []ScopeOption `json:"scopeOptions,omitempty"`
	CanEdit      bool          `json:"canEdit"`
}

// DetailResp 详情响应。
type DetailResp struct {
	ID            uint            `json:"id"`
	Name          string          `json:"name"`
	ParentID      uint            `json:"parentId"`
	ParentName    string          `json:"parentName"`
	CoachAccount  string          `json:"coachAccount"`
	CoachName     string          `json:"coachName"`
	POAccount     string          `json:"poAccount"`
	POName        string          `json:"poName"`
	Slogan        string          `json:"slogan"`
	Declaration   string          `json:"declaration"`
	Logo          string          `json:"logo"`
	Status        string          `json:"status"`
	StatusLabel   string          `json:"statusLabel"`
	CreatedDate   string          `json:"createdDate"`
	FormalCount   int             `json:"formalCount"`
	Formal        []MemberItem    `json:"formal"`
	PendingJoin   []MemberItem    `json:"pendingJoin"`
	Pending       *PendingSummary `json:"pending,omitempty"`
	History       []HistoryItem   `json:"history"`
	CanConfirm    bool            `json:"canConfirm"`
	CanEdit       bool            `json:"canEdit"`
	ParentOptions []ParentOption  `json:"parentOptions,omitempty"`
}

// UpdateBasicReq 基本信息保存（不走确认流）。
type UpdateBasicReq struct {
	ID          uint   `json:"-"`
	Name        string `json:"name"`
	Slogan      string `json:"slogan"`
	Declaration string `json:"declaration"`
	Logo        string `json:"logo"`
	ParentID    *uint  `json:"parentId"`
}

func (r *UpdateBasicReq) Validate() []FieldError {
	r.Name = strings.TrimSpace(r.Name)
	r.Slogan = strings.TrimSpace(r.Slogan)
	r.Declaration = strings.TrimSpace(r.Declaration)
	r.Logo = strings.TrimSpace(r.Logo)
	var errs []FieldError
	if r.ID == 0 {
		errs = append(errs, FieldError{Field: "_form", Message: "小组 ID 无效"})
	}
	if r.Name == "" {
		errs = append(errs, FieldError{Field: "name", Message: "团队名称不能为空"})
	} else if utf8.RuneCountInString(r.Name) > 100 || hasControlRune(r.Name) {
		errs = append(errs, FieldError{Field: "name", Message: "团队名称过长或包含非法字符"})
	}
	if utf8.RuneCountInString(r.Slogan) > 200 || hasControlRune(r.Slogan) {
		errs = append(errs, FieldError{Field: "slogan", Message: "团队口号过长或包含非法字符"})
	}
	if utf8.RuneCountInString(r.Declaration) > 2000 || hasControlRune(r.Declaration) {
		errs = append(errs, FieldError{Field: "declaration", Message: "团队信条过长或包含非法字符"})
	}
	if utf8.RuneCountInString(r.Logo) > 500 || hasControlRune(r.Logo) || invalidLogoURL(r.Logo) {
		errs = append(errs, FieldError{Field: "logo", Message: "Logo 过长、包含非法字符或使用了不允许的地址"})
	}
	return errs
}

// AdjustItemReq 单条调整。
type AdjustItemReq struct {
	Account        string  `json:"account"`
	ActionType     string  `json:"actionType"`
	Role           string  `json:"role"`
	AvailableHours float64 `json:"availableHours"`
}

// SubmitAdjustmentReq 提交成员调整。
type SubmitAdjustmentReq struct {
	TeamgroupID uint            `json:"-"`
	Reason      string          `json:"reason"`
	Items       []AdjustItemReq `json:"items"`
}

func (r *SubmitAdjustmentReq) Validate() []FieldError {
	r.Reason = strings.TrimSpace(r.Reason)
	var errs []FieldError
	if r.TeamgroupID == 0 {
		errs = append(errs, FieldError{Field: "_form", Message: "小组 ID 无效"})
	}
	if len(r.Items) == 0 {
		errs = append(errs, FieldError{Field: "items", Message: "请至少提交一条调整"})
	}
	seen := map[string]bool{}
	for i, it := range r.Items {
		acc := strings.TrimSpace(it.Account)
		act := strings.TrimSpace(it.ActionType)
		r.Items[i].Account = acc
		r.Items[i].ActionType = act
		r.Items[i].Role = strings.TrimSpace(it.Role)
		if acc == "" {
			errs = append(errs, FieldError{Field: "items", Message: "成员账号不能为空"})
			continue
		}
		if seen[acc] {
			errs = append(errs, FieldError{Field: "items", Message: "同一调整单内账号不能重复：" + acc})
		}
		seen[acc] = true
		if act != ActionAdd && act != ActionRemove && act != ActionRoleChange {
			errs = append(errs, FieldError{Field: "items", Message: "调整类型无效：" + act})
		}
	}
	return errs
}

// ConfirmReq 确认调整。
type ConfirmReq struct {
	AdjustmentID int64 `json:"-"`
}

// RejectReq 驳回调整。
type RejectReq struct {
	AdjustmentID int64  `json:"-"`
	Reason       string `json:"reason"`
}

func (r *RejectReq) Validate() []FieldError {
	r.Reason = strings.TrimSpace(r.Reason)
	if r.AdjustmentID <= 0 {
		return []FieldError{{Field: "_form", Message: "调整单 ID 无效"}}
	}
	if r.Reason == "" {
		return []FieldError{{Field: "reason", Message: "请填写驳回原因"}}
	}
	if utf8.RuneCountInString(r.Reason) > 500 {
		return []FieldError{{Field: "reason", Message: "驳回原因不能超过 500 个字符"}}
	}
	return nil
}

// AdjustmentDetailResp 确认 Drawer 用详情。
type AdjustmentDetailResp struct {
	ID          int64            `json:"id"`
	TeamgroupID uint             `json:"teamgroupId"`
	TeamName    string           `json:"teamName"`
	AdjustNo    string           `json:"adjustNo"`
	Status      string           `json:"status"`
	Reason      string           `json:"reason"`
	SubmittedBy string           `json:"submittedBy"`
	SubmittedAt string           `json:"submittedAt"`
	Items       []AdjustmentLine `json:"items"`
	CanConfirm  bool             `json:"canConfirm"`
}

// AdjustmentLine Drawer 差异行。
type AdjustmentLine struct {
	Account        string  `json:"account"`
	Name           string  `json:"name"`
	ActionType     string  `json:"actionType"`
	Role           string  `json:"role"`
	PrevRole       string  `json:"prevRole"`
	AvailableHours float64 `json:"availableHours"`
	PrevHours      float64 `json:"prevHours"`
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format("2006-01-02 15:04")
}

func formatDate(t *time.Time) string {
	if t == nil || t.IsZero() {
		return ""
	}
	return t.Format("2006-01-02")
}

func statusLabel(status string) string {
	switch status {
	case "enable", "doing":
		return "启用"
	case "disable", "closed", "suspended":
		return "停用"
	default:
		if status == "" {
			return "启用"
		}
		return status
	}
}
