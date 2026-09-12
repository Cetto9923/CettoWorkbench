package testtask

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
	"workbench/internal/pkg/zentao"
)

// fakeZentaoFixture 封装一个 httptest 服务，对 /tokens 与 /projects/:id/testtasks 返回可控结果。
type fakeZentaoFixture struct {
	server         *httptest.Server
	lastPayload    atomic.Value
	tokenAccount   atomic.Value
	tokenIssued    atomic.Int32
	createCalls    atomic.Int32
	failNextCreate atomic.Int32 // 从下一次创建请求开始连续失败 N 次（用于前 N 条全失败场景）
	failOnCall     atomic.Int32 // 失败「第 N 次」创建请求：1 表示第 1 次失败；用于「先成功若干条后再失败」场景
	delayCreate    time.Duration
}

func newFakeZentao(t *testing.T) *fakeZentaoFixture {
	t.Helper()
	f := &fakeZentaoFixture{}
	f.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/tokens"):
			f.tokenIssued.Add(1)
			var login map[string]any
			_ = json.NewDecoder(r.Body).Decode(&login)
			if account, ok := login["account"].(string); ok {
				f.tokenAccount.Store(account)
			}
			_, _ = io.WriteString(w, `{"token":"tok-test","tokenExpired":false}`)
		case r.Method == http.MethodPost && strings.Contains(r.URL.Path, "/projects/") && strings.HasSuffix(r.URL.Path, "/testtasks"):
			if d := f.delayCreate; d > 0 {
				time.Sleep(d)
			}
			body, _ := io.ReadAll(r.Body)
			_ = body
			nextCall := f.createCalls.Load() + 1
			if target := f.failOnCall.Load(); target > 0 && target == nextCall {
				// 仅在指定序号上失败：让前 N-1 条正常返回
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = io.WriteString(w, `{"message":"禅道服务异常"}`)
				return
			}
			if remaining := f.failNextCreate.Load(); remaining > 0 {
				f.failNextCreate.Add(-1)
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = io.WriteString(w, `{"message":"禅道服务异常"}`)
				return
			}
			f.createCalls.Add(1)
			// 校验请求中包含必要字段
			var payload map[string]any
			_ = json.Unmarshal(body, &payload)
			f.lastPayload.Store(payload)
			w.WriteHeader(http.StatusOK)
			_, _ = fmt.Fprintf(w, `{"id":%d,"name":"%s"}`, 9000+f.createCalls.Load(), "tt")
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(f.server.Close)
	return f
}

func (f *fakeZentaoFixture) client() *zentao.Client {
	return zentao.NewClient(f.server.URL)
}
