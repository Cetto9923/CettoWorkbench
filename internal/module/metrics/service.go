// =============================================================================
// 文件: internal/module/metrics/service.go
// 模块: 指标管理 (metrics)
// 类型: service
// 职责: 指标定义、运行快照派生、状态判定 (PRD §14/§15 "无数据显示 — 而非 0%")、
//       filter → page 切片、summary 聚合；可复用 helper 供 radar 复用。
// 依赖: internal/module/metrics/repo.go
// =============================================================================

package metrics

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// 5 个分类中文 label（前后端共用的合法枚举）。
const (
	CategoryDemand      = "需求治理"
	CategoryDelivery    = "交付效率"
	CategoryQuality     = "研发质量"
	CategoryCompliance  = "规范执行"
	CategoryPerformance = "效能管理"
)

// ValidCategories 是 ManageListReq.Category 的服务端白名单；前端 select 同步枚举。
var ValidCategories = []string{
	CategoryDemand,
	CategoryDelivery,
	CategoryQuality,
	CategoryCompliance,
	CategoryPerformance,
}

// IsValidCategory 返回 c 是否为 5 分类之一。
func IsValidCategory(c string) bool {
	for _, x := range ValidCategories {
		if x == c {
			return true
		}
	}
	return false
}

// Service 提供指标列表聚合能力；不持有业务规则之外的副作用。
type Service struct{ repo *Repo }

// NewService 装配 Service。
func NewService(repo *Repo) *Service { return &Service{repo: repo} }

// MetricStatusByRate 判定比率类指标的 status。
//   - total == 0 → "unknown"（PRD §14/§15，避免 0% 假风险）；
//   - goodDirection="up"（越大越好）：>=0.9 normal / >=0.7 warn / 否则 danger；
//   - goodDirection="down"（越小越好）：<=0.1 normal / <=0.3 warn / 否则 danger。
//
// 该 helper 与 MetricTextForRate 配套使用，供 radar 复用。
func MetricStatusByRate(value, total int64, goodDirection string) string {
	if total == 0 {
		return "unknown"
	}
	ratio := float64(value) / float64(total)
	if goodDirection == "up" {
		switch {
		case ratio >= 0.9:
			return "normal"
		case ratio >= 0.7:
			return "warn"
		default:
			return "danger"
		}
	}
	switch {
	case ratio <= 0.1:
		return "normal"
	case ratio <= 0.3:
		return "warn"
	default:
		return "danger"
	}
}

// MetricStatusByCount 判定计数类指标的 status。
//   - goodDirection="up"（越大越好）：>=danger normal / >=warningwarn / 否则 danger；
//   - goodDirection="down"（越小越好）：<=warning normal / <=danger warn / 否则 danger。
func MetricStatusByCount(value, warning, danger int64, goodDirection string) string {
	if goodDirection == "up" {
		switch {
		case value >= danger:
			return "normal"
		case value >= warning:
			return "warn"
		default:
			return "danger"
		}
	}
	switch {
	case value <= warning:
		return "normal"
	case value <= danger:
		return "warn"
	default:
		return "danger"
	}
}

// MetricTextForRate 把 ratio 渲染成 "85%" 或 "—"（total=0 占位）。
func MetricTextForRate(value, total int64) string {
	if total == 0 {
		return "—"
	}
	return fmt.Sprintf("%.0f%%", float64(value)*100/float64(total))
}

// MetricTextForCount 把 count 渲染成数字字符串；allowDash=true 时 0 → "—"（与 MetricStatus="unknown" 配套）。
func MetricTextForCount(value int64, allowDash bool) string {
	if allowDash && value == 0 {
		return "—"
	}
	return strconv.FormatInt(value, 10)
}

// MetricStatusLabel 把 status 翻译为前端展示用的中文 label（manage / radar 共用）。
func MetricStatusLabel(status string) string {
	switch status {
	case "normal":
		return "正常"
	case "warn":
		return "关注"
	case "danger":
		return "风险"
	case "unknown":
		return "暂无数据"
	default:
		return "—"
	}
}

// CategoryColorClass 把分类映射成 CSS 颜色修饰类（前端 manage / radar 共用）。
// 颜色取自个人工作台语义 token：--color-primary / --color-info / --color-warning / --color-danger / --color-success。
func CategoryColorClass(category string) string {
	switch category {
	case CategoryDemand:
		return "cat-demand"
	case CategoryDelivery:
		return "cat-delivery"
	case CategoryQuality:
		return "cat-quality"
	case CategoryCompliance:
		return "cat-compliance"
	case CategoryPerformance:
		return "cat-performance"
	default:
		return "cat-default"
	}
}

// List 返回 ManageListResp：先拉一次 Snapshot（11 个 COUNT 子查询，单 SQL），
// 拼装 items → 按 category/status/keyword 过滤 → 按 page/pageSize 切片 → 聚合 summary。
// 不引入 query-per-metric fan-out；过滤/分页在 12 条 bounded 列表上做（database.md）。
func (s *Service) List(ctx context.Context, req ManageListReq) (ManageListResp, error) {
	req.Normalize()
	if err := req.Validate(); err != nil {
		return ManageListResp{}, err
	}
	snap, err := s.repo.Snapshot(ctx)
	if err != nil {
		return ManageListResp{}, err
	}
	items := s.buildItems(snap, time.Now().Format("2006-01-02 15:04"))
	filtered := filterItems(items, req)
	total := int64(len(filtered))
	start := (req.Page - 1) * req.PageSize
	if start > int(total) {
		start = int(total)
	}
	end := start + req.PageSize
	if end > int(total) {
		end = int(total)
	}
	return ManageListResp{
		Summary:  computeSummary(items),
		Items:    filtered[start:end],
		Filter:   req,
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
	}, nil
}

// filterItems 在 12 条 bounded items 上按 category / status / keyword 过滤。
// keyword 大小写不敏感，匹配 Name / Code / DataSource / Source。
func filterItems(items []MetricItem, req ManageListReq) []MetricItem {
	if req.Category == "" && req.Status == "" && req.Keyword == "" {
		return items
	}
	kw := strings.ToLower(req.Keyword)
	out := make([]MetricItem, 0, len(items))
	for _, m := range items {
		if req.Category != "" && m.Category != req.Category {
			continue
		}
		if req.Status != "" && m.Status != req.Status {
			continue
		}
		if kw != "" {
			hay := strings.ToLower(strings.Join([]string{m.Name, m.Code, m.DataSource, m.Source, m.OwnerRole}, " "))
			if !strings.Contains(hay, kw) {
				continue
			}
		}
		out = append(out, m)
	}
	return out
}

// computeSummary 4 项摘要一次循环聚合（避免 query-per-metric fan-out，database.md）。
func computeSummary(items []MetricItem) ManageSummary {
	s := ManageSummary{
		TotalItems:    int64(len(items)),
		CategoryCount: make(map[string]int64, len(ValidCategories)),
	}
	for _, m := range items {
		s.CategoryCount[m.Category]++
		if m.Value != "—" {
			s.WithSnapshot++
		}
		if m.Status == "warn" || m.Status == "danger" {
			s.Abnormal++
		}
	}
	return s
}

// buildItems 根据 snapshot 拼装 12 条指标定义；规范执行类指标因门禁数据未同步
// 统一返回 "—" + status="unknown"，体现 PRD §14/§15 "无数据显示 — 而非 0%"。
//
// 12 条指标按 5 分类分组：需求治理 2 / 交付效率 3 / 研发质量 3 / 规范执行 2 / 效能管理 2。
func (s *Service) buildItems(snap snapshot, lastTime string) []MetricItem {
	return []MetricItem{
		// ===== 需求治理 =====
		{
			Code: "story.total", Name: "研发需求总量", Category: CategoryDemand,
			Unit: "个", DataSource: "禅道 zt_story", Period: "实时",
			Value:  MetricTextForCount(snap.Stories, false),
			Target: "—", GoodDirection: "up",
			WarningThreshold: "—", DangerThreshold: "—",
			Status:       "normal",
			OwnerRole:    "PO",
			Description:  "研发需求（Story）的累计总数，反映长期需求池规模",
			Formula:      "COUNT(zt_story WHERE deleted='0')",
			Source:       "禅道 zt_story",
			LastCalcTime: lastTime,
		},
		{
			Code: "story.active", Name: "进行中研发需求", Category: CategoryDemand,
			Unit: "个", DataSource: "禅道 zt_story", Period: "实时",
			Value:  MetricTextForCount(snap.StoriesActive, false),
			Target: "≤30", GoodDirection: "down",
			WarningThreshold: "30", DangerThreshold: "60",
			Status:       MetricStatusByCount(snap.StoriesActive, 30, 60, "down"),
			OwnerRole:    "PO",
			Description:  "未关闭/未发布的研发需求数，反映当前需求在研压力",
			Formula:      "COUNT(zt_story WHERE status NOT IN ('closed','released'))",
			Source:       "禅道 zt_story",
			LastCalcTime: lastTime,
		},

		// ===== 交付效率 =====
		{
			Code: "story.doneRate", Name: "研发需求完成率", Category: CategoryDelivery,
			Unit: "%", DataSource: "禅道 zt_story", Period: "实时",
			Value:  MetricTextForRate(snap.StoriesDone, snap.Stories),
			Target: "≥90%", GoodDirection: "up",
			WarningThreshold: "70%", DangerThreshold: "90%",
			Status:       MetricStatusByRate(snap.StoriesDone, snap.Stories, "up"),
			OwnerRole:    "PO",
			Description:  "已关闭/已发布研发需求占总需求的比例，反映整体交付完成度",
			Formula:      "(closed+released) / total",
			Source:       "禅道 zt_story",
			LastCalcTime: lastTime,
		},
		{
			Code: "story.closedOnTimeRate", Name: "按时关闭率", Category: CategoryDelivery,
			Unit: "%", DataSource: "禅道 zt_story", Period: "近 30 天",
			Value:  MetricTextForRate(snap.StoriesClosedOnTime, snap.StoriesDone),
			Target: "≥80%", GoodDirection: "up",
			WarningThreshold: "60%", DangerThreshold: "80%",
			Status:       MetricStatusByRate(snap.StoriesClosedOnTime, snap.StoriesDone, "up"),
			OwnerRole:    "PO",
			Description:  "已关闭研发需求中，实际关闭日期未晚于预计关闭日期的比例",
			Formula:      "SUM(closedDate<=estimatedLaunch) / SUM(closed+released)",
			Source:       "禅道 zt_story",
			LastCalcTime: lastTime,
		},
		{
			Code: "task.overdue", Name: "逾期任务数", Category: CategoryDelivery,
			Unit: "个", DataSource: "禅道 zt_task", Period: "实时",
			Value:  MetricTextForCount(snap.TasksOverdue, false),
			Target: "≤5", GoodDirection: "down",
			WarningThreshold: "5", DangerThreshold: "10",
			Status:       MetricStatusByCount(snap.TasksOverdue, 5, 10, "down"),
			OwnerRole:    "PO",
			Description:  "截止日期早于今天且未关闭/取消的任务数",
			Formula:      "COUNT(zt_task WHERE status NOT IN ('closed','cancel') AND deadline<today)",
			Source:       "禅道 zt_task",
			LastCalcTime: lastTime,
		},

		// ===== 研发质量 =====
		{
			Code: "bug.open", Name: "未关闭缺陷", Category: CategoryQuality,
			Unit: "个", DataSource: "禅道 zt_bug", Period: "实时",
			Value:  MetricTextForCount(snap.Bugs, false),
			Target: "≤5", GoodDirection: "down",
			WarningThreshold: "5", DangerThreshold: "10",
			Status:       MetricStatusByCount(snap.Bugs, 5, 10, "down"),
			OwnerRole:    "测试",
			Description:  "未关闭/未取消的缺陷总数，反映当前缺陷压力",
			Formula:      "COUNT(zt_bug WHERE status NOT IN ('closed','cancelled'))",
			Source:       "禅道 zt_bug",
			LastCalcTime: lastTime,
		},
		{
			Code: "bug.p1p2", Name: "致命/严重缺陷数", Category: CategoryQuality,
			Unit: "个", DataSource: "禅道 zt_bug", Period: "实时",
			Value:  MetricTextForCount(snap.BugsP1P2, false),
			Target: "≤2", GoodDirection: "down",
			WarningThreshold: "2", DangerThreshold: "5",
			Status:       MetricStatusByCount(snap.BugsP1P2, 2, 5, "down"),
			OwnerRole:    "测试",
			Description:  "严重程度为 1 级（致命）或 2 级（严重）的未关闭缺陷数",
			Formula:      "COUNT(zt_bug WHERE severity IN ('1','2') AND status NOT IN ('closed','cancelled'))",
			Source:       "禅道 zt_bug",
			LastCalcTime: lastTime,
		},
		{
			Code: "bug.resolutionRate", Name: "缺陷解决率", Category: CategoryQuality,
			Unit: "%", DataSource: "禅道 zt_bug", Period: "近 30 天",
			Value:  MetricTextForRate(snap.BugsResolved, snap.BugsTotal),
			Target: "≥85%", GoodDirection: "up",
			WarningThreshold: "70%", DangerThreshold: "85%",
			Status:       MetricStatusByRate(snap.BugsResolved, snap.BugsTotal, "up"),
			OwnerRole:    "测试",
			Description:  "已解决/已关闭缺陷占总缺陷数的比例",
			Formula:      "(resolved+closed) / total",
			Source:       "禅道 zt_bug",
			LastCalcTime: lastTime,
		},

		// ===== 规范执行（门禁数据未同步，统一 "—" 占位 + status="unknown"，PRD §14/§15） =====
		{
			Code: "norm.completeness", Name: "业务需求富文本完整性", Category: CategoryCompliance,
			Unit: "—", DataSource: "门禁数据待同步", Period: "日",
			Value:  "—",
			Target: "≥95%", GoodDirection: "up",
			WarningThreshold: "80%", DangerThreshold: "95%",
			Status:       "unknown",
			OwnerRole:    "PMO",
			Description:  "业务需求富文本正文（非空描述+字段完整）的覆盖率；门禁数据未同步，暂以 — 占位",
			Formula:      "COUNT(demand WHERE description!='') / COUNT(demand)",
			Source:       "门禁（待同步）",
			LastCalcTime: lastTime,
		},
		{
			Code: "norm.owner", Name: "责任人必填覆盖率", Category: CategoryCompliance,
			Unit: "—", DataSource: "门禁数据待同步", Period: "日",
			Value:  "—",
			Target: "≥98%", GoodDirection: "up",
			WarningThreshold: "90%", DangerThreshold: "98%",
			Status:       "unknown",
			OwnerRole:    "PMO",
			Description:  "需求/任务/Bug 中责任人字段非空的比例；门禁数据未同步，暂以 — 占位",
			Formula:      "COUNT(obj WHERE assignedTo!='') / COUNT(obj)",
			Source:       "门禁（待同步）",
			LastCalcTime: lastTime,
		},

		// ===== 效能管理 =====
		{
			Code: "task.total", Name: "任务总量", Category: CategoryPerformance,
			Unit: "个", DataSource: "禅道 zt_task", Period: "实时",
			Value:  MetricTextForCount(snap.Tasks, false),
			Target: "—", GoodDirection: "up",
			WarningThreshold: "—", DangerThreshold: "—",
			Status:       "normal",
			OwnerRole:    "SM",
			Description:  "任务（含已完成）的累计总数，反映长期任务池规模",
			Formula:      "COUNT(zt_task WHERE deleted='0')",
			Source:       "禅道 zt_task",
			LastCalcTime: lastTime,
		},
		{
			Code: "task.open", Name: "未完成任务", Category: CategoryPerformance,
			Unit: "个", DataSource: "禅道 zt_task", Period: "实时",
			Value:  MetricTextForCount(snap.TasksOpen, false),
			Target: "≤20", GoodDirection: "down",
			WarningThreshold: "20", DangerThreshold: "40",
			Status:       MetricStatusByCount(snap.TasksOpen, 20, 40, "down"),
			OwnerRole:    "SM",
			Description:  "未关闭/未取消的任务数，反映当前任务压力",
			Formula:      "COUNT(zt_task WHERE status NOT IN ('closed','cancel'))",
			Source:       "禅道 zt_task",
			LastCalcTime: lastTime,
		},
	}
}
