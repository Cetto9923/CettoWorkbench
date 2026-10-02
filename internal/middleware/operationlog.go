// =============================================================================
// 文件: internal/middleware/operationlog.go
// 模块: 中间件
// 类型: middleware
// 职责: 记录写操作请求的操作日志。
// 依赖: internal/model
// =============================================================================

package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
	"workbench/internal/pkg/redact"

	"github.com/alexedwards/scs/v2"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"workbench/internal/model"
)

// RecordOperationLog 记录 POST/PUT/DELETE 请求的操作审计日志。
func RecordOperationLog(db *gorm.DB, sessionMgr *scs.SessionManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method == http.MethodGet {
			c.Next()
			return
		}
		if c.Request.Method != http.MethodPost &&
			c.Request.Method != http.MethodPut &&
			c.Request.Method != http.MethodDelete {
			c.Next()
			return
		}

		var raw []byte
		if strings.Contains(c.Request.Header.Get("Content-Type"), "application/json") && c.Request.Body != nil {
			raw, _ = io.ReadAll(io.LimitReader(c.Request.Body, 64*1024+1))
			c.Request.Body = io.NopCloser(io.MultiReader(bytes.NewReader(raw), c.Request.Body))
		}
		c.Next()

		if db == nil {
			return
		}

		userID, account := operationLogActor(c, sessionMgr)

		body := buildOperationLogBody(c.Request.PostForm)
		if len(c.Request.PostForm) == 0 {
			_ = c.Request.ParseForm()
			body = buildOperationLogBody(c.Request.PostForm)
		}

		if len(raw) > 0 {
			safe, _ := json.Marshal(redact.JSON(raw))
			body = string(safe)
		}

		logEntry := model.OperationLog{
			TenantID:   0,
			UserID:     uint64(nonNegativeInt64(userID)),
			Account:    account,
			Method:     c.Request.Method,
			Path:       c.Request.URL.Path,
			Query:      buildOperationLogBody(c.Request.URL.Query()),
			Body:       body,
			IP:         c.ClientIP(),
			UserAgent:  c.Request.UserAgent(),
			StatusCode: c.Writer.Status(),
		}

		if err := saveOperationLogSafely(db.WithContext(c.Request.Context()), logEntry); err != nil {
			zap.L().Error("record operation log failed", zap.Error(err))
		}
	}
}

// operationLogActor 优先取 RequireLogin 注入的当前用户；会话中只存 userID，不存账号。
func operationLogActor(c *gin.Context, sessionMgr *scs.SessionManager) (int64, string) {
	if u := CurrentUser(c); u != nil {
		return u.ID, strings.TrimSpace(u.Account)
	}
	userID := int64(0)
	if sessionMgr != nil {
		userID = sessionMgr.GetInt64(c.Request.Context(), "userID")
		if userID <= 0 {
			userID = sessionMgr.GetInt64(c.Request.Context(), "userId")
		}
	}
	return userID, ""
}

func buildOperationLogBody(postForm url.Values) string {
	if len(postForm) == 0 {
		return ""
	}
	filteredForm := make(url.Values, len(postForm))
	for key, values := range postForm {
		safeValues := make([]string, len(values))
		copy(safeValues, values)
		if redact.Sensitive(key) {
			for idx := range safeValues {
				safeValues[idx] = "[FILTERED]"
			}
		}
		filteredForm[key] = safeValues
	}
	return filteredForm.Encode()
}

func nonNegativeInt64(v int64) int64 {
	if v < 0 {
		return 0
	}
	return v
}

// 后台审计写入不能因驱动或回调 panic 终止整个进程。
func saveOperationLogSafely(db *gorm.DB, entry model.OperationLog) (err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("operation log panic: %v", recovered)
		}
	}()
	ctx, cancel := context.WithTimeout(context.WithoutCancel(db.Statement.Context), 250*time.Millisecond)
	defer cancel()
	return db.WithContext(ctx).Create(&entry).Error
}
