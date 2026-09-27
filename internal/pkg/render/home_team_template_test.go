package render

import (
	"html/template"
	"path/filepath"
	"runtime"
	"testing"
)

func TestHomeTeamTemplateIsPartOfProductionTemplateCache(t *testing.T) {
	_, source, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve test source path")
	}
	renderer := &Renderer{templateDir: filepath.Clean(filepath.Join(filepath.Dir(source), "../../../web/templates")), cache: make(map[string]*template.Template)}
	if err := renderer.warmCache(); err != nil {
		t.Fatalf("load production templates: %v", err)
	}
	if renderer.cache["po/home_team"] == nil {
		t.Fatal("po/home_team template was not loaded into the production cache")
	}
}
