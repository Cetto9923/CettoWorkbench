// =============================================================================
// 文件: internal/module/po/form_version_follow.go
// 模块: PO 工作台
// 类型: form
// 职责: 版本跟进页面的 Req/Resp 数据契约与校验。
// =============================================================================

package po

import "time"

// aiPilotEnabled 控制「AI 发现」块是否渲染。
// 第一期仅用确定性规则，不接大模型；默认关闭时整块不渲染、页面不出现任何 AI 元素。
// 后续 AI 叠加层只需把这里换成读取试点名单，页面无需改动。
const aiPilotEnabled = false

// 发布判断三态，与设计文档「可按期 / 风险 / 阻塞」一致。
const (
	VFJudgementOnTrack = "ontrack"
	VFJudgementRisk    = "risk"
	VFJudgementBlocked = "blocked"
)

// VersionFollowListReq 版本跟进列表查询条件（GET，form tag）。
type VersionFollowListReq struct {
	WindowID  uint   `form:"windowId"`
	Scope     string `form:"scope" validate:"in=,todo,following"`
	Stage     string `form:"stage"`
	System    string `form:"system"`
	Owner     string `form:"owner"`
	Judgement string `form:"judgement"`
	Keyword   string `form:"keyword"`
	Page      int    `form:"page" validate:"min=1"`
	PageSize  int    `form:"pageSize" validate:"min=1,max=100"`
}

// Validate 手写校验，不引入 go-playground/validator。
func (r VersionFollowListReq) Validate() []FieldError {
	var errs []FieldError
	if r.Page < 1 {
		errs = append(errs, FieldError{Field: "page", Message: "页码必须大于 0"})
	}
	if r.PageSize < 1 || r.PageSize > 100 {
		errs = append(errs, FieldError{Field: "pageSize", Message: "每页条数需在 1-100 之间"})
	}
	if r.Scope != "" && r.Scope != "todo" && r.Scope != "following" {
		errs = append(errs, FieldError{Field: "scope", Message: "范围取值不合法"})
	}
	if r.Judgement != "" && r.Judgement != VFJudgementOnTrack &&
		r.Judgement != VFJudgementRisk && r.Judgement != VFJudgementBlocked {
		errs = append(errs, FieldError{Field: "judgement", Message: "发布判断取值不合法"})
	}
	return errs
}

// VersionFollowWindowResp 版本窗口条上的一枚窗口。
type VersionFollowWindowResp struct {
	ID           uint64 `json:"id"`
	Name         string `json:"name"`
	ReleaseDate  string `json:"releaseDate"`
	Status       string `json:"status"`
	DemandCount  int    `json:"demandCount"`
	RiskCount    int    `json:"riskCount"`
	BlockedCount int    `json:"blockedCount"`
	Released     bool   `json:"released"`
}

// VersionFollowStageCount 阶段分布进度条上的一段。
type VersionFollowStageCount struct {
	Stage string `json:"stage"`
	Count int    `json:"count"`
}

// VersionFollowAIFinding 一条确定性规则命中项；AI 试点开启时才出现在响应里。
type VersionFollowAIFinding struct {
	RuleName  string `json:"ruleName"`
	Field     string `json:"field"`
	Value     string `json:"value"`
	TargetURL string `json:"targetUrl"`
}

// VersionFollowItemResp 需求列表的一行。
type VersionFollowItemResp struct {
	DemandID    uint64 `json:"demandId"`
	DemandNo    string `json:"demandNo"`
	Title       string `json:"title"`
	System      string `json:"system"`
	Priority    string `json:"priority"`
	Stage       string `json:"stage"`
	StageStatus string `json:"stageStatus"`
	// StayDays 是「已停留 N 天」：需求停留在当前阶段的时长，替代原型的迷你进度列。
	StayDays int `json:"stayDays"`
	// Overdue 为超期提示；未超期为空串。
	Overdue       string                   `json:"overdue"`
	Judgement     string                   `json:"judgement"`
	Reason        string                   `json:"reason"`
	Owner         string                   `json:"owner"`
	ActionLabel   string                   `json:"actionLabel"`
	ActionURL     string                   `json:"actionUrl"`
	ActionEnabled bool                     `json:"actionEnabled"`
	ActionReason  string                   `json:"actionReason"`
	Findings      []VersionFollowAIFinding `json:"findings"`
}

// VersionFollowCheckItemResp 展开行的交付前置检查清单一项。
type VersionFollowCheckItemResp struct {
	Label  string `json:"label"`
	State  string `json:"state"`
	Detail string `json:"detail"`
	Phase2 bool   `json:"phase2"`
}

// VersionFollowItemDetailResp 展开行的附加内容。
type VersionFollowItemDetailResp struct {
	Checks      []VersionFollowCheckItemResp `json:"checks"`
	Stories     []string                     `json:"stories"`
	Tasks       []string                     `json:"tasks"`
	SourceName  string                       `json:"sourceName"`
	SourceItems []string                     `json:"sourceItems"`
}

// VersionFollowListResp 列表接口响应。
type VersionFollowListResp struct {
	WindowID     uint64                        `json:"windowId"`
	Items        []VersionFollowItemResp       `json:"items"`
	Details      []VersionFollowItemDetailResp `json:"details"`
	Windows      []VersionFollowWindowResp     `json:"windows"`
	StageCounts  []VersionFollowStageCount     `json:"stageCounts"`
	DistanceDays int                           `json:"distanceDays"`
	OnTrack      int                           `json:"onTrack"`
	Risk         int                           `json:"risk"`
	Blocked      int                           `json:"blocked"`
	Total        int64                         `json:"total"`
	Page         int                           `json:"page"`
	PageSize     int                           `json:"pageSize"`
	AIPilotOn    bool                          `json:"aiPilotOn"`
	// OrphanTestedCount 是「已到测试后阶段但未挂版本窗口」的需求数；仅 AI 试点开启时有值。
	OrphanTestedCount int `json:"orphanTestedCount"`
}

// 九阶段在首页价值流阶段表 valueStreamStages[1:] 中的序号，供规则直接比较，避免重复查表。
// 顺序与首页一致：受理 0 / 澄清 1 / 排期 2 / 提测 3 / 联调测试 4 / 验收 5 / 发起交付 6 / 发布 7 / 评价反馈 8。
const (
	stageIdxAccept         = 0
	stageIdxClarify        = 1
	stageIdxSchedule       = 2
	stageIdxDeveloping     = 3
	stageIdxTesting        = 4
	stageIdxWaitAcceptance = 5
	stageIdxAcceptanced    = 6
	stageIdxPublish        = 7
	stageIdxReleased       = 8
)

// nowFunc 便于规则单测注入固定时间。
var nowFunc = time.Now
