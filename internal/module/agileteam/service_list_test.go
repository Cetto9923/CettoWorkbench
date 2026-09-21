package agileteam

import "testing"

func TestBuildTeamFamiliesKeepsParentContext(t *testing.T) {
	t.Parallel()
	parent := ListItem{ID: 1, Name: "信贷专项团队", Type: "parent"}
	child := ListItem{ID: 11, Name: "对公1组", ParentID: 1, ParentName: "信贷专项团队", Type: "child"}
	orphan := ListItem{ID: 99, Name: "独立组", Type: "parent"}
	all := map[uint]ListItem{1: parent, 11: child, 99: orphan}

	families := buildTeamFamilies([]ListItem{child}, all)
	if len(families) != 1 {
		t.Fatalf("want 1 family, got %d", len(families))
	}
	if families[0].Parent.ID != 1 || families[0].Parent.ChildCount != 1 {
		t.Fatalf("parent context missing: %+v", families[0].Parent)
	}
	if len(families[0].Children) != 1 || families[0].Children[0].ID != 11 {
		t.Fatalf("child missing: %+v", families[0].Children)
	}
}

func TestPageFamiliesDoesNotSplitParentFromChildren(t *testing.T) {
	t.Parallel()
	families := []teamFamily{
		{Parent: ListItem{ID: 1, Name: "P1"}, Children: []ListItem{{ID: 2}, {ID: 3}}},
		{Parent: ListItem{ID: 10, Name: "P2"}, Children: []ListItem{{ID: 11}}},
	}
	items, pages := pageFamilies(families, 1, 3)
	if pages != 2 {
		t.Fatalf("want 2 pages (P1 family=3 rows fills page1), got %d", pages)
	}
	if len(items) != 3 || items[0].ID != 1 || items[2].ID != 3 {
		t.Fatalf("page1 should be entire P1 family, got %+v", items)
	}
	items2, _ := pageFamilies(families, 2, 3)
	if len(items2) != 2 || items2[0].ID != 10 {
		t.Fatalf("page2 should be P2 family, got %+v", items2)
	}
}
