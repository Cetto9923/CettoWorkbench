// =============================================================================
// 文件: internal/module/po/handler_story_deliver_test.go
// 模块: PO 工作台
// 类型: test
// 职责: 研需交付错误返回与取消操作不得绕过必填验证。
// =============================================================================
package po

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestStoryDeliverInvalidInputs(t *testing.T) {
	for _, body := range []string{`{}`, `{"mode":"cancel"}`, `{"mode":"edit"}`} {
		rec := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(rec)
		c.Request = httptest.NewRequest(http.MethodPost, "/stories/68485/deliver", strings.NewReader(body))
		c.Params = gin.Params{{Key: "id", Value: "68485"}}
		h := &Handler{}
		h.DeliverStory(c)
		if rec.Code != http.StatusUnprocessableEntity || !strings.Contains(rec.Body.String(), `"errors"`) {
			t.Fatalf("body=%s status=%d result=%s", body, rec.Code, rec.Body.String())
		}
	}
}

func TestDeliverActionErrorStatus(t *testing.T) {
	for _, tc := range []struct {
		err    error
		status int
	}{
		{errStoryNotFound, http.StatusNotFound}, {errHomeActionNotFound, http.StatusNotFound},
		{errHomeActionForbidden, http.StatusForbidden}, {errHomeActionConflict, http.StatusConflict},
		{fmt.Errorf("wrapped: %w", errDeliverBlocked), http.StatusConflict},
	} {
		rec := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(rec)
		writeDeliverActionError(c, nil, "test", 68485, tc.err)
		if rec.Code != tc.status {
			t.Fatalf("err=%v status=%d want=%d", tc.err, rec.Code, tc.status)
		}
	}
}
