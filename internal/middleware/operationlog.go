// =============================================================================
// 文件: internal/middleware/operationlog.go
// 模块: 中间件
// 类型: middleware
// 职责: 记录写操作请求的操作日志。
// 依赖: internal/model
// =============================================================================

package middleware

import (
	"net/http"
	"net/url"
	"strings"
	"sync"

	"github.com/alexedwards/scs/v2"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"workbench/internal/model"
)

var sensitiveOperationLogFields = map[string]struct{}{
	"password":        {},
	"newpassword":     {},
	"confirmpassword": {},
	"oldpassword":     {},
	"token":           {},
	"secret":          {},
}

var operationLogTableInit sync.Once

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

		c.Next()

		if db == nil {
			return
		}
		operationLogTableInit.Do(func() {
			// 启动后首次写操作日志时兜底建表，避免因目标库缺表导致审计日志永久写入失败。
			_ = db.AutoMigrate(&model.OperationLog{})
		})

		userID := int64(0)
		account := ""
		if sessionMgr != nil {
			userID = sessionMgr.GetInt64(c.Request.Context(), "userID")
			if userID <= 0 {
				userID = sessionMgr.GetInt64(c.Request.Context(), "userId")
			}
			account = strings.TrimSpace(sessionMgr.GetString(c.Request.Context(), "account"))
		}

		body := buildOperationLogBody(c.Request.PostForm)
		if len(c.Request.PostForm) == 0 {
			_ = c.Request.ParseForm()
			body = buildOperationLogBody(c.Request.PostForm)
		}

		logEntry := model.OperationLog{
			TenantID:   0,
			UserID:     uint64(maxInt64(userID)),
			Account:    account,
			Method:     c.Request.Method,
			Path:       c.Request.URL.Path,
			Query:      c.Request.URL.RawQuery,
			Body:       body,
			IP:         c.ClientIP(),
			UserAgent:  c.Request.UserAgent(),
			StatusCode: c.Writer.Status(),
		}

		go func(entry model.OperationLog) {
			_ = db.Create(&entry).Error
		}(logEntry)
	}
}

func buildOperationLogBody(postForm url.Values) string {
	if len(postForm) == 0 {
		return ""
	}
	filteredForm := make(url.Values, len(postForm))
	for key, values := range postForm {
		safeValues := make([]string, len(values))
		copy(safeValues, values)
		if _, sensitive := sensitiveOperationLogFields[strings.ToLower(strings.TrimSpace(key))]; sensitive {
			for idx := range safeValues {
				safeValues[idx] = "[FILTERED]"
			}
		}
		filteredForm[key] = safeValues
	}
	return filteredForm.Encode()
}

func maxInt64(v int64) int64 {
	if v < 0 {
		return 0
	}
	return v
}
