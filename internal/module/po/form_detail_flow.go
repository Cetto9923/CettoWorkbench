// =============================================================================
// 文件: internal/module/po/form_detail_flow.go
// 模块: PO 工作台
// 类型: form / model
// 职责: 需求详情 B3 流程与审批（变更记录、挂起日志、评审记录、主管部门审批）及
//       管理信息（7 项重要检查项、8 项实际时间）结构体定义。
// =============================================================================

package po

// DetailFlowApproval 流程与审批（原型「流程与审批」页签，四块）。
type DetailFlowApproval struct {
	ManagerReviews []DemandManagerReviewItem `json:"managerReviews,omitempty"`
	DemandChanges  []DemandChangeItem        `json:"demandChanges,omitempty"`
	HangLogs       []DemandHangLogItem       `json:"hangLogs,omitempty"`
	ReviewRecords  []DemandReviewRecordItem  `json:"reviewRecords,omitempty"`
}

// DemandChangeItem 需求变更记录。
type DemandChangeItem struct {
	ID                   uint   `json:"id"`
	DemandID             uint   `json:"demandId"`
	ChangeBy             string `json:"changeBy"`
	ChangeByName         string `json:"changeByName"`
	ChangeDate           string `json:"changeDate"`
	ChangeType           string `json:"changeType"`
	ChangeTypeLabel      string `json:"changeTypeLabel"`
	ReasonType           string `json:"reasonType"`
	ChangeReason         string `json:"changeReason"`
	Desc                 string `json:"desc"`
	OldDesc              string `json:"oldDesc"`
	ChangeMainDept       string `json:"changeMainDept"`
	ChangeReviewer       string `json:"changeReviewer"`
	ChangeReviewerName   string `json:"changeReviewerName"`
	Result               string `json:"result"`
	ResultLabel          string `json:"resultLabel"`
	CreatedDate          string `json:"createdDate"`
	EstimateLaunchChange string `json:"estimateLaunchChange,omitempty"`
	DevelopFinishChange  string `json:"developFinishChange,omitempty"`
	TestFinishChange     string `json:"testFinishChange,omitempty"`
	VerifyFinishChange   string `json:"verifyFinishChange,omitempty"`
}

// DemandHangLogItem 需求挂起日志。
type DemandHangLogItem struct {
	ID              uint   `json:"id"`
	Account         string `json:"account"`
	AccountName     string `json:"accountName"`
	Action          string `json:"action"`
	ActionLabel     string `json:"actionLabel"`
	HangUpType      string `json:"hangUpType"`
	HangUpTypeLabel string `json:"hangUpTypeLabel"`
	Date            string `json:"date"`
	Extra           string `json:"extra"`
	Comment         string `json:"comment"`
}

// DemandReviewRecordItem 评审记录。
type DemandReviewRecordItem struct {
	ID                uint   `json:"id"`
	ReviewType        string `json:"reviewType"`
	ReviewTypeLabel   string `json:"reviewTypeLabel"`
	ReviewDate        string `json:"reviewDate"`
	ReviewResult      string `json:"reviewResult"`
	ReviewStatus      string `json:"reviewStatus"`
	ReviewStatusLabel string `json:"reviewStatusLabel"`
	CreatedBy         string `json:"createdBy"`
	CreatedByName     string `json:"createdByName"`
	CreatedDate       string `json:"createdDate"`
}

// DemandManagerReviewItem 主管部门审批记录。
type DemandManagerReviewItem struct {
	ID                  uint     `json:"id"`
	Result              string   `json:"result"`
	ResultLabel         string   `json:"resultLabel"`
	Reviewer            string   `json:"reviewer"`
	ReviewerName        string   `json:"reviewerName"`
	Comment             string   `json:"comment"`
	SubmitedBy          string   `json:"submitedBy"`
	SubmitedByName      string   `json:"submitedByName"`
	SubmitedDate        string   `json:"submitedDate"`
	ReviewDate          string   `json:"reviewDate"`
	ResultStatus        string   `json:"resultStatus"`
	Products            []string `json:"products,omitempty"`
	Departments         []string `json:"departments,omitempty"`
	DepartmentReviewers []string `json:"departmentReviewers,omitempty"`
}

// DetailManagementInfo 详情管理信息段（重要检查项与实际时间）。
type DetailManagementInfo struct {
	ImportantChecks *DemandImportantChecks `json:"importantChecks,omitempty"`
	ActualTimes     *DemandActualTimes     `json:"actualTimes,omitempty"`
}

// DemandImportantChecks 7 项重要检查项。
type DemandImportantChecks struct {
	MultiLegalPersonLogo      string `json:"multiLegalPersonLogo,omitempty"`
	MultiLegalPersonLogoLabel string `json:"multiLegalPersonLogoLabel,omitempty"`
	IsRelatedAccounts         string `json:"isRelatedAccounts,omitempty"`
	IsRelatedAccountsLabel    string `json:"isRelatedAccountsLabel,omitempty"`
	IsImportantOrder          string `json:"isImportantOrder,omitempty"`
	IsImportantOrderLabel     string `json:"isImportantOrderLabel,omitempty"`
	IsNeedReview              string `json:"isNeedReview,omitempty"`
	IsNeedReviewLabel         string `json:"isNeedReviewLabel,omitempty"`
	OnetimeAcceptance         string `json:"onetimeAcceptance,omitempty"`
	OnetimeAcceptanceLabel    string `json:"onetimeAcceptanceLabel,omitempty"`
	IsCarReview               string `json:"isCarReview,omitempty"`
	IsCarReviewLabel          string `json:"isCarReviewLabel,omitempty"`
	VerifyDate                string `json:"verifyDate,omitempty"`
	VerifyDateLabel           string `json:"verifyDateLabel,omitempty"`
}

// DemandActualTimes 8 项实际时间。
type DemandActualTimes struct {
	ClarifyDate              string `json:"clarifyDate,omitempty"`              // 首次澄清时间
	FirstToStoryDate         string `json:"firstToStoryDate,omitempty"`         // 首次转研发时间 (同 actualDevStartDate)
	ActualDevStartDate       string `json:"actualDevStartDate,omitempty"`       // 实际开发开始时间
	ActualTestStartDate      string `json:"actualTestStartDate,omitempty"`      // 实际测试开始时间
	ActualDevCompletionDate  string `json:"actualDevCompletionDate,omitempty"`  // 实际开发完成时间 (同 actualTestStartDate)
	SubmitAcceptanceDate     string `json:"submitAcceptanceDate,omitempty"`     // 提交验收时间
	ActualTestCompletionDate string `json:"actualTestCompletionDate,omitempty"` // 实际测试完成时间 (同 submitAcceptanceDate)
	AcceptancedDate          string `json:"acceptancedDate,omitempty"`          // 实际验收完成时间
	ReviewedDate             string `json:"reviewedDate,omitempty"`             // 业务评审时间
	DeliverDate              string `json:"deliverDate,omitempty"`              // 交付时间
	RealReleaseDate          string `json:"realReleaseDate,omitempty"`          // 实际发布时间
}
