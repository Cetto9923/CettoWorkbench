// =============================================================================
// 文件: internal/module/metrics/service.go
// 模块: 指标管理 (metrics)
// 类型: service
// 职责: 指标列表聚合：从 metricCatalog（唯一 SSOT）+ repo.Snapshot 组装 MetricItem，
//       filter → page 切片、summary 聚合。指标元数据定义与状态判定逻辑见 catalog.go。
// 依赖: internal/module/metrics/repo.go
// =============================================================================

package metrics

import (
	"context"
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

// List 返回 ManageListResp：单次 Snapshot → 从 catalog 组装 items → 过滤 → 分页 → summary。
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

// filterItems 在有界 items 上按 category / status / keyword 过滤。
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

// computeSummary 摘要一次循环聚合（避免 query-per-metric fan-out，database.md）。
// WithSnapshot = value 不为 "—" 的指标条数（有真实数据）；unavailable 不计入。
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

// buildItems 从 metricCatalog（SSOT）组装 MetricItem 列表。
// 禅道聚合指标从 snapshot 取运行值；外部未接入指标 value="—" + status="unknown"（unavailable）。
func (s *Service) buildItems(snap snapshot, lastTime string) []MetricItem {
	items := make([]MetricItem, 0, len(metricCatalog))
	for _, d := range metricCatalog {
		if !d.Enabled {
			continue
		}
		items = append(items, buildItem(d, snap, lastTime))
	}
	return items
}

// buildItem 把一条目录定义 + 运行快照组装成前端展示契约 MetricItem。
func buildItem(d metricDef, snap snapshot, lastTime string) MetricItem {
	value, hasData := valueFor(d, snap)
	status := evaluateStatus(d, value, hasData)
	return MetricItem{
		Code:             d.Code,
		Name:             d.Name,
		Category:         d.Category,
		Unit:             d.Unit,
		DataSource:       d.SourceLabel,
		Period:           d.Period,
		Value:            formatValue(d.Unit, value, hasData),
		Target:           d.Target,
		WarningThreshold: "—", // 两阈值模型：warning 是 target/danger 之间的派生中间态
		DangerThreshold:  d.Danger,
		GoodDirection:    d.Direction,
		Status:           status,
		Enabled:          d.Enabled,
		OwnerRole:        d.OwnerRole,
		Description:      d.Description,
		Formula:          d.Formula,
		LastCalcTime:     lastTime,
		Source:           d.SourceLabel,
	}
}
