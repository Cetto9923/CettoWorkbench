// =============================================================================
// 文件: internal/module/metrics/catalog.go
// 模块: 指标管理 (metrics)
// 类型: catalog（指标元数据唯一事实源 SSOT）
// 职责: 定义 21 条指标元数据目录（code/name/category/unit/description/sourceType/
//       direction/target/danger/order/enabled），以及「禅道聚合 vs 外部未接入」
//       的数据来源分层。前端 manage/radar 一律从 /metrics/api 读取本目录，
//       不再各自维护指标字典。
// 依赖: 无
// =============================================================================

package metrics

import (
	"fmt"
	"strconv"
)

// sourceType 是数据来源分层枚举。
const (
	// SourceZentao 禅道实时聚合（repo.Snapshot 提供真实 SQL）。
	SourceZentao = "zentao"
	// SourceExternal 外部数据源（FineReport / DevOps / 门禁），尚未接入。
	SourceExternal = "external"
)

// metricDef 是指标元数据目录中的一条定义。
//
// 两阈值模型（target=达标线 / danger=危险线，direction 定方向）：
//   - up：  value>=target → normal；danger<=value<target → warn；value<danger → danger
//   - down：value<=target → normal；target<value<=danger → warn；value>danger → danger
//
// warning 是 target 与 danger 之间的派生中间态，不设独立阈值字段。
type metricDef struct {
	Code        string  // 唯一英文标识符
	Name        string  // 中文名
	Category    string  // 5 分类之一（ValidCategories）
	Unit        string  // "%" / "个" / "天" / "—"
	Description string  // 指标口径
	SourceType  string  // SourceZentao | SourceExternal
	SourceLabel string  // 人类可读数据源
	Period      string  // 实时 / 月度 / 日
	Direction   string  // up（越大越好）| down（越小越好）
	Target      string  // 达标线展示文本，如 "≥90%" / "≤30" / "—"
	Danger      string  // 危险线展示文本，如 "60%" / "60" / "—"
	TargetValue float64 // 达标线数值（无目标 = -1）
	DangerValue float64 // 危险线数值（无阈值 = -1）
	OwnerRole   string  // PO / SM / PMO / 测试 等
	Formula     string  // 派生公式 / SQL 摘要
	Order       int     // 展示顺序
	Enabled     bool    // 是否启用
}

// metricCatalog 是 21 条指标元数据目录（唯一 SSOT）。
// 顺序按 5 分类分组：需求治理(5) / 交付效率(5) / 研发质量(6) / 规范执行(2) / 效能管理(3)。
var metricCatalog = []metricDef{
	// ===== 需求治理（5）=====
	{
		Code: "story.total", Name: "研发需求总量", Category: CategoryDemand,
		Unit: "个", Description: "研发需求（Story）的累计总数，反映长期需求池规模",
		SourceType: SourceZentao, SourceLabel: "禅道 zt_story", Period: "实时",
		Direction: "up", Target: "—", Danger: "—", TargetValue: -1, DangerValue: -1,
		OwnerRole: "PO", Formula: "COUNT(zt_story WHERE deleted='0')", Order: 1, Enabled: true,
	},
	{
		Code: "story.active", Name: "进行中研发需求", Category: CategoryDemand,
		Unit: "个", Description: "未关闭/未发布的研发需求数，反映当前需求在研压力",
		SourceType: SourceZentao, SourceLabel: "禅道 zt_story", Period: "实时",
		Direction: "down", Target: "≤30", Danger: "60", TargetValue: 30, DangerValue: 60,
		OwnerRole: "PO", Formula: "COUNT(zt_story WHERE status NOT IN ('closed','released'))", Order: 2, Enabled: true,
	},
	{
		Code: "story.overIteration", Name: "超两迭代周期占比", Category: CategoryDemand,
		Unit: "%", Description: "实施周期超过 42 天的工单占该看板小组进行中工单的比例",
		SourceType: SourceExternal, SourceLabel: "FineReport (hNBB) / 禅道看板", Period: "月度",
		Direction: "down", Target: "≤10%", Danger: "20%", TargetValue: 10, DangerValue: 20,
		OwnerRole: "PO", Formula: "实施周期>42天需求数 ÷ 看板进行中需求数 * 100%", Order: 3, Enabled: true,
	},
	{
		Code: "story.unscheduled", Name: "超2周未排期单数", Category: CategoryDemand,
		Unit: "个", Description: "评审通过超 14 天仍未进行需求澄清的单数",
		SourceType: SourceExternal, SourceLabel: "FineReport (hNBB) / 禅道 zt_demand", Period: "月度",
		Direction: "down", Target: "≤3个", Danger: "8个", TargetValue: 3, DangerValue: 8,
		OwnerRole: "PO", Formula: "COUNT(zt_demand WHERE 评审通过超14天未澄清)", Order: 4, Enabled: true,
	},
	{
		Code: "story.delayedLaunch", Name: "上线延期数", Category: CategoryDemand,
		Unit: "个", Description: "实际发布时间晚于预计上线时间的业务需求单数",
		SourceType: SourceExternal, SourceLabel: "FineReport (hNBB) / 禅道需求", Period: "月度",
		Direction: "down", Target: "0个", Danger: "2个", TargetValue: 0, DangerValue: 2,
		OwnerRole: "PO", Formula: "COUNT(releasedDate > estimateLaunch)", Order: 5, Enabled: true,
	},

	// ===== 交付效率（5）=====
	{
		Code: "delivery.cycle", Name: "交付周期", Category: CategoryDelivery,
		Unit: "天", Description: "业务需求从业务评审通过到完成上线的天数，扣除外部挂起天数",
		SourceType: SourceExternal, SourceLabel: "FineReport (JbuB) / 禅道 zt_demand.teamGroup", Period: "月度",
		Direction: "down", Target: "≤30天", Danger: "50天", TargetValue: 30, DangerValue: 50,
		OwnerRole: "PO", Formula: "AVG((实际发布时间 - 业务评审通过时间) - 挂起天数)", Order: 6, Enabled: true,
	},
	{
		Code: "implement.cycle", Name: "实施周期", Category: CategoryDelivery,
		Unit: "天", Description: "业务需求从首次需求澄清到完成上线的天数，扣除挂起天数",
		SourceType: SourceExternal, SourceLabel: "FineReport (JbuB) / 禅道 zt_demand.teamGroup", Period: "月度",
		Direction: "down", Target: "≤20天", Danger: "35天", TargetValue: 20, DangerValue: 35,
		OwnerRole: "PO", Formula: "AVG((实际发布时间 - 首次需求澄清时间) - 挂起天数)", Order: 7, Enabled: true,
	},
	{
		Code: "story.doneRate", Name: "研发需求完成率", Category: CategoryDelivery,
		Unit: "%", Description: "已关闭/已发布研发需求占总需求的比例，反映整体交付完成度",
		SourceType: SourceZentao, SourceLabel: "禅道 zt_story", Period: "实时",
		Direction: "up", Target: "≥90%", Danger: "60%", TargetValue: 90, DangerValue: 60,
		OwnerRole: "PO", Formula: "(closed+released) / total", Order: 8, Enabled: true,
	},
	{
		Code: "story.closedOnTimeRate", Name: "按时关闭率", Category: CategoryDelivery,
		Unit: "%", Description: "已关闭研发需求中，实际关闭日期未晚于预计关闭日期的比例",
		SourceType: SourceZentao, SourceLabel: "禅道 zt_story", Period: "近 30 天",
		Direction: "up", Target: "≥80%", Danger: "60%", TargetValue: 80, DangerValue: 60,
		OwnerRole: "PO", Formula: "SUM(closedDate<=estimatedLaunch) / SUM(closed+released)", Order: 9, Enabled: true,
	},
	{
		Code: "task.overdue", Name: "逾期任务数", Category: CategoryDelivery,
		Unit: "个", Description: "截止日期早于今天且未关闭/取消的任务数",
		SourceType: SourceZentao, SourceLabel: "禅道 zt_task", Period: "实时",
		Direction: "down", Target: "≤5", Danger: "10", TargetValue: 5, DangerValue: 10,
		OwnerRole: "PO", Formula: "COUNT(zt_task WHERE status NOT IN ('closed','cancel') AND deadline<today)", Order: 10, Enabled: true,
	},

	// ===== 研发质量（6）=====
	{
		Code: "bug.open", Name: "未关闭缺陷", Category: CategoryQuality,
		Unit: "个", Description: "未关闭/未取消的缺陷总数，反映当前缺陷压力",
		SourceType: SourceZentao, SourceLabel: "禅道 zt_bug", Period: "实时",
		Direction: "down", Target: "≤5", Danger: "10", TargetValue: 5, DangerValue: 10,
		OwnerRole: "测试", Formula: "COUNT(zt_bug WHERE status NOT IN ('closed','cancelled'))", Order: 11, Enabled: true,
	},
	{
		Code: "bug.p1p2", Name: "致命/严重缺陷数", Category: CategoryQuality,
		Unit: "个", Description: "严重程度为 1 级（致命）或 2 级（严重）的未关闭缺陷数",
		SourceType: SourceZentao, SourceLabel: "禅道 zt_bug", Period: "实时",
		Direction: "down", Target: "≤2", Danger: "5", TargetValue: 2, DangerValue: 5,
		OwnerRole: "测试", Formula: "COUNT(zt_bug WHERE severity IN ('1','2') AND status NOT IN ('closed','cancelled'))", Order: 12, Enabled: true,
	},
	{
		Code: "bug.resolutionRate", Name: "缺陷解决率", Category: CategoryQuality,
		Unit: "%", Description: "已解决/已关闭缺陷占总缺陷数的比例",
		SourceType: SourceZentao, SourceLabel: "禅道 zt_bug", Period: "近 30 天",
		Direction: "up", Target: "≥85%", Danger: "70%", TargetValue: 85, DangerValue: 70,
		OwnerRole: "测试", Formula: "(resolved+closed) / total", Order: 13, Enabled: true,
	},
	{
		Code: "gate.passRate", Name: "质量门禁通过率", Category: CategoryQuality,
		Unit: "%", Description: "通过质量门禁检查的研发需求占需门禁研发需求的比例",
		SourceType: SourceExternal, SourceLabel: "DevOps / 门禁平台", Period: "月度",
		Direction: "up", Target: "≥90%", Danger: "80%", TargetValue: 90, DangerValue: 80,
		OwnerRole: "开发组长", Formula: "门禁通过数 ÷ 需门禁需求总数 * 100%", Order: 14, Enabled: true,
	},
	{
		Code: "bug.closeRate", Name: "缺陷关闭率", Category: CategoryQuality,
		Unit: "%", Description: "该看板敏捷小组名下处理的缺陷关闭完成情况",
		SourceType: SourceExternal, SourceLabel: "禅道 zt_bug / 看板团队成员", Period: "月度",
		Direction: "up", Target: "≥90%", Danger: "80%", TargetValue: 90, DangerValue: 80,
		OwnerRole: "测试负责人", Formula: "关闭缺陷数 ÷ 缺陷总数 * 100%", Order: 15, Enabled: true,
	},
	{
		Code: "bug.responseRate", Name: "缺陷响应效率", Category: CategoryQuality,
		Unit: "%", Description: "致命 1 天、严重 3 天、一般 5 天内解决响应效率",
		SourceType: SourceExternal, SourceLabel: "禅道 zt_bug", Period: "月度",
		Direction: "up", Target: "≥85%", Danger: "75%", TargetValue: 85, DangerValue: 75,
		OwnerRole: "开发组长", Formula: "各等级限期内解决缺陷数 ÷ 对应等级缺陷总数 * 100%", Order: 16, Enabled: true,
	},

	// ===== 规范执行（2，门禁未同步 → 外部未接入）=====
	{
		Code: "norm.completeness", Name: "业务需求富文本完整性", Category: CategoryCompliance,
		Unit: "%", Description: "业务需求富文本正文（非空描述+字段完整）的覆盖率；门禁数据未同步",
		SourceType: SourceExternal, SourceLabel: "门禁（待同步）", Period: "日",
		Direction: "up", Target: "≥95%", Danger: "80%", TargetValue: 95, DangerValue: 80,
		OwnerRole: "PMO", Formula: "COUNT(demand WHERE description!='') / COUNT(demand)", Order: 17, Enabled: true,
	},
	{
		Code: "norm.owner", Name: "责任人必填覆盖率", Category: CategoryCompliance,
		Unit: "%", Description: "需求/任务/Bug 中责任人字段非空的比例；门禁数据未同步",
		SourceType: SourceExternal, SourceLabel: "门禁（待同步）", Period: "日",
		Direction: "up", Target: "≥98%", Danger: "90%", TargetValue: 98, DangerValue: 90,
		OwnerRole: "PMO", Formula: "COUNT(obj WHERE assignedTo!='') / COUNT(obj)", Order: 18, Enabled: true,
	},

	// ===== 效能管理（3）=====
	{
		Code: "task.total", Name: "任务总量", Category: CategoryPerformance,
		Unit: "个", Description: "任务（含已完成）的累计总数，反映长期任务池规模",
		SourceType: SourceZentao, SourceLabel: "禅道 zt_task", Period: "实时",
		Direction: "up", Target: "—", Danger: "—", TargetValue: -1, DangerValue: -1,
		OwnerRole: "SM", Formula: "COUNT(zt_task WHERE deleted='0')", Order: 19, Enabled: true,
	},
	{
		Code: "task.open", Name: "未完成任务", Category: CategoryPerformance,
		Unit: "个", Description: "未关闭/未取消的任务数，反映当前任务压力",
		SourceType: SourceZentao, SourceLabel: "禅道 zt_task", Period: "实时",
		Direction: "down", Target: "≤20", Danger: "40", TargetValue: 20, DangerValue: 40,
		OwnerRole: "SM", Formula: "COUNT(zt_task WHERE status NOT IN ('closed','cancel'))", Order: 20, Enabled: true,
	},
	{
		Code: "sp.deviationRate", Name: "SP偏差率", Category: CategoryPerformance,
		Unit: "%", Description: "看板小组业务需求规模（故事点）与实际人力投入偏差率",
		SourceType: SourceExternal, SourceLabel: "FineReport (Y3VO) / 禅道工时", Period: "月度",
		Direction: "down", Target: "≤15%", Danger: "35%", TargetValue: 15, DangerValue: 35,
		OwnerRole: "SM", Formula: "|实际工时 - (故事点 × 7小时)| ÷ (故事点 × 7小时)", Order: 21, Enabled: true,
	},
}

// valueFor 返回指标的运行值（禅道聚合）或 (0,false)（外部未接入）。
// rate 类指标返回百分比数值（0-100）；count 类返回整数。
func valueFor(d metricDef, snap snapshot) (float64, bool) {
	if d.SourceType != SourceZentao {
		return 0, false // 外部 / 门禁未接入 → unavailable
	}
	switch d.Code {
	case "story.total":
		return float64(snap.Stories), true
	case "story.active":
		return float64(snap.StoriesActive), true
	case "story.doneRate":
		if snap.Stories == 0 {
			return 0, false
		}
		return float64(snap.StoriesDone) * 100 / float64(snap.Stories), true
	case "story.closedOnTimeRate":
		if snap.StoriesDone == 0 {
			return 0, false
		}
		return float64(snap.StoriesClosedOnTime) * 100 / float64(snap.StoriesDone), true
	case "task.overdue":
		return float64(snap.TasksOverdue), true
	case "bug.open":
		return float64(snap.Bugs), true
	case "bug.p1p2":
		return float64(snap.BugsP1P2), true
	case "bug.resolutionRate":
		if snap.BugsTotal == 0 {
			return 0, false
		}
		return float64(snap.BugsResolved) * 100 / float64(snap.BugsTotal), true
	case "task.total":
		return float64(snap.Tasks), true
	case "task.open":
		return float64(snap.TasksOpen), true
	default:
		return 0, false
	}
}

// evaluateStatus 用两阈值模型判定状态：normal / warn / danger / unknown。
// hasData=false → unknown（unavailable，不参与评分/均值/排名/趋势）。
// 无目标（target 与 danger 均为 -1）→ 恒 normal（纯累计计数指标）。
func evaluateStatus(d metricDef, value float64, hasData bool) string {
	if !hasData {
		return "unknown"
	}
	if d.TargetValue < 0 && d.DangerValue < 0 {
		return "normal"
	}
	if d.Direction == "down" {
		switch {
		case value <= d.TargetValue:
			return "normal"
		case value <= d.DangerValue:
			return "warn"
		default:
			return "danger"
		}
	}
	// up
	switch {
	case value >= d.TargetValue:
		return "normal"
	case value >= d.DangerValue:
		return "warn"
	default:
		return "danger"
	}
}

// formatValue 把运行值渲染成展示文本；hasData=false → "—"。
func formatValue(unit string, value float64, hasData bool) string {
	if !hasData {
		return "—"
	}
	if unit == "%" {
		return fmt.Sprintf("%.0f%%", value)
	}
	return strconv.FormatInt(int64(value), 10)
}
