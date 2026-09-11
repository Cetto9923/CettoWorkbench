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

func TestClientURLBuilding(t *testing.T) {
	tests := []struct {
		name        string
		baseURL     string
		apiPath     string
		expectedAPI string
		webPath     string
		expectedWeb string
	}{
		{
			name:        "bare domain",
			baseURL:     "http://127.0.0.1:8080",
			apiPath:     "/tokens",
			expectedAPI: "http://127.0.0.1:8080/api.php/v1/tokens",
			webPath:     "/demand-view-1.html",
			expectedWeb: "http://127.0.0.1:8080/demand-view-1.html",
		},
		{
			name:        "with trailing slash",
			baseURL:     "http://127.0.0.1:8080/",
			apiPath:     "tokens",
			expectedAPI: "http://127.0.0.1:8080/api.php/v1/tokens",
			webPath:     "demand-view-1.html",
			expectedWeb: "http://127.0.0.1:8080/demand-view-1.html",
		},
		{
			name:        "with /api.php/v1 suffix",
			baseURL:     "https://customer.chandao.net/api.php/v1",
			apiPath:     "/demand/1/withdrawReview",
			expectedAPI: "https://customer.chandao.net/api.php/v1/demand/1/withdrawReview",
			webPath:     "/demand-view-1.html",
			expectedWeb: "https://customer.chandao.net/demand-view-1.html",
		},
		{
			name:        "with /v1 suffix",
			baseURL:     "https://api.test/v1",
			apiPath:     "/tokens",
			expectedAPI: "https://api.test/api.php/v1/tokens",
			webPath:     "/home",
			expectedWeb: "https://api.test/home",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c := NewClient(tc.baseURL)
			if got := c.apiURL(tc.apiPath); got != tc.expectedAPI {
				t.Errorf("apiURL(%q) = %q, want %q", tc.apiPath, got, tc.expectedAPI)
			}
			if got := c.webURL(tc.webPath); got != tc.expectedWeb {
				t.Errorf("webURL(%q) = %q, want %q", tc.webPath, got, tc.expectedWeb)
			}
		})
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
