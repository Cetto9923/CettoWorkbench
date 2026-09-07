// =============================================================================
// 文件: internal/module/metrics/radar_service.go
// 模块: 指标管理 (metrics) - /metrics/radar 派生逻辑
// 类型: service (radar 拆分)
// 职责: 5 分类 KPI 卡得分 / 严重度 / 主要风险指标派生；
//       复用 service.go 的 buildItems 与 repo.RadarSummary（= Snapshot），
//       不引入 query-per-metric fan-out（database.md）。
// 依赖: internal/module/metrics/service.go
// =============================================================================

package metrics

import (
	"context"
	"math"
	"sort"
	"strconv"
	"time"
)

// Radar 返回指标雷达响应：5 分类 KPI 卡片 + 异常指标分组列表。
//
// 算法（与 PRD §14/§15 一致）：
//
//	score = (NormalCount*1.0 + WarnCount*0.5) / TotalCount * 100 ；
//	total==0 → score=0 + Severity="unknown"（不显示假风险）。
//	score≥80 → "normal"；60≤score<80 → "warn"；score<60 → "danger"。
//
// TopRiskCode / TopRiskName：取该分类 danger 指标中按 code 升序的首条；
// 无 danger 时为空字符串。
//
// 数据来源：复用 RadarSummary（= Snapshot）单 SQL + buildItems（manage 共用），
// 不引入第二次 query-per-metric fan-out。
func (s *Service) Radar(ctx context.Context) (RadarResp, error) {
	snap, err := s.repo.RadarSummary(ctx)
	if err != nil {
		return RadarResp{}, err
	}
	now := time.Now().Format("2006-01-02 15:04")
	items := s.buildItems(snap, now)

	// 1) 按 5 分类聚合状态计数
	byCategory := make(map[string]*RadarCategoryScore, len(ValidCategories))
	for _, c := range ValidCategories {
		byCategory[c] = &RadarCategoryScore{Category: c, Items: []MetricItem{}}
	}
	for _, m := range items {
		bucket, ok := byCategory[m.Category]
		if !ok {
			// 防御：buildItems 不应输出未在白名单的分类，但万一出现就跳过
			continue
		}
		bucket.Items = append(bucket.Items, m)
		bucket.Total++
		switch m.Status {
		case "normal":
			bucket.NormalCount++
		case "warn":
			bucket.WarnCount++
		case "danger":
			bucket.DangerCount++
		case "unknown":
			bucket.UnknownCount++
		}
	}

	// 2) 拼装 KPI 卡 + 计算 score / severity / topRisk
	categories := make([]RadarCategoryItem, 0, len(ValidCategories))
	for _, c := range ValidCategories {
		bucket := byCategory[c]
		score, severity := computeCategoryScore(bucket)
		topCode, topName := topRiskOf(bucket)
		rendered := "—"
		if bucket.Total > 0 {
			rendered = strconv.Itoa(score)
		}
		categories = append(categories, RadarCategoryItem{
			Category:     c,
			Score:        rendered,
			ScoreValue:   score,
			Severity:     severity,
			TotalCount:   bucket.Total,
			NormalCount:  bucket.NormalCount,
			WarnCount:    bucket.WarnCount,
			DangerCount:  bucket.DangerCount,
			UnknownCount: bucket.UnknownCount,
			TopRiskCode:  topCode,
			TopRiskName:  topName,
		})
	}

	// 3) 异常指标：warn / danger，按 Category → Code 排序
	abnormal := make([]MetricItem, 0, len(items))
	for _, m := range items {
		if m.Status == "warn" || m.Status == "danger" {
			abnormal = append(abnormal, m)
		}
	}
	sort.SliceStable(abnormal, func(i, j int) bool {
		if abnormal[i].Category != abnormal[j].Category {
			return categoryOrder(abnormal[i].Category) < categoryOrder(abnormal[j].Category)
		}
		return abnormal[i].Code < abnormal[j].Code
	})

	return RadarResp{
		Categories:    categories,
		AbnormalItems: abnormal,
		GeneratedAt:   now,
	}, nil
}

// computeCategoryScore 按 PRD §14/§15 计算分类得分与严重度。
//
//	total==0 → score=0, severity="unknown"（不展示假风险）
//	否则 score = (N + W*0.5) / total * 100（四舍五入到整数）
//	score≥80 → "normal"；60≤score<80 → "warn"；score<60 → "danger"
func computeCategoryScore(b *RadarCategoryScore) (int, string) {
	if b == nil || b.Total == 0 {
		return 0, "unknown"
	}
	raw := (float64(b.NormalCount) + float64(b.WarnCount)*0.5) / float64(b.Total) * 100.0
	score := int(math.Round(raw))
	switch {
	case score >= 80:
		return score, "normal"
	case score >= 60:
		return score, "warn"
	default:
		return score, "danger"
	}
}

// topRiskOf 在 danger 指标中按 code 升序取首条作为「主要风险」；无 danger 时返回空。
func topRiskOf(b *RadarCategoryScore) (string, string) {
	if b == nil || len(b.Items) == 0 {
		return "", ""
	}
	// 复制一份 danger 列表排序，避免修改 b.Items
	dangers := make([]MetricItem, 0, len(b.Items))
	for _, m := range b.Items {
		if m.Status == "danger" {
			dangers = append(dangers, m)
		}
	}
	if len(dangers) == 0 {
		return "", ""
	}
	sort.Slice(dangers, func(i, j int) bool { return dangers[i].Code < dangers[j].Code })
	return dangers[0].Code, dangers[0].Name
}

// categoryOrder 返回分类在 KPI 卡中的固定顺序索引（用于异常列表分组排序）。
func categoryOrder(c string) int {
	for i, v := range ValidCategories {
		if v == c {
			return i
		}
	}
	return len(ValidCategories)
}
