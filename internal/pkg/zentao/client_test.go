package zentao

import (
	"errors"
	"strings"
	"testing"

	"workbench/internal/config"
)

func TestDefaultClientUsesAPIAndFallsBackToURL(t *testing.T) {
	SetConfig(config.ZentaoConfig{URL: "https://web.test", API: "https://api.test/v1"})
	if got := DefaultClient().baseURL; got != "https://api.test/v1" {
		t.Fatalf("DefaultClient baseURL = %q", got)
	}
	SetConfig(config.ZentaoConfig{URL: "https://web.test"})
	if got := DefaultClient().baseURL; got != "https://web.test" {
		t.Fatalf("DefaultClient fallback baseURL = %q", got)
	}
}

func TestParseZentaoAPIError(t *testing.T) {
	t.Run("plain error string", func(t *testing.T) {
		body := []byte(`{"error":"需求不存在"}`)
		err := parseZentaoAPIError(body, 400)
		if err == nil || !strings.Contains(err.Error(), "需求不存在") {
			t.Fatalf("expected error containing '需求不存在', got %v", err)
		}
		if !errors.Is(err, ErrZentaoAPIError) {
			t.Fatalf("expected ErrZentaoAPIError, got %v", err)
		}
	})

	t.Run("array error from PHP validation", func(t *testing.T) {
		body := []byte(`{"error":"Array"}`)
		err := parseZentaoAPIError(body, 400)
		if err == nil || !strings.Contains(err.Error(), "数据校验失败") {
			t.Fatalf("expected error containing '数据校验失败', got %v", err)
		}
	})

	t.Run("array of error messages", func(t *testing.T) {
		body := []byte(`{"error":["主系统不能为空","需求规模估算不能为空"]}`)
		err := parseZentaoAPIError(body, 400)
		if err == nil || !strings.Contains(err.Error(), "主系统不能为空; 需求规模估算不能为空") {
			t.Fatalf("expected concatenated error messages, got %v", err)
		}
	})

	t.Run("map of field error messages", func(t *testing.T) {
		body := []byte(`{"error":{"category":"需求类别不能为空"}}`)
		err := parseZentaoAPIError(body, 400)
		if err == nil || !strings.Contains(err.Error(), "category: 需求类别不能为空") {
			t.Fatalf("expected map error formatted, got %v", err)
		}
	})

	t.Run("message field fallback", func(t *testing.T) {
		body := []byte(`{"message":"保存失败"}`)
		err := parseZentaoAPIError(body, 400)
		if err == nil || !strings.Contains(err.Error(), "保存失败") {
			t.Fatalf("expected message parsed, got %v", err)
		}
	})
}
