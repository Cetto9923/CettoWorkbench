package po

import (
	"reflect"
	"testing"
)

// TestBuildClarifyProductItems 覆盖产品/PM 名称回退与主系统标记。
func TestBuildClarifyProductItems(t *testing.T) {
	data := clarifyFormData{
		prodMap: map[string]string{"101": "个人金融", "102": "对公金融"},
		userMap: map[string]string{"zhangsan": "张三", "lisi": "李四"},
	}

	tests := []struct {
		name       string
		mainSystem string
		clarifies  []demandClarifyDBItem
		want       []ClarifyProductItem
	}{
		{
			name:       "product and pm names resolved from maps",
			mainSystem: "101",
			clarifies:  []demandClarifyDBItem{{ID: 1, Product: "101", PM: "zhangsan"}},
			want: []ClarifyProductItem{
				{ID: 1, ProductID: "101", ProductName: "个人金融", PM: "zhangsan", PMName: "张三", IsMainSystem: true},
			},
		},
		{
			name:       "unknown product and pm fall back to raw values",
			mainSystem: "101",
			clarifies:  []demandClarifyDBItem{{ID: 2, Product: "999", PM: "unknown"}},
			want: []ClarifyProductItem{
				{ID: 2, ProductID: "999", ProductName: "999", PM: "unknown", PMName: "unknown"},
			},
		},
		{
			name:       "empty main system never marks main",
			mainSystem: "",
			clarifies:  []demandClarifyDBItem{{ID: 3, Product: "101"}},
			want: []ClarifyProductItem{
				{ID: 3, ProductID: "101", ProductName: "个人金融"},
			},
		},
		{
			name:       "empty input yields empty slice",
			mainSystem: "",
			clarifies:  nil,
			want:       []ClarifyProductItem{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildClarifyProductItems(&demandClarifyRawRow{MainSystem: tt.mainSystem}, tt.clarifies, data)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("buildClarifyProductItems() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

// TestBuildClarifyProductItems_CarriesDBFields 校验 DB 行字段原样搬运。
func TestBuildClarifyProductItems_CarriesDBFields(t *testing.T) {
	data := clarifyFormData{
		prodMap: map[string]string{"102": "对公金融"},
		userMap: map[string]string{"lisi": "李四"},
	}

	got := buildClarifyProductItems(&demandClarifyRawRow{}, []demandClarifyDBItem{{
		ID:                   4,
		Product:              "102",
		PM:                   "lisi",
		DemandCompletionDate: "2026-01-02",
		SystemClarifyDesc:    "desc",
		IsAdditionalInfo:     "0",
		AdditionalInfo:       "info",
	}}, data)

	want := ClarifyProductItem{
		ID:                   4,
		ProductID:            "102",
		ProductName:          "对公金融",
		PM:                   "lisi",
		PMName:               "李四",
		DemandCompletionDate: "2026-01-02",
		SystemClarifyDesc:    "desc",
		IsAdditionalInfo:     "0",
		AdditionalInfo:       "info",
	}
	if !reflect.DeepEqual(got, []ClarifyProductItem{want}) {
		t.Errorf("buildClarifyProductItems() = %+v, want %+v", got, want)
	}
}

// TestBuildClarifyUserStoryItems 覆盖点数关键字与 revpoint 回退。
func TestBuildClarifyUserStoryItems(t *testing.T) {
	cfg := ClarifyConfigData{PointToKeyword: map[string]string{"3": "M", "5": "L"}}

	tests := []struct {
		name        string
		userStories []demandUserStoryDBItem
		want        []ClarifyUserStoryItem
	}{
		{
			name:        "revpoint zero falls back to point",
			userStories: []demandUserStoryDBItem{{ID: 1, Point: 3, Revpoint: 0}},
			want:        []ClarifyUserStoryItem{{ID: 1, Point: 3, PointKeyword: "M", Revpoint: 3, Checked: true}},
		},
		{
			name: "non-zero revpoint kept while keyword still keyed by point",
			userStories: []demandUserStoryDBItem{
				{ID: 2, Point: 3, Revpoint: 5},
			},
			want: []ClarifyUserStoryItem{{ID: 2, Point: 3, PointKeyword: "M", Revpoint: 5, Checked: true}},
		},
		{
			name: "carried fields are copied verbatim",
			userStories: []demandUserStoryDBItem{
				{ID: 4, Role: "dev", GV: "张三", Product: "101", SourceType: "manual", AICode: "AI-1"},
			},
			want: []ClarifyUserStoryItem{{
				ID:         4,
				Role:       "dev",
				GV:         "张三",
				ProductID:  "101",
				SourceType: "manual",
				AICode:     "AI-1",
				Checked:    true,
			}},
		},
		{
			name:        "empty input yields empty slice",
			userStories: nil,
			want:        []ClarifyUserStoryItem{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildClarifyUserStoryItems(tt.userStories, cfg)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("buildClarifyUserStoryItems() = %+v, want %+v", got, tt.want)
			}
		})
	}
}
