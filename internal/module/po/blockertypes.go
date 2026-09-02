// =============================================================================
// 文件: internal/module/po/blockertypes.go
// 模块: PO 工作台
// 类型: readonly
// 职责: 卡点(PO 首页"卡点快速响应"区块)的常量、枚举与请求/响应结构体定义。
//       拆分自 blocker.go:常量与类型独立后,后续加新等级/新字段不影响主 Service
//       文件头注释合规。
// 依赖: 无
// =============================================================================

package po

// BlockerLevel 卡点等级。
type BlockerLevel string

const (
	BlockerLevelBlocked BlockerLevel = "blocked" // 关键路径已逾期且核心人未到岗(最重)
	BlockerLevelOverdue BlockerLevel = "overdue" // 已跨过截止日(任一关键日 ≤ today)
	BlockerLevelRisk    BlockerLevel = "risk"    // 关键日缺失或临近(≤ today+3)
	BlockerLevelCoord   BlockerLevel = "coord"   // 当前账号是主责任人,但日期未达
)

// BlockerStageLimit 单阶段取数上限(避免单阶段吞掉全部限额)。
const BlockerStageLimit = 4

// BlockerOverallLimit 整体上限(越上限滚动,渲染层负责可滚动)。
const BlockerOverallLimit = 24

// blockerStatuses 卡点拉取的价值流阶段(热环节)。
// 与现有 service.go 的 mysqlStageFilters key 对齐;新增 waitdeliver 是真实业务
// 状态但 PO 当前 filters 未覆盖,因此 blockerStatuses 不包含它。
var blockerStatuses = []string{
	"clarify",        // 澄清
	"schedule",       // 排期
	"developing",     // 提测
	"testing",        // 联调测试
	"waitacceptance", // 验收
	"acceptanced",    // 交付
}

// BlockerReq 卡点请求(预留扩展)。
type BlockerReq struct{}

// Validate 校验(无入参)。
func (r *BlockerReq) Validate() []FieldError {
	return nil
}

// BlockerDetail 单条卡点。
type BlockerDetail struct {
	Kind        string       `json:"kind"` // demand / story
	ID          string       `json:"id"`
	Level       BlockerLevel `json:"level"`
	LevelLabel  string       `json:"levelLabel"`
	Title       string       `json:"title"`
	Owner       string       `json:"owner"`
	DueAt       string       `json:"dueAt"`    // YYYY-MM-DD
	DueLabel    string       `json:"dueLabel"` // "今日已超 3 天" / "距今 2 天" / "今日"
	Stage       string       `json:"stage"`
	ZentaoUrl   string       `json:"zentaoUrl"`
	IsOwnAction bool         `json:"isOwnAction"`
}

// BlockerResp 卡点响应。
type BlockerResp struct {
	Items []BlockerDetail `json:"items"`
}
