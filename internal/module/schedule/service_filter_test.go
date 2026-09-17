package schedule

import "testing"

func TestNormalizeFilterCountsTab(t *testing.T) {
	t.Parallel()

	tests := []struct {
		in   string
		want string
	}{
		{in: "", want: FilterCountsTabDemand},
		{in: "demand", want: FilterCountsTabDemand},
		{in: " story ", want: FilterCountsTabStory},
		{in: "STORY", want: FilterCountsTabStory},
		{in: "biz", want: FilterCountsTabDemand},
		{in: "indep", want: FilterCountsTabDemand},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.in+"→"+tt.want, func(t *testing.T) {
			t.Parallel()
			got := NormalizeFilterCountsTab(tt.in)
			if got != tt.want {
				t.Fatalf("NormalizeFilterCountsTab(%q)=%q, want %q", tt.in, got, tt.want)
			}
		})
	}
}
