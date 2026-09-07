// =============================================================================
// 文件: internal/module/po/form_detail.go
// 模块: PO 工作台
// 类型: action
// 职责: 业务需求统一详情（五大 Tab、父子导航、去重优化）Req/Resp 结构体。
// 依赖: 无
// =============================================================================

package po

import (
	"strconv"
	"strings"

	"workbench/internal/module/po/primaryaction"
)

// DemandDetailReq 是需求详情查询入参。
type DemandDetailReq struct {
	ID string `form:"id" uri:"id"`
}

// ExtractDemandID 从入参字符串（支持 US123 或 123）解析为 uint ID。
func (r *DemandDetailReq) ExtractDemandID() uint {
	raw := strings.TrimSpace(r.ID)
	raw = strings.TrimPrefix(raw, "US")
	raw = strings.TrimPrefix(raw, "us")
	val, err := strconv.ParseUint(raw, 10, 64)
	if err != nil {
		return 0
	}
	return uint(val)
}

// Validate 校验需求 ID。
func (r *DemandDetailReq) Validate() []FieldError {
	if r.ExtractDemandID() == 0 {
		return []FieldError{{Field: "id", Message: "需求 ID 无效"}}
	}
	return nil
}

// DemandDetailResp 是业务需求统一详情输出。
type DemandDetailResp struct {
	Success         bool                         `json:"success"`
	Mode            string                       `json:"mode"` // parentAggregate | childUnit | selfUnit
	Summary         DemandSummary                `json:"summary"`
	ParentAggregate *ParentAggregateData         `json:"parentAggregate,omitempty"`
	RelationContext *RelationContextData         `json:"relationContext,omitempty"`
	ValueStream     *DetailValueStream           `json:"valueStream,omitempty"`
	Spotlight       *DetailSpotlight             `json:"spotlight,omitempty"`
	Requirement     *DetailRequirement           `json:"requirement,omitempty"`
	Execution       *DetailExecution             `json:"execution,omitempty"`
	Delivery        *DetailDelivery              `json:"delivery,omitempty"`
	History         *DetailHistory               `json:"history,omitempty"`
	Config          DetailConfig                 `json:"config"`
	PrimaryAction   *primaryaction.PrimaryAction `json:"primaryAction,omitempty"` // Stage 5: 服务端主操作
}

// DetailConfig 集中配置（前端禁止 hardcode）。
type DetailConfig struct {
	DeliveryCycleTargetDays int `json:"deliveryCycleTargetDays"`
}

// DemandSummary 需求通用摘要。
type DemandSummary struct {
	ID               string `json:"id"`
	Code             string `json:"code"`
	DemandID         uint   `json:"demandId"`
	Title            string `json:"title"`
	Source           string `json:"source"`
	SourceNote       string `json:"sourceNote"`
	Category         string `json:"category"`
	BSA              string `json:"bsa"`
	Duration         string `json:"duration"`
	FeedbackedBy     string `json:"feedbackedBy"`
	ProposerName     string `json:"proposerName"`
	ProposerDept     string `json:"proposerDept"`
	Originator       string `json:"originator"`
	OwnerName        string `json:"ownerName"`
	TestOwner        string `json:"testOwner"`
	Product          string `json:"product"`
	PoolName         string `json:"poolName"`
	Priority         string `json:"priority"`
	Status           string `json:"status"`
	ZentaoStatus     string `json:"zentaoStatus"`
	ValueStage       string `json:"valueStage"`
	ValueStageLabel  string `json:"valueStageLabel"`
	EstimateLaunch   string `json:"estimateLaunch"`
	AcceptOwner      string `json:"acceptOwner"`
	Reviewer         string `json:"reviewer"`
	CurrentOwner     string `json:"currentOwner"`
	MainSystem       string `json:"mainSystem"`
	MainSystemName   string `json:"mainSystemName"`
	Desc             string `json:"desc"`
	VerifyPlan       string `json:"verifyPlan"`
	EstimateDelivery int    `json:"estimateDelivery"`
	DevelopFinish    string `json:"developFinish"`
	TestFinish       string `json:"testFinish"`
	VerifyFinish     string `json:"verifyFinish"`
	StoriesCount     int    `json:"storiesCount"`
	TasksDone        int    `json:"tasksDone"`
	TasksTotal       int    `json:"tasksTotal"`
	CasesExecuted    int    `json:"casesExecuted"`
	CasesTotal       int    `json:"casesTotal"`
	BugsUnresolved   int    `json:"bugsUnresolved"`
	AcceptanceStatus string `json:"acceptanceStatus"`
	CreatedDate      string `json:"createdDate"`
	EditedDate       string `json:"editedDate"`
}

// ParentAggregateData 父需求聚合专用视图数据。
type ParentAggregateData struct {
	UnitTotal      int                `json:"unitTotal"`
	UnitOnline     int                `json:"unitOnline"`
	RisksCount     int                `json:"risksCount"`
	AttentionItems []AttentionItem    `json:"attentionItems"`
	DeliveryUnits  []DeliveryUnitItem `json:"deliveryUnits"`
}

// AttentionItem 父需求需要重点关注的子需求项。
type AttentionItem struct {
	DemandID uint   `json:"demandId"`
	Code     string `json:"code"`
	Title    string `json:"title"`
	Stage    string `json:"stage"`
	Owner    string `json:"owner"`
	RiskDesc string `json:"riskDesc"`
	IsRisk   bool   `json:"isRisk"`
}

// DeliveryUnitItem 子业务需求/交付单元行。
type DeliveryUnitItem struct {
	DemandID   uint   `json:"demandId"`
	Code       string `json:"code"`
	Title      string `json:"title"`
	Stage      string `json:"stage"`
	Owner      string `json:"owner"`
	StoriesNum int    `json:"storiesNum"`
	TasksNum   int    `json:"tasksNum"`
	LaunchDate string `json:"launchDate"`
	IsDone     bool   `json:"isDone"`
	HasRisk    bool   `json:"hasRisk"`
}

// RelationContextData 父子同级导航条。
type RelationContextData struct {
	Parent   *RelationDemand  `json:"parent,omitempty"`
	Siblings []RelationDemand `json:"siblings,omitempty"`
}

// RelationDemand 关系导航中的单条需求。
type RelationDemand struct {
	DemandID   uint   `json:"demandId"`
	Code       string `json:"code"`
	Title      string `json:"title"`
	Stage      string `json:"stage"`
	Owner      string `json:"owner"`
	LaunchDate string `json:"launchDate"`
	IsCurrent  bool   `json:"isCurrent"`
	HasRisk    bool   `json:"hasRisk"`
	IsDone     bool   `json:"isDone"`
}

// DetailValueStream 价值流周期与阶段耗时。
type DetailValueStream struct {
	EstimatedCycleDays int               `json:"estimatedCycleDays"`
	UsedCycleDays      int               `json:"usedCycleDays"`
	TargetCycleDays    int               `json:"targetCycleDays"`
	DiffCycleDays      int               `json:"diffCycleDays"`
	IsOverdue          bool              `json:"isOverdue"`
	Stages             []ValueStreamItem `json:"stages"`
}

// ValueStreamItem 9 阶段中的一阶段。
type ValueStreamItem struct {
	Key          string `json:"key"`
	Label        string `json:"label"`
	Role         string `json:"role"`
	Status       string `json:"status"` // done | current | future
	DurationText string `json:"durationText"`
	DurationKind string `json:"durationKind"` // 实际 | 已持续 | 预计
}

// DetailSpotlight 当前待办聚光灯。
type DetailSpotlight struct {
	Badge         string `json:"badge"`
	Title         string `json:"title"`
	Desc          string `json:"desc"`
	ActionLabel   string `json:"actionLabel"`
	TargetTab     string `json:"targetTab"`
	TargetSection string `json:"targetSection"`
}

// DetailRequirement Tab 2：需求与澄清。
type DetailRequirement struct {
	SpecHtml       string              `json:"specHtml"`
	VerifyHtml     string              `json:"verifyHtml"`
	Clarifications []ClarificationItem `json:"clarifications"`
	UserStories    []UserStoryItem     `json:"userStories"`
	Attachments    []AttachmentItem    `json:"attachments"`
	ClarifyZtURL   string              `json:"clarifyZtUrl"`
}

// ClarificationItem 产品维度澄清说明。
type ClarificationItem struct {
	Product     string `json:"product"`
	ProductName string `json:"productName"`
	Analyst     string `json:"analyst"`
	Content     string `json:"content"`
	DevEnd      string `json:"devEnd"`
	TestEnd     string `json:"testEnd"`
}

// UserStoryItem 用户故事。
type UserStoryItem struct {
	Role       string `json:"role"`
	Scene      string `json:"scene"`
	System     string `json:"system"`
	StoryPoint string `json:"storyPoint"`
	VerifyDesc string `json:"verifyDesc"`
}

// AttachmentItem 附件。
type AttachmentItem struct {
	ID       uint   `json:"id"`
	Title    string `json:"title"`
	Size     string `json:"size"`
	Created  string `json:"created"`
	Download string `json:"download"`
}

// DetailExecution Tab 3：研发与测试。
type DetailExecution struct {
	Stories          []StoryItem         `json:"stories"`
	TestOrders       []TestOrderItem     `json:"testOrders"`
	TestOrderSummary TestOrderSummary    `json:"testOrderSummary"`
	TestCaseSummary  TestCaseSummary     `json:"testCaseSummary"`
	BugSummary       BugSummary          `json:"bugSummary"`
	QualitySummary   QualitySummary      `json:"qualitySummary"`
	QualityOverview  QualityGateOverview `json:"qualityOverview"`
	AppQualityTree   []AppQualityNode    `json:"appQualityTree"`
	LeadsMatrix      LeadsMatrix         `json:"leadsMatrix"`
}

// StoryItem 关联的研发需求。
type StoryItem struct {
	ID         uint   `json:"id"`
	Code       string `json:"code"`
	Title      string `json:"title"`
	Product    string `json:"product"`
	Owner      string `json:"owner"`
	Status     string `json:"status"`
	TasksDone  int    `json:"tasksDone"`
	TasksTotal int    `json:"tasksTotal"`
	BugsTotal  int    `json:"bugsTotal"`
	BugsActive int    `json:"bugsActive"`
}

// TestOrderItem 测试单。
type TestOrderItem struct {
	ID          uint   `json:"id"`
	Code        string `json:"code"`
	Title       string `json:"title"`
	Stage       string `json:"stage"` // SIT | UAT
	Status      string `json:"status"`
	StatusLabel string `json:"statusLabel"`
	Owner       string `json:"owner"`
	BeginDate   string `json:"beginDate"`
	EndDate     string `json:"endDate"`
	ZtURL       string `json:"ztUrl"`
}

// TestOrderSummary 测试单统计。
type TestOrderSummary struct {
	TotalCount   int `json:"totalCount"`
	DoingCount   int `json:"doingCount"`
	DoneCount    int `json:"doneCount"`
	StoriesCount int `json:"storiesCount"`
}

// TestCaseSummary 用例统计。
type TestCaseSummary struct {
	TotalCount      int     `json:"totalCount"`
	ExecutedCount   int     `json:"executedCount"`
	UnexecutedCount int     `json:"unexecutedCount"`
	PassedCount     int     `json:"passedCount"`
	FailedCount     int     `json:"failedCount"`
	ExecutionRate   float64 `json:"executionRate"`
	PassRate        float64 `json:"passRate"`
}

// BugSummary 缺陷统计。
type BugSummary struct {
	TotalCount       int `json:"totalCount"`
	ActiveCount      int `json:"activeCount"`
	ResolvedCount    int `json:"resolvedCount"`
	DeliveryBlocking int `json:"deliveryBlocking"`
}

// QualitySummary 代码质量统计（保留向后兼容）。
// Available=false 表示扫描源未接入，AvgScore/门禁数不得解读为真实结果。
type QualitySummary struct {
	Available     bool    `json:"available"`
	Source        string  `json:"source"` // none | scanner
	AvgScore      float64 `json:"avgScore"`
	BranchesCount int     `json:"branchesCount"`
	PassedGates   int     `json:"passedGates"`
	TotalGates    int     `json:"totalGates"`
}

// QualityGateOverview 顶部卡片使用的应用门禁概况。
// Available=false 时前端应展示「未接入」，不得把 0 当成「全部通过」。
type QualityGateOverview struct {
	Available     bool   `json:"available"`
	Source        string `json:"source"`        // none | scanner
	AppsCount     int    `json:"appsCount"`     // 涉及应用/系统数
	BranchesCount int    `json:"branchesCount"` // 关联代码分支数
	PassedGates   int    `json:"passedGates"`   // 最新 MR 门禁通过数
	FailedGates   int    `json:"failedGates"`   // 最新 MR 门禁失败数
}

// AppQualityNode 按系统/应用组织的代码质量树节点。
type AppQualityNode struct {
	AppName     string              `json:"appName"`     // 系统/应用名称
	AppCode     string              `json:"appCode"`     // 应用标识 (如 mobile-hall-client)
	GatePassed  bool                `json:"gatePassed"`  // 整体门禁是否通过（仅 Available 时有效）
	Available   bool                `json:"available"`   // 是否有真实扫描结果
	BranchCount int                 `json:"branchCount"` // 分支数
	Branches    []BranchQualityItem `json:"branches"`    // 该应用下研发需求对应的代码分支及最新 MR
}

// BranchQualityItem 研发需求分支及最新 MR 扫描明细。
// GateStatus: pass | fail | unknown；Available=false 时分数/覆盖率无业务含义。
type BranchQualityItem struct {
	StoryCode  string  `json:"storyCode"`  // 关联研发需求禅道原始 ID
	StoryTitle string  `json:"storyTitle"` // 研需标题
	BranchName string  `json:"branchName"` // 分支名称；未接入时为空
	LatestMR   string  `json:"latestMr"`   // 最新 MR 编号；未接入时为空
	GateStatus string  `json:"gateStatus"` // pass | fail | unknown
	Available  bool    `json:"available"`
	Score      float64 `json:"score"`
	BugsCount  int     `json:"bugsCount"`
	CodeSmells int     `json:"codeSmells"`
	Coverage   float64 `json:"coverage"`
	ScanTime   string  `json:"scanTime"`
	Committer  string  `json:"committer"`
}

// LeadsMatrix 研发与测试负责人。
type LeadsMatrix struct {
	DevLeads []string `json:"devLeads"`
	TestLead string   `json:"testLead"`
}

// DetailDelivery Tab 4：交付上线。
type DetailDelivery struct {
	DevDone          bool   `json:"devDone"`
	TestPassed       bool   `json:"testPassed"`
	BugsResolved     bool   `json:"bugsResolved"`
	AcceptanceDone   bool   `json:"acceptanceDone"`
	EstimateLaunch   string `json:"estimateLaunch"`
	PublishWindow    string `json:"publishWindow"`
	VerifyConclusion string `json:"verifyConclusion"`
}

// DetailHistory Tab 5：过程记录。
type DetailHistory struct {
	Actions        []ActionHistoryItem `json:"actions"`
	StageDurations []StageDurationItem `json:"stageDurations"`
	Lifecycle      DemandLifecycle     `json:"lifecycle"`
}

// ActionHistoryItem 禅道 Action 历史。
type ActionHistoryItem struct {
	Date   string `json:"date"`
	Actor  string `json:"actor"`
	Action string `json:"action"`
	Extra  string `json:"extra"`
}

// StageDurationItem 阶段耗时。
type StageDurationItem struct {
	Stage     string `json:"stage"`
	StartDate string `json:"startDate"`
	EndDate   string `json:"endDate"`
	Duration  string `json:"duration"`
	Kind      string `json:"kind"` // 实际 | 已持续 | 预测
	Role      string `json:"role"`
}

// DemandLifecycle 需求生命周期经办人。
type DemandLifecycle struct {
	CreatedBy      string `json:"createdBy"`
	CreatedDate    string `json:"createdDate"`
	AssignedTo     string `json:"assignedTo"`
	AssignedDate   string `json:"assignedDate"`
	Reviewer       string `json:"reviewer"`
	ReviewedDate   string `json:"reviewedDate"`
	LastEditedBy   string `json:"lastEditedBy"`
	LastEditedDate string `json:"lastEditedDate"`
	ClosedBy       string `json:"closedBy"`
	ClosedDate     string `json:"closedDate"`
	ClosedReason   string `json:"closedReason"`
}
