// =============================================================================
// 文件: internal/module/po/repoboardmetrics_test.go
// 模块: PO 工作台
// 类型: test
// 职责: 证明看板 4 项指标的名称、达标文案与阈值全部来自 metrics 目录，
//       看板不再自带一份魔法数（对齐一致性检视 A13）。
// =============================================================================

package po

import (
	"testing"

	"workbench/internal/module/metrics"
)

func TestBoardGroupMetricsReadsCatalog(t *testing.T) {
	cases := []struct {
		key         string
		name        string
		target      string
		targetValue float64
		dangerValue float64
	}{
		{"delivery", "交付周期", "≤30天", 30, 50},
		{"implement", "实施周期", "≤20天", 20, 35},
		{"overIteration", "超两迭代周期占比", "≤10%", 10, 20},
		{"unscheduled", "超2周未排期单数", "≤3个", 3, 8},
	}

	ms := boardGroupMetrics()
	if len(ms) != len(cases) {
		t.Fatalf("board metrics count = %d, want %d", len(ms), len(cases))
	}
	for i, c := range cases {
		m := ms[i]
		if m.Key != c.key {
			t.Fatalf("metric[%d].Key = %q, want %q", i, m.Key, c.key)
		}
		if m.Name != c.name || m.Target != c.target {
			t.Errorf("%s: name/target = %q/%q, want %q/%q", c.key, m.Name, m.Target, c.name, c.target)
		}
		checkBoardSpecFromCatalog(t, i, c.key, c.targetValue, c.dangerValue)
	}

	// 目录里的天数常量必须落在报告指出的口径上（>42 天算超迭代；超 14 天未澄清算未排期）。
	if metrics.OverIterationDays != 42 {
		t.Errorf("OverIterationDays = %d, want 42", metrics.OverIterationDays)
	}
	if metrics.UnscheduledClarifyDays != 14 {
		t.Errorf("UnscheduledClarifyDays = %d, want 14", metrics.UnscheduledClarifyDays)
	}
}

// checkBoardSpecFromCatalog 断言看板阈值与目录同源，而非另抄一份魔法数。
func checkBoardSpecFromCatalog(t *testing.T, idx int, key string, targetValue, dangerValue float64) {
	t.Helper()
	s := boardSpec(key)
	if s.TargetValue != targetValue || s.DangerValue != dangerValue {
		t.Errorf("%s: boardSpec = goal %.0f / warn %.0f, want %.0f / %.0f",
			key, s.TargetValue, s.DangerValue, targetValue, dangerValue)
	}
	code := boardMetricCodes[idx].code
	ds, ok := metrics.SpecOf(code)
	if !ok || ds.TargetValue != s.TargetValue || ds.DangerValue != s.DangerValue {
		t.Errorf("%s(%s): 看板阈值与目录条目不一致", key, code)
	}
}

func TestBoardMetricKeysAreDistinct(t *testing.T) {
	seen := map[string]bool{}
	for _, d := range boardMetricCodes {
		if seen[d.key] {
			t.Fatalf("duplicate board key %q", d.key)
		}
		seen[d.key] = true
		if _, ok := metrics.SpecOf(d.code); !ok {
			t.Fatalf("board key %q 指向目录中不存在的 code %q", d.key, d.code)
		}
	}
}

func TestBoardMetricThresholdFolding(t *testing.T) {
	// down 指标：<=达标线 good，达标线~危险线 warn，>危险线 risk。
	ms := boardGroupMetrics()
	s := boardSpec("delivery")
	cases := []struct {
		v    float64
		want string
	}{
		{s.TargetValue, "good"},
		{s.TargetValue + 1, "warn"},
		{s.DangerValue, "warn"},
		{s.DangerValue + 1, "risk"},
	}
	for _, c := range cases {
		setMetricValue(ms, "delivery", "", c.v, s.DangerValue, s.TargetValue)
		if got := metricOf(ms, "delivery").State; got != c.want {
			t.Errorf("delivery %.0f => %s, want %s", c.v, got, c.want)
		}
	}
	u := boardSpec("unscheduled")
	setMetricValue(ms, "unscheduled", "9个", 9, u.DangerValue, u.TargetValue)
	if got := metricOf(ms, "unscheduled").State; got != "risk" {
		t.Errorf("unscheduled 9 => %s, want risk", got)
	}
}
