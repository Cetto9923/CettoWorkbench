// =============================================================================
// 文件: internal/module/testtask/form.go
// 模块: 提测办理
// 类型: action
// 职责: 提测上下文请求/响应结构。
// 依赖: 无
// =============================================================================

package testtask

// DemandContextRow 业需上下文查询行（zt_demand + 主系统名；人员为账号，展示名由 user map 解析）。
type DemandContextRow struct {
	ID             uint   `gorm:"column:id"`
	Name           string `gorm:"column:name"`
	Status         string `gorm:"column:status"`
	Stage          string `gorm:"column:stage"`
	BRA            string `gorm:"column:BRA"`
	RD             string `gorm:"column:RD"`
	QD             string `gorm:"column:QD"`
	MainSystemID   uint   `gorm:"column:main_system_id"`
	MainSystemName string `gorm:"column:product_name"`
	EstimateLaunch string `gorm:"column:estimate_launch"`
}

// SystemItem 需求涉及产品/系统（选择系统列表项）。
type SystemItem struct {
	ID     uint   `json:"id"`
	Name   string `json:"name"`
	IsMain bool   `json:"isMain"`
}

// ExecutionOption 所属执行下拉项（创建新版本）。
// Value 为「项目id-执行id」，Label 为「项目名称/执行名称」。
type ExecutionOption struct {
	Value       string `json:"value"`
	Label       string `json:"label"`
	ProjectID   uint   `json:"projectId"`
	ExecutionID uint   `json:"executionId"`
}

// ContextResp 提测弹窗「当前需求上下文」JSON。
type ContextResp struct {
	DemandID       uint         `json:"demandId"`
	Title          string       `json:"title"`
	Stage          string       `json:"stage"`
	RawStatus      string       `json:"rawStatus"`
	MainSystemName string       `json:"mainSystemName"`
	EstimateLaunch string       `json:"estimateLaunch"`
	BRAName        string       `json:"braName"`
	RDName         string       `json:"rdName"`
	QDName         string       `json:"qdName"`
	HandlerName    string       `json:"handlerName"`
	Systems        []SystemItem `json:"systems"`
}
