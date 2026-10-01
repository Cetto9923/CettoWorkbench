// =============================================================================
// 文件: internal/pkg/render/home_testtask_test.go
// 模块: 基础设施
// 类型: infra
// 职责: 验证 po/home 模板集能解析并执行 components 版 po/testtask 片段
// 依赖: internal/pkg/render/render.go
// =============================================================================

package render

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
)

func TestPOHomeTemplateIncludesTesttaskModal(t *testing.T) {
	t.Chdir(filepath.Join("..", "..", ".."))

	r := &Renderer{templateDir: filepath.Clean("web/templates")}
	tpl, err := r.parseTemplates("po/home")
	if err != nil {
		t.Fatalf("parseTemplates(po/home): %v", err)
	}
	if tpl.Lookup("po/testtask") == nil {
		t.Fatal("template po/testtask missing from po/home set")
	}

	var buf bytes.Buffer
	if err := tpl.ExecuteTemplate(&buf, "po/testtask", map[string]any{}); err != nil {
		t.Fatalf("execute po/testtask: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, `id="poTesttaskModal"`) {
		t.Fatalf("po/testtask output missing modal root, got: %q", out)
	}
}
