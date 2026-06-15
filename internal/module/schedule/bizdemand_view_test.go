package schedule

import (
	"testing"
	"workbench/internal/config"
	"workbench/internal/pkg/zentao"
)

func TestDemandDetailURL(t *testing.T) {
	base := "https://changshu.wrk.oop.cc"
	zentao.SetConfig(config.ZentaoConfig{URL: base + "/"})
	got := zentao.DemandViewURLWithBase(base, 37404)
	want := "https://changshu.wrk.oop.cc/index.php?m=demand&f=view&demandID=37404"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
	items := toBizRequirementsView([]BizDemandItem{{ID: 37404, Name: "x"}}, base)
	if len(items) != 1 || string(items[0].DetailURL) != want {
		t.Fatalf("view DetailURL=%q", items[0].DetailURL)
	}
}
