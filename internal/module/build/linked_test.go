// =============================================================================
// 文件: internal/module/build/linked_test.go
// 模块: 版本管理
// 类型: action
// 职责: BuildLinkedStoryItems 保持已关联 ID 顺序的单测。
// 依赖: 无
// =============================================================================

package build

import "testing"

func TestBuildLinkedStoryItemsPreservesOrder(t *testing.T) {
	ids := []uint{18111, 10007, 9315}
	byID := map[uint]storyTitleRow{
		10007: {ID: 10007, Title: "需求B"},
		18111: {ID: 18111, Title: "需求A"},
		9315:  {ID: 9315, Title: ""},
	}
	got := BuildLinkedStoryItems(ids, byID)
	if len(got) != 3 {
		t.Fatalf("len: got %d, want 3", len(got))
	}
	if got[0].ID != 18111 || got[0].Title != "需求A" {
		t.Fatalf("got[0]: %+v", got[0])
	}
	if got[1].ID != 10007 || got[1].Title != "需求B" {
		t.Fatalf("got[1]: %+v", got[1])
	}
	if got[2].ID != 9315 || got[2].Title != "" {
		t.Fatalf("got[2]: %+v", got[2])
	}
	if empty := BuildLinkedStoryItems(nil, nil); empty == nil || len(empty) != 0 {
		t.Fatalf("nil: %#v", empty)
	}
	// 缺失标题的 id 仍保留，便于前端展示 #id
	partial := BuildLinkedStoryItems([]uint{1, 2}, map[uint]storyTitleRow{1: {ID: 1, Title: "有"}})
	if len(partial) != 2 || partial[1].ID != 2 || partial[1].Title != "" {
		t.Fatalf("partial: %+v", partial)
	}
}
