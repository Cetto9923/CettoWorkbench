// =============================================================================
// 文件: internal/module/metrics/service_test.go
// 模块: 指标管理 (metrics)
// 类型: test
// 职责: 状态判定 / 文本格式化 / 分类映射 helper 的单元测试；
//       覆盖 PRD §14/§15 "无数据显示 — 而非 0%" 的边界。
// 依赖: 无
// =============================================================================

package metrics

import (
	"testing"
)

func TestMetricStatusByRate_EmptyDenominatorReturnsUnknown(t *testing.T) {
	if got := MetricStatusByRate(0, 0, "up"); got != "unknown" {
		t.Errorf("MetricStatusByRate(0,0,up) = %q, want unknown", got)
	}
	if got := MetricStatusByRate(5, 0, "down"); got != "unknown" {
		t.Errorf("MetricStatusByRate(5,0,down) = %q, want unknown", got)
	}
}

func TestMetricStatusByRate_UpDirection(t *testing.T) {
	cases := []struct {
		value, total int64
		want         string
	}{
		{95, 100, "normal"}, // 0.95 >= 0.9
		{90, 100, "normal"}, // 0.90 >= 0.9
		{80, 100, "warn"},   // 0.80 >= 0.7
		{70, 100, "warn"},   // 0.70 >= 0.7
		{50, 100, "danger"}, // 0.50 < 0.7
	}
	for _, tc := range cases {
		if got := MetricStatusByRate(tc.value, tc.total, "up"); got != tc.want {
			t.Errorf("MetricStatusByRate(%d,%d,up) = %q, want %q", tc.value, tc.total, got, tc.want)
		}
	}
}

func TestMetricStatusByRate_DownDirection(t *testing.T) {
	cases := []struct {
		value, total int64
		want         string
	}{
		{5, 100, "normal"},  // 0.05 <= 0.1
		{10, 100, "normal"}, // 0.10 <= 0.1
		{20, 100, "warn"},   // 0.20 <= 0.3
		{30, 100, "warn"},   // 0.30 <= 0.3
		{50, 100, "danger"}, // 0.50 > 0.3
	}
	for _, tc := range cases {
		if got := MetricStatusByRate(tc.value, tc.total, "down"); got != tc.want {
			t.Errorf("MetricStatusByRate(%d,%d,down) = %q, want %q", tc.value, tc.total, got, tc.want)
		}
	}
}

func TestMetricStatusByCount_UpDirection(t *testing.T) {
	// up: 越大越好（达标 = danger 表示「最少」也要达到的数）
	cases := []struct {
		value, warning, danger int64
		want                   string
	}{
		{100, 30, 50, "normal"}, // >= danger
		{50, 30, 50, "normal"},  // >= danger
		{40, 30, 50, "warn"},    // >= warning, < danger
		{20, 30, 50, "danger"},  // < warning
	}
	for _, tc := range cases {
		if got := MetricStatusByCount(tc.value, tc.warning, tc.danger, "up"); got != tc.want {
			t.Errorf("MetricStatusByCount(%d,w=%d,d=%d,up) = %q, want %q",
				tc.value, tc.warning, tc.danger, got, tc.want)
		}
	}
}

func TestMetricStatusByCount_DownDirection(t *testing.T) {
	// down: 越小越好（<=warning normal / <=danger warn / >danger danger）
	cases := []struct {
		value, warning, danger int64
		want                   string
	}{
		{3, 5, 10, "normal"},  // <= warning
		{5, 5, 10, "normal"},  // == warning
		{7, 5, 10, "warn"},    // <= danger, > warning
		{10, 5, 10, "warn"},   // == danger
		{15, 5, 10, "danger"}, // > danger
	}
	for _, tc := range cases {
		if got := MetricStatusByCount(tc.value, tc.warning, tc.danger, "down"); got != tc.want {
			t.Errorf("MetricStatusByCount(%d,w=%d,d=%d,down) = %q, want %q",
				tc.value, tc.warning, tc.danger, got, tc.want)
		}
	}
}

func TestMetricTextForRate_DashOnZero(t *testing.T) {
	if got := MetricTextForRate(0, 0); got != "—" {
		t.Errorf("MetricTextForRate(0,0) = %q, want —", got)
	}
	if got := MetricTextForRate(85, 100); got != "85%" {
		t.Errorf("MetricTextForRate(85,100) = %q, want 85%%", got)
	}
}

func TestMetricTextForCount_AllowDash(t *testing.T) {
	if got := MetricTextForCount(0, true); got != "—" {
		t.Errorf("MetricTextForCount(0,allowDash) = %q, want —", got)
	}
	if got := MetricTextForCount(0, false); got != "0" {
		t.Errorf("MetricTextForCount(0,!allowDash) = %q, want 0", got)
	}
	if got := MetricTextForCount(42, true); got != "42" {
		t.Errorf("MetricTextForCount(42,allowDash) = %q, want 42", got)
	}
}

func TestMetricStatusLabel(t *testing.T) {
	cases := map[string]string{
		"normal":  "正常",
		"warn":    "关注",
		"danger":  "风险",
		"unknown": "暂无数据",
		"":        "—",
		"garbage": "—",
	}
	for in, want := range cases {
		if got := MetricStatusLabel(in); got != want {
			t.Errorf("MetricStatusLabel(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestCategoryColorClass(t *testing.T) {
	cases := map[string]string{
		CategoryDemand:      "cat-demand",
		CategoryDelivery:    "cat-delivery",
		CategoryQuality:     "cat-quality",
		CategoryCompliance:  "cat-compliance",
		CategoryPerformance: "cat-performance",
		"":                  "cat-default",
		"未知":                "cat-default",
	}
	for in, want := range cases {
		if got := CategoryColorClass(in); got != want {
			t.Errorf("CategoryColorClass(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestIsValidCategory(t *testing.T) {
	for _, c := range ValidCategories {
		if !IsValidCategory(c) {
			t.Errorf("IsValidCategory(%q) = false, want true", c)
		}
	}
	if IsValidCategory("未知") || IsValidCategory("") {
		t.Error("IsValidCategory should reject non-canonical values")
	}
}

func TestManageListReqValidate(t *testing.T) {
	r := ManageListReq{Status: "weird"}
	if err := r.Validate(); err == nil {
		t.Error("expected invalid status to fail")
	}
	for _, ok := range []string{"", "normal", "warn", "danger", "unknown"} {
		r := ManageListReq{Status: ok}
		if err := r.Validate(); err != nil {
			t.Errorf("status=%q expected pass, got %v", ok, err)
		}
	}
}

func TestManageListReqNormalizeDefaults(t *testing.T) {
	r := ManageListReq{}
	r.Normalize()
	if r.Page != 1 {
		t.Errorf("expected default page=1, got %d", r.Page)
	}
	if r.PageSize != 15 {
		t.Errorf("expected default pageSize=15, got %d", r.PageSize)
	}
	r = ManageListReq{Page: 0, PageSize: 999}
	r.Normalize()
	if r.PageSize != 15 {
		t.Errorf("expected over-100 pageSize to clamp to 15, got %d", r.PageSize)
	}
}

func TestComputeSummary_BoundedDataset(t *testing.T) {
	items := []MetricItem{
		{Category: CategoryDemand, Value: "10", Status: "normal"},
		{Category: CategoryDemand, Value: "20", Status: "warn"},
		{Category: CategoryDelivery, Value: "—", Status: "unknown"},
		{Category: CategoryQuality, Value: "5", Status: "danger"},
	}
	s := computeSummary(items)
	if s.TotalItems != 4 {
		t.Errorf("TotalItems=%d want 4", s.TotalItems)
	}
	if s.CategoryCount[CategoryDemand] != 2 || s.CategoryCount[CategoryDelivery] != 1 || s.CategoryCount[CategoryQuality] != 1 {
		t.Errorf("CategoryCount mismatch: %+v", s.CategoryCount)
	}
	if s.WithSnapshot != 3 {
		t.Errorf("WithSnapshot=%d want 3 (— excluded)", s.WithSnapshot)
	}
	if s.Abnormal != 2 {
		t.Errorf("Abnormal=%d want 2 (warn+danger)", s.Abnormal)
	}
}

func TestFilterItems(t *testing.T) {
	items := []MetricItem{
		{Code: "a", Name: "Alpha", Category: CategoryDemand, Status: "normal", DataSource: "禅道"},
		{Code: "b", Name: "Beta", Category: CategoryQuality, Status: "danger", DataSource: "DevOps"},
		{Code: "c", Name: "Gamma", Category: CategoryDemand, Status: "warn", DataSource: "门禁"},
	}
	req := ManageListReq{Category: CategoryDemand, Status: "", Keyword: ""}
	out := filterItems(items, req)
	if len(out) != 2 || out[0].Code != "a" || out[1].Code != "c" {
		t.Errorf("category filter expected [a,c], got %v", codes(out))
	}
	req = ManageListReq{Status: "danger"}
	out = filterItems(items, req)
	if len(out) != 1 || out[0].Code != "b" {
		t.Errorf("status filter expected [b], got %v", codes(out))
	}
	req = ManageListReq{Keyword: "禅道"}
	out = filterItems(items, req)
	if len(out) != 1 || out[0].Code != "a" {
		t.Errorf("keyword filter expected [a], got %v", codes(out))
	}
	req = ManageListReq{}
	out = filterItems(items, req)
	if len(out) != 3 {
		t.Errorf("empty filter expected all 3, got %v", codes(out))
	}
}

func codes(items []MetricItem) []string {
	out := make([]string, 0, len(items))
	for _, m := range items {
		out = append(out, m.Code)
	}
	return out
}

func TestComputeCategoryScore_EmptyTotalIsUnknown(t *testing.T) {
	score, sev := computeCategoryScore(&RadarCategoryScore{Total: 0})
	if score != 0 || sev != "unknown" {
		t.Errorf("empty total: score=%d sev=%q want 0/unknown", score, sev)
	}
}

func TestComputeCategoryScore_SeverityBands(t *testing.T) {
	cases := []struct {
		name                 string
		normal, warn, danger int
		wantScore            int
		wantSev              string
	}{
		{"all_normal", 10, 0, 0, 100, "normal"},
		{"all_warn", 0, 10, 0, 50, "danger"},
		{"all_danger", 0, 0, 10, 0, "danger"},
		{"mixed_high", 8, 2, 0, 90, "normal"},   // (8+1)/10*100=90
		{"mixed_mid", 7, 2, 1, 80, "normal"},    // (7+1)/10*100=80 → boundary
		{"mixed_low_warn", 6, 3, 1, 75, "warn"}, // (6+1.5)/10*100=75
		{"mixed_danger", 4, 2, 4, 50, "danger"}, // (4+1)/10*100=50
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			b := &RadarCategoryScore{
				NormalCount: tc.normal, WarnCount: tc.warn, DangerCount: tc.danger,
				Total: tc.normal + tc.warn + tc.danger,
			}
			gotScore, gotSev := computeCategoryScore(b)
			if gotScore != tc.wantScore || gotSev != tc.wantSev {
				t.Errorf("got (score=%d, sev=%q), want (score=%d, sev=%q)",
					gotScore, gotSev, tc.wantScore, tc.wantSev)
			}
		})
	}
}

func TestTopRiskOf_PicksFirstByCodeAscending(t *testing.T) {
	b := &RadarCategoryScore{Items: []MetricItem{
		{Code: "z.danger", Name: "Z", Status: "danger"},
		{Code: "a.danger", Name: "A", Status: "danger"},
		{Code: "m.normal", Name: "M", Status: "normal"},
	}}
	code, name := topRiskOf(b)
	if code != "a.danger" || name != "A" {
		t.Errorf("topRiskOf = (%q, %q), want (a.danger, A)", code, name)
	}
}

func TestTopRiskOf_NoDangerIsEmpty(t *testing.T) {
	b := &RadarCategoryScore{Items: []MetricItem{
		{Code: "x.normal", Status: "normal"},
		{Code: "y.warn", Status: "warn"},
	}}
	code, name := topRiskOf(b)
	if code != "" || name != "" {
		t.Errorf("expected empty, got (%q, %q)", code, name)
	}
}

func TestRadarScore_Boundary60IsWarn(t *testing.T) {
	// 5 normal + 5 warn / 10 total → (5+2.5)/10*100 = 75 → warn
	b := &RadarCategoryScore{NormalCount: 5, WarnCount: 5, Total: 10}
	score, sev := computeCategoryScore(b)
	if score != 75 || sev != "warn" {
		t.Errorf("got (%d, %q), want (75, warn)", score, sev)
	}
}

func TestRadarScore_Boundary80IsNormal(t *testing.T) {
	// 8 normal + 0 warn / 10 → 80 → normal (>=80)
	b := &RadarCategoryScore{NormalCount: 8, WarnCount: 1, DangerCount: 1, Total: 10}
	score, sev := computeCategoryScore(b)
	if score != 85 || sev != "normal" {
		t.Errorf("got (%d, %q), want (85, normal)", score, sev)
	}
}

func TestCategoryOrder_ReturnsValidIndex(t *testing.T) {
	idx := categoryOrder(CategoryDemand)
	if idx != 0 {
		t.Errorf("CategoryDemand order = %d, want 0", idx)
	}
	idx = categoryOrder("未知")
	if idx != len(ValidCategories) {
		t.Errorf("unknown category order = %d, want %d", idx, len(ValidCategories))
	}
}
