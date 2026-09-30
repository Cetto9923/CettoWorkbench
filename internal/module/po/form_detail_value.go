// =============================================================================
// 文件: internal/module/po/form_detail_value.go
// 模块: PO 工作台
// 类型: form
// 职责: 定义业务需求详情页价值模型（ValueModel）与配置结构体。
// 依赖: 无
// =============================================================================

package po

// DemandValueModel 需求价值模型与费用估算数据块（仅在总开关开启时返回）。
type DemandValueModel struct {
	Enabled                bool     `json:"enabled"`
	Skipped                bool     `json:"skipped"`
	DemandValue            *float64 `json:"demandValue"`
	DemandValueDate        string   `json:"demandValueDate"`
	IsParent               bool     `json:"isParent"`
	ChildEligible          int      `json:"childEligible"`
	ChildEstimated         int      `json:"childEstimated"`
	ChildSum               bool     `json:"childSum"`
	CostLow                *float64 `json:"costLow"`
	CostHigh               *float64 `json:"costHigh"`
	IntervalKind           string   `json:"intervalKind"`
	IntervalPercent        *float64 `json:"intervalPercent"`
	CostPerMonthConfigured bool     `json:"costPerMonthConfigured"`
	CostAvailable          bool     `json:"costAvailable"`
	CostUnavailableReason  string   `json:"costUnavailableReason"`
}

// DemandValueConfig 从 zt_config 读取的价值模型系统配置快照。
type DemandValueConfig struct {
	Enabled              bool
	CostPerMonth         *float64
	IntervalMethod       string
	IntervalFixedPercent *float64
	NoAiCategories       []string
}

// ChildDemandValueRow 子需求参与价值估算取数行。
type ChildDemandValueRow struct {
	ID          uint
	Category    string
	DemandValue *float64
}
