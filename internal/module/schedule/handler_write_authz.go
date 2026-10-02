// =============================================================================
// 文件: internal/module/schedule/handler_write_authz.go
// 模块: 排期工作台
// 类型: action
// 职责: 排期写入口 403/404/409；版本窗口非鉴权失败仍保持 200 + success:false。
// 依赖: internal/pkg/errorx
// =============================================================================

package schedule

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"workbench/internal/pkg/errorx"
)

func writeForbiddenJSON(c *gin.Context, msg string) {
	c.JSON(http.StatusForbidden, gin.H{"success": false, "message": msg})
}

func (h *Handler) writeWriteAuthZError(c *gin.Context, logMsg string, objectID uint, err error) {
	if bizErr, ok := errorx.IsBizError(err); ok {
		switch bizErr.Code {
		case errorx.ErrCodeForbidden:
			writeForbiddenJSON(c, bizErr.Msg)
			return
		case errorx.ErrCodeNotFound:
			c.JSON(http.StatusNotFound, gin.H{"success": false, "message": bizErr.Msg})
			return
		case errorx.ErrCodeConflict:
			c.JSON(http.StatusConflict, gin.H{"success": false, "message": bizErr.Msg})
			return
		}
	}
	if h.logger != nil {
		h.logger.Error(logMsg, zap.Error(err), zap.Uint("object_id", objectID))
	}
	c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
}

func clientErrorText(err error) string {
	if biz, ok := errorx.IsBizError(err); ok {
		return biz.Msg
	}
	return err.Error()
}

func (h *Handler) writeWindowAuthZError(c *gin.Context, windowID uint64, err error, logMsg, fallbackMsg string) {
	if bizErr, ok := errorx.IsBizError(err); ok && bizErr.Code == errorx.ErrCodeForbidden {
		writeForbiddenJSON(c, bizErr.Msg)
		return
	}
	if h.logger != nil {
		h.logger.Error(logMsg, zap.Error(err), zap.Uint64("window_id", windowID))
	}
	c.JSON(http.StatusOK, gin.H{"success": false, "error": fallbackMsg})
}
