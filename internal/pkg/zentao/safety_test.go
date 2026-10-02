// =============================================================================
// 文件: internal/pkg/zentao/safety_test.go
// 模块: 基础设施
// 类型: test
// 职责: 验证日志不泄漏凭据，以及业务失败不能被 HTTP 200 掩盖。
// =============================================================================

package zentao

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"workbench/internal/config"
)

func TestAPILogRedactsNestedCredentials(t *testing.T) {
	raw := []byte(`{"token":"synthetic-secret-a","data":[{"oldPassword":"synthetic-secret-b","refresh_token":"synthetic-secret-c"}],"id":7}`)
	safe, err := json.Marshal(encodeResponseBody(raw))
	if err != nil {
		t.Fatal(err)
	}
	for _, secret := range []string{"synthetic-secret-a", "synthetic-secret-b", "synthetic-secret-c"} {
		if strings.Contains(string(safe), secret) {
			t.Fatalf("credential reached log: %s", safe)
		}
	}
	if !strings.Contains(string(safe), `"id":7`) {
		t.Fatalf("lost non-sensitive log context: %s", safe)
	}
	if encodeResponseBody([]byte("non-json secret")) == "non-json secret" {
		t.Fatal("unparsed response leaked")
	}
}

func TestNativeWriteHTTP200FailureAndReadNotice(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/tokens" {
			_, _ = w.Write([]byte(`{"token":"synthetic-test-token"}`))
			return
		}
		_, _ = w.Write([]byte(`{"result":"fail","message":"native action rejected"}`))
	}))
	defer upstream.Close()
	client := NewClient(config.ZentaoConfig{API: upstream.URL})
	ctx := WithAccount(context.Background(), "fixture")
	if err := client.Do(ctx, http.MethodPost, "/demand/1/deliver", nil, nil); err == nil || !strings.Contains(err.Error(), "native action rejected") {
		t.Fatalf("write failure = %v", err)
	}
	var notice struct {
		Result string `json:"result"`
	}
	if err := client.Do(ctx, http.MethodGet, "/demand/1/checkReviewToStoryNotice", nil, &notice); err != nil || notice.Result != "fail" {
		t.Fatalf("read notice changed: %+v, %v", notice, err)
	}
}
