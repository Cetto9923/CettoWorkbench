package build

import (
	"testing"
)

func TestResolveExecutionID(t *testing.T) {
	t.Parallel()
	if got := ResolveExecutionID(10, 20); got != 10 {
		t.Fatalf("prefer execution: got %d", got)
	}
	if got := ResolveExecutionID(0, 20); got != 20 {
		t.Fatalf("fallback project: got %d", got)
	}
	if got := ResolveExecutionID(0, 0); got != 0 {
		t.Fatalf("empty: got %d", got)
	}
}

func TestParseCSVUintIDs(t *testing.T) {
	t.Parallel()
	got := ParseCSVUintIDs(",12,,0,34,12,")
	if len(got) != 2 || got[0] != 12 || got[1] != 34 {
		t.Fatalf("got %#v", got)
	}
	if got := ParseCSVUintIDs(""); len(got) != 0 {
		t.Fatalf("empty: %#v", got)
	}
}

func TestMergeAllStoriesCSV(t *testing.T) {
	t.Parallel()
	got := MergeAllStoriesCSV("1,2", []string{"2,3", "", "4"})
	want := []uint{1, 2, 3, 4}
	if len(got) != len(want) {
		t.Fatalf("got %#v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %#v want %#v", got, want)
		}
	}
}

func TestShouldDefaultCheckStage(t *testing.T) {
	t.Parallel()
	for _, stage := range []string{"developed", "tested", "closed"} {
		if !ShouldDefaultCheckStage(stage) {
			t.Fatalf("%s should check", stage)
		}
	}
	if ShouldDefaultCheckStage("projected") {
		t.Fatal("projected should not check")
	}
}

func TestStoryStatusLabel(t *testing.T) {
	t.Parallel()
	if got := StoryStatusLabel("active"); got != "激活" {
		t.Fatalf("got %q", got)
	}
	if got := StoryStatusLabel("unknown"); got != "unknown" {
		t.Fatalf("fallback got %q", got)
	}
}

func TestStoryStageLabel(t *testing.T) {
	t.Parallel()
	if got := StoryStageLabel("projected"); got != "研发立项" {
		t.Fatalf("got %q", got)
	}
}

func TestFormatEstimate(t *testing.T) {
	t.Parallel()
	if got := FormatEstimate(0); got != "0h" {
		t.Fatalf("got %q", got)
	}
	if got := FormatEstimate(1.5); got != "1.5h" {
		t.Fatalf("got %q", got)
	}
}

func TestPriorityClass(t *testing.T) {
	t.Parallel()
	if got := PriorityClass(0); got != "P4" {
		t.Fatalf("pri0: %q", got)
	}
	if got := PriorityClass(3); got != "P3" {
		t.Fatalf("pri3: %q", got)
	}
}
