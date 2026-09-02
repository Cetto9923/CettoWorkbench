// =============================================================================
// 文件: internal/middleware/sqlrequest.go
// 模块: 中间件
// 类型: middleware
// 职责: 为每个 HTTP 请求注入 request_id 并在结束时写入 SQL 汇总日志。
// 依赖: internal/pkg/sqllog
// =============================================================================

package middleware

import (
	"time"

	"github.com/gin-gonic/gin"

	"workbench/internal/pkg/sqllog"
)

// SQLRequestContext 注入 request_id 与 SQL 请求上下文，请求结束后写汇总行。
func SQLRequestContext() gin.HandlerFunc {
	return func(c *gin.Context) {
		state := &sqllog.RequestState{
			RequestID: sqllog.NewRequestID(),
			Method:    c.Request.Method,
			Start:     time.Now(),
		}
		c.Request = c.Request.WithContext(sqllog.WithRequestState(c.Request.Context(), state))
		c.Next()

		route := c.FullPath()
		if route == "" {
			route = c.Request.URL.Path
		}
		sqllog.LogRequestSummary(state, route, time.Since(state.Start))
	}
}
