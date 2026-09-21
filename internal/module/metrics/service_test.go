// =============================================================================
// 文件: internal/module/metrics/service_test.go
// 模块: 指标管理 (metrics)
// 类型: test
// 职责: 指标目录（catalog）、两阈值状态判定、组装逻辑的单元测试；
//       覆盖 PRD §14/§15 "无数据显示 — 而非 0%" 与 unavailable 不参与评分的边界。
// 依赖: 无
// =============================================================================

package metrics

import (
	"testing"
)

func TestEvaluateStatus_UpDirection(t *testing.T) {
	// story.doneRate 语义：target=90（达标线）、danger=60（危险线）、direction=up。
	d := metricDef{Direction: "up", TargetValue: 90, DangerValue: 60}
	cases := []struct {
		value   float64
		hasData bool
		want    string
	}{
		{95, true, "normal"},  // >= 90
		{90, true, "normal"},  // == target 边界
		{85, true, "warn"},    // [60,90)
		{60, true, "warn"},    // == danger 边界
		{59, true, "danger"},  // < 60
		{0, false, "unknown"}, // unavailable
	}
	for _, tc := range cases {
		if got := evaluateStatus(d, tc.value, tc.hasData); got != tc.want {
			t.Errorf("evaluateStatus(up, %.0f, %v) = %q, want %q", tc.value, tc.hasData, got, tc.want)
		}
	}
}

func TestEvaluateStatus_DownDirection(t *testing.T) {
	d := metricDef{Direction: "down", TargetValue: 30, DangerValue: 60}
	cases := []struct {
		value float64
		want  string
	}{
		{20, "normal"}, // <= 30
		{30, "normal"}, // == target 边界
		{45, "warn"},   // (30,60]
		{60, "warn"},   // == danger 边界
		{80, "danger"}, // > 60
	}
	for _, tc := range cases {
		if got := evaluateStatus(d, tc.value, true); got != tc.want {
			t.Errorf("evaluateStatus(down, %.0f) = %q, want %q", tc.value, got, tc.want)
		}
	}
}

func TestEvaluateStatus_NoTargetIsNormal(t *testing.T) {
	d := metricDef{Direction: "up", TargetValue: -1, DangerValue: -1}
	if got := evaluateStatus(d, 100, true); got != "normal" {
		t.Errorf("no-target metric should be normal, got %q", got)
	}
}

func TestCatalog_UniqueCodesAndSize(t *testing.T) {
	if len(metricCatalog) != 21 {
		t.Fatalf("metricCatalog has %d entries, want 21", len(metricCatalog))
	}
	seen := make(map[string]bool, len(metricCatalog))
	for _, d := range metricCatalog {
		if d.Code == "" {
			t.Error("catalog entry has empty code")
		}
		if seen[d.Code] {
			t.Errorf("duplicate code %q in catalog", d.Code)
		}
		seen[d.Code] = true
		if !IsValidCategory(d.Category) {
			t.Errorf("code %q has invalid category %q", d.Code, d.Category)
		}
	}
}

func TestCatalog_SourceTypeDistribution(t *testing.T) {
	zentao, external := 0, 0
	for _, d := range metricCatalog {
		switch d.SourceType {
		case SourceZentao:
			zentao++
		case SourceExternal:
			external++
		default:
			t.Errorf("code %q has unknown sourceType %q", d.Code, d.SourceType)
		}
	}
	if zentao != 10 || external != 11 {
		t.Errorf("sourceType distribution = zentao %d / external %d, want 10 / 11", zentao, external)
	}
}

func TestCatalog_DoneRateSemantics(t *testing.T) {
	for _, d := range metricCatalog {
		if d.Code == "story.doneRate" {
			if d.Direction != "up" || d.TargetValue != 90 || d.DangerValue != 60 {
				t.Errorf("story.doneRate semantics = dir %q target %.0f danger %.0f, want up/90/60",
					d.Direction, d.TargetValue, d.DangerValue)
			}
			return
		}
	}
	t.Fatal("story.doneRate not found in catalog")
}

func TestValueFor_DoneRate(t *testing.T) {
	snap := snapshot{Stories: 100, StoriesDone: 85}
	v, ok := valueFor(metricDef{Code: "story.doneRate", SourceType: SourceZentao}, snap)
	if !ok || v != 85 {
		t.Errorf("valueFor(doneRate, 85/100) = (%v, %v), want (85, true)", v, ok)
	}
	// 分母为 0 → unavailable
	if _, ok := valueFor(metricDef{Code: "story.doneRate", SourceType: SourceZentao}, snapshot{}); ok {
		t.Error("valueFor(doneRate, 0/0) should be unavailable")
	}
}

func TestValueFor_ExternalIsUnavailable(t *testing.T) {
	if _, ok := valueFor(metricDef{Code: "delivery.cycle", SourceType: SourceExternal}, snapshot{Stories: 1}); ok {
		t.Error("external metric should be unavailable regardless of snapshot")
	}
}

func TestBuildItems_AssemblesCatalog(t *testing.T) {
	s := &Service{}
	items := s.buildItems(snapshot{Stories: 10, StoriesActive: 3, StoriesDone: 9}, "now")
	if len(items) != 21 {
		t.Fatalf("buildItems produced %d items, want 21", len(items))
	}
	byCode := make(map[string]MetricItem, len(items))
	for _, m := range items {
		byCode[m.Code] = m
	}
	// 外部指标必须 unavailable
	for _, code := range []string{"delivery.cycle", "gate.passRate", "sp.deviationRate", "norm.completeness"} {
		m, ok := byCode[code]
		if !ok {
			t.Errorf("code %q missing from items", code)
			continue
		}
		if m.Value != "—" || m.Status != "unknown" {
			t.Errorf("external metric %q = value %q status %q, want —/unknown", code, m.Value, m.Status)
		}
	}
	// 禅道指标有真实值：story.doneRate = 9/10 = 90% → normal
	if m := byCode["story.doneRate"]; m.Value != "90%" || m.Status != "normal" {
		t.Errorf("story.doneRate = value %q status %q, want 90%%/normal", m.Value, m.Status)
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
