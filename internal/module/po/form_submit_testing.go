package po

// SubmitTestPageData 是提测办理页面的服务端视图模型。
type SubmitTestPageData struct {
	DemandID          uint
	Title             string
	Stage             string
	RawStatus         string
	MainSystemName    string
	EstimateLaunch    string
	BRAName           string
	RDName            string
	QDName            string
	HandlerName       string
	Eligible          bool
	Reason            string
	SubmitUnavailable string
	Units             []SubmitTestUnit
	ZentaoURL         string
}

// SubmitTestUnit 是一个按产品、分支和执行聚合的原子提测单元。
type SubmitTestUnit struct {
	Product       uint
	ProductName   string
	Branch        uint
	Execution     uint
	ExecutionName string
	Project       uint
	BranchLabel   string
	Stories       []SubmitTestStory
}

type SubmitTestStory struct {
	ID    uint
	Title string
}
