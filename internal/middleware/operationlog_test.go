// =============================================================================
// 文件: internal/middleware/operationlog_test.go
// 模块: 中间件
// 类型: test
// 职责: 验证操作日志取到的操作人信息，不连接真实数据库。
// 依赖: internal/model
// =============================================================================

package middleware

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"workbench/internal/model"
)

func TestOperationLogActor(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, tc := range []struct {
		name        string
		user        *model.User
		wantID      int64
		wantAccount string
	}{
		{"已登录用户取账号", &model.User{ID: 5, Account: " zhangsan "}, 5, "zhangsan"},
		{"未登录为空", nil, 0, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = httptest.NewRequest("POST", "/x", nil)
			if tc.user != nil {
				c.Set("currentUser", tc.user)
			}
			gotID, gotAccount := operationLogActor(c, nil)
			if gotID != tc.wantID || gotAccount != tc.wantAccount {
				t.Fatalf("operationLogActor() = (%d, %q), want (%d, %q)", gotID, gotAccount, tc.wantID, tc.wantAccount)
			}
		})
	}
}
