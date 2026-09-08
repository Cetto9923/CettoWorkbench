package po

import "testing"

func TestGroupSubmitTestUnitsGroupsAndSortsServerScope(t *testing.T) {
	units := groupSubmitTestUnits([]submitTestStoryRow{
		{ID: 20, Product: 2, Branch: 0, Execution: 11, Project: 101, ProductName: "产品二", Title: "第二条"},
		{ID: 10, Product: 1, Branch: 0, Execution: 9, Project: 100, ProductName: "产品一", Title: "第一条"},
		{ID: 11, Product: 1, Branch: 0, Execution: 9, Project: 100, ProductName: "产品一", Title: "同组条目"},
	})

	if len(units) != 2 {
		t.Fatalf("unit count = %d, want 2", len(units))
	}
	if units[0].Product != 1 || units[0].Execution != 9 {
		t.Fatalf("first unit = product %d execution %d, want product 1 execution 9", units[0].Product, units[0].Execution)
	}
	if len(units[0].Stories) != 2 || units[0].Stories[0].ID != 10 || units[0].Stories[1].ID != 11 {
		t.Fatalf("first unit stories = %#v, want [10 11]", units[0].Stories)
	}
	if units[1].Product != 2 || units[1].Stories[0].ID != 20 {
		t.Fatalf("second unit = %#v, want product 2/story 20", units[1])
	}
}
