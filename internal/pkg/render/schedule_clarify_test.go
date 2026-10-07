// =============================================================================
// 文件: internal/pkg/render/schedule_clarify_test.go
// 模块: 基础设施
// 类型: test
// 职责: 验证窗口详细列表复用澄清弹窗，只为可澄清行提供入口。
// =============================================================================
package render

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
)

func TestScheduleWindowClarifyEntry(t *testing.T) {
	t.Chdir(filepath.Join("..", "..", ".."))
	r := &Renderer{templateDir: "web/templates"}
	tpl, err := r.parseTemplates("schedule/index")
	if err != nil {
		t.Fatal(err)
	}
	data := map[string]any{
		"SelectedWindowMap": map[int]bool{2: true},
		"Windows":           []map[string]any{{"ID": 2, "ShortName": "测试窗口", "UsedHours": 0, "CapacityHours": 35, "UsedPercent": 0, "RemainingHours": 35, "DemandCount": 2, "BlockedCount": 0}},
		"BizRequirements": []map[string]any{
			{"DemandID": 101, "ID": "US101", "CanClarify": true, "ActionLabel": "去排期"},
			{"DemandID": 102, "ID": "US102", "CanClarify": false, "ActionLabel": "去排期"},
			{"DemandID": 103, "ID": "US103", "CanClarify": false, "ActionLabel": "详情"},
		},
		"BizTotal": int64(2), "IndependentTotal": int64(0),
	}
	var out bytes.Buffer
	if err := tpl.ExecuteTemplate(&out, "content", data); err != nil {
		t.Fatal(err)
	}
	html := out.String()
	if strings.Count(html, "js-drawer-clarify-btn") != 1 || !strings.Contains(html, `data-demand-id="101">澄清`) {
		t.Fatal("eligible row must have exactly one clarify entry")
	}
	if !strings.Contains(html, `id="poDemandClarifyModal"`) {
		t.Fatal("shared clarify modal missing")
	}
	if strings.Count(html, "js-change-window") != 1 || !strings.Contains(html, `js-change-window" data-demand-id="103"`) {
		t.Fatal("only parent detail row must retain change-window entry")
	}
}
