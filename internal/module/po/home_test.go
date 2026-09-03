// =============================================================================
// 文件: internal/module/po/home_test.go
// 模块: PO 工作台
// 类型: action
// 职责: 固化首页阶段与前端真实性契约。
// 依赖: 无
// =============================================================================

package po

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestValueStreamStagesRemainOrdered(t *testing.T) {
	want := []string{
		"all",
		"accept",
		"clarify",
		"schedule",
		"developing",
		"testing",
		"waitacceptance",
		"acceptanced",
		"publish",
		"released",
	}

	got := make([]string, 0, len(valueStreamStages))
	for _, stage := range valueStreamStages {
		got = append(got, stage.status)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("value stream stage order = %v, want %v", got, want)
	}
}

func TestDemandsReqValidate(t *testing.T) {
	req := DemandsReq{Status: " schedule "}
	if errs := req.Validate(); len(errs) != 0 {
		t.Fatalf("valid status returned errors: %v", errs)
	}
	if req.Status != "schedule" {
		t.Fatalf("trimmed status = %q, want schedule", req.Status)
	}

	invalid := DemandsReq{Status: "unknown"}
	if errs := invalid.Validate(); len(errs) != 1 || errs[0].Field != "status" {
		t.Fatalf("invalid status errors = %v", errs)
	}
}

func TestHomeFrontendTruthContract(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	template := readHomeFile(t, filepath.Join(root, "web", "templates", "po", "home.html"))
	script := readHomeFile(t, filepath.Join(root, "web", "static", "js", "po", "home.js"))

	for _, marker := range []string{
		"$stage.DemandCount",
		"$stage.StoryCount",
		"aria-pressed=",
		"id=\"top5List\"",
		"href=\"/schedule\"",
		"今日必推",
		"待我处理",
		"阻塞",
		"超期",
		"挂起",
		".KPI.Today",
		".KPI.Blocked",
		".KPI.Overdue",
		".KPI.Suspended",
	} {
		if !strings.Contains(template, marker) {
			t.Errorf("home template missing %q", marker)
		}
	}

	for _, forbidden := range []string{
		"data-focal=",
		"风险 <strong>0</strong>",
		"最长 —天",
		"target=\"_blank\"",
		`home-hl-num">—<`, // 数字必须是真实值,禁止破折号占位
	} {
		if strings.Contains(template, forbidden) {
			t.Errorf("home template contains unsupported placeholder %q", forbidden)
		}
	}

	for _, forbidden := range []string{"window.open(", "target=\\\"_blank\\\"", "FOCAL_LABELS"} {
		if strings.Contains(script, forbidden) {
			t.Errorf("home script violates current-page or truth contract with %q", forbidden)
		}
	}
}

func readHomeFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(data)
}

// TestHomeHandlerRendersKPI 锁住 handler.go 渲染首页必须传 "KPI" 键给模板的契约。
// 回归 259e29bf 修复的 bug —— handler.go 曾因 render.Page 调用漏传 resp.KPI 而导致首页 5 个焦点摘要永远为 0。
// 真实数据由 Service 计算，但模板能否拿到 KPI 全靠 handler 显式传值；少一个 key 整行 KPI 消失。
func TestHomeHandlerRendersKPI(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	handler := readHomeFile(t, filepath.Join(root, "internal", "module", "po", "handler.go"))

	if !strings.Contains(handler, `"KPI"`) {
		t.Errorf("handler.go render.Page call must pass \"KPI\" key with resp.KPI value; missing in:\n%s", handler)
	}
	if !strings.Contains(handler, "resp.KPI") {
		t.Errorf("handler.go must reference resp.KPI (HomeResp.KPI), not a hardcoded empty struct; missing in:\n%s", handler)
	}
}
