package schedule

// DemandSchedulingResp 排期一体化弹窗加载数据。
type DemandSchedulingResp struct {
	*DemandSchedulingDetail
	InvolvedProducts   []ZtProductOption                          `json:"involvedProducts"`
	ProductProjects    map[string][]DemandSchedulingProjectOption `json:"productProjects"`
	ProjectExecutions  map[string][]ZtExecutionOption             `json:"projectExecutions"`
	Stories            []DemandSchedulingStoryItem                `json:"stories"`
	UserStories        []UserStoryItem                            `json:"userStories"`
	StoryDefaults      []DemandSchedulingStoryDefault             `json:"storyDefaults"`
	WindowProductPlans []SchedulingWindowProductPlan              `json:"windowProductPlans"`
	ProductPlans       map[string][]SchedulingProductPlanOption   `json:"productPlans"`
	Windows            []SchedulingWindowOption                   `json:"windows"`
	Users              []SchedulingUserOption                     `json:"users"`
}

// DemandSchedulingClarifyDefault 是澄清记录按系统归并后的研发需求默认值来源。
type DemandSchedulingClarifyDefault struct {
	ProductID   uint
	ProductName string
	Analyst     string
}

// DemandSchedulingStoryDefault 是转研发需求时按禅道二开规则生成的默认值。
type DemandSchedulingStoryDefault struct {
	ProductID      uint   `json:"productId"`
	ProductName    string `json:"productName"`
	Title          string `json:"title"`
	Spec           string `json:"spec"`
	AssignedTo     string `json:"assignedTo"`
	AssignedToName string `json:"assignedToName"`
	PlanID         uint   `json:"planId"`
	PlanName       string `json:"planName"`
	Type           string `json:"type"`
	TypeLabel      string `json:"typeLabel"`
	Pri            int    `json:"pri"`
	Estimate       int    `json:"estimate"`
}

// SchedulingWindowProductPlan 是版本窗口下系统对应的禅道产品计划。
type SchedulingWindowProductPlan struct {
	WindowID  uint   `json:"windowId"`
	ProductID uint   `json:"productId"`
	PlanID    uint   `json:"planId"`
	PlanName  string `json:"planName"`
}

// SchedulingProductPlanOption 是同产品下可手工选择的研发需求计划。
type SchedulingProductPlanOption struct {
	ID        uint   `json:"id"`
	ProductID uint   `json:"productId"`
	Title     string `json:"title"`
}
