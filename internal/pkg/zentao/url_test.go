package zentao

import (
	"testing"
	"workbench/internal/config"
)

func TestURL(t *testing.T) {
	SetConfig(config.ZentaoConfig{URL: "https://changshu.wrk.oop.cc/"})
	got := DemandViewURL(37404)
	want := "https://changshu.wrk.oop.cc/index.php?m=demand&f=view&demandID=37404"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
	gotWithBase := URLWithBase("https://example.zentao.net", "story", "view", "storyID=1")
	if gotWithBase != "https://example.zentao.net/index.php?m=story&f=view&storyID=1" {
		t.Fatalf("URLWithBase got %q", gotWithBase)
	}
	SetConfig(config.ZentaoConfig{})
	if DemandViewURL(1) != "" {
		t.Fatal("expected empty when no base url")
	}
}
