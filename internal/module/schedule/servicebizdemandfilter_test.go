package schedule

import "testing"

func TestFilterUnscheduledBizDemandTree(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		topDemands   []ZtDemand
		children     map[int][]ZtDemand
		productCount map[uint]int
		wantTopIDs   []uint
		wantChildIDs []uint
	}{
		{
			name:       "hide parent when no children match",
			topDemands: []ZtDemand{{ID: 1, AssignedTo: "po"}},
			children: map[int][]ZtDemand{
				1: {
					{ID: 11, AssignedTo: "po"},
					{ID: 12, AssignedTo: "po"},
				},
			},
			productCount: map[uint]int{},
			wantTopIDs:   []uint{},
			wantChildIDs: nil,
		},
		{
			name:       "show parent and matching children only",
			topDemands: []ZtDemand{{ID: 2}},
			children: map[int][]ZtDemand{
				2: {
					{ID: 21, AssignedTo: "po"},
					{ID: 22, AssignedTo: "po"},
				},
			},
			productCount: map[uint]int{21: 1},
			wantTopIDs:   []uint{2},
			wantChildIDs: []uint{21},
		},
		{
			name:         "show leaf parent when it matches",
			topDemands:   []ZtDemand{{ID: 3, AssignedTo: "po"}},
			children:     map[int][]ZtDemand{},
			productCount: map[uint]int{3: 1},
			wantTopIDs:   []uint{3},
			wantChildIDs: nil,
		},
		{
			name:       "parent self match does not show when all children fail",
			topDemands: []ZtDemand{{ID: 4, AssignedTo: "po"}},
			children: map[int][]ZtDemand{
				4: {{ID: 41, AssignedTo: "po"}},
			},
			productCount: map[uint]int{4: 1},
			wantTopIDs:   []uint{},
			wantChildIDs: nil,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := &bizDemandAssembleContext{
				account:              "po",
				childByParent:        copyChildDemandMap(tt.children),
				storiesByDemand:      map[uint][]ZtStory{},
				productCountByDemand: tt.productCount,
				clarifyPMByDemand:    map[uint]bool{},
				windowByStory:        map[uint]StoryWindowRef{},
				taskStatByStory:      map[uint]StoryTaskStat{},
			}

			got := ctx.filterUnscheduledBizDemandTree(tt.topDemands)
			assertDemandIDs(t, got, tt.wantTopIDs)
			if tt.wantChildIDs != nil {
				assertDemandIDs(t, ctx.childByParent[int(tt.wantTopIDs[0])], tt.wantChildIDs)
			}
		})
	}
}

func copyChildDemandMap(in map[int][]ZtDemand) map[int][]ZtDemand {
	out := make(map[int][]ZtDemand, len(in))
	for parent, children := range in {
		out[parent] = append([]ZtDemand(nil), children...)
	}
	return out
}

func assertDemandIDs(t *testing.T, got []ZtDemand, want []uint) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("got %d demands, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i].ID != want[i] {
			t.Fatalf("got demand id at %d = %d, want %d", i, got[i].ID, want[i])
		}
	}
}
