package po

import (
	"testing"

	"workbench/internal/module/po/primaryaction"
)

func TestHomeActionPriority(t *testing.T) {
	cases := []struct {
		name string
		item WorkItemDetail
		want int
	}{
		{name: "review", item: WorkItemDetail{CanReview: true}, want: 0},
		{name: "submit", item: WorkItemDetail{PrimaryAction: actionForTest(primaryaction.KeySubmitReview)}, want: 1},
		{name: "view", item: WorkItemDetail{}, want: 2},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := homeActionPriority(tc.item); got != tc.want {
				t.Fatalf("priority = %d, want %d", got, tc.want)
			}
		})
	}
}

func actionForTest(key primaryaction.ActionKey) *primaryaction.PrimaryAction {
	action := primaryaction.Enabled(string(key), "", string(primaryaction.KindDrawer), "")
	return &action
}
