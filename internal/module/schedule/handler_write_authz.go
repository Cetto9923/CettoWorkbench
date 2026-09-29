// =============================================================================
// 文件: internal/module/schedule/handler_write_authz.go
// 模块: 排期工作台
// 类型: action
// 职责: 排期其余写入口（研发需求排期/任务、版本窗口）的对象级鉴权 403 出口。
//       响应体与第 3 轮业需排期保存保持一致：{"success":false,"message":"…"}。
//       单独成文件，避免让已超 500 行的 handler.go / handler_demand.go 继续变长。
// 依赖: internal/pkg/errorx
// =============================================================================

package schedule

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"workbench/internal/pkg/errorx"
)

// writeForbiddenJSON 输出无权访问的统一 403 响应体。
func writeForbiddenJSON(c *gin.Context, msg string) {
	c.JSON(http.StatusForbidden, gin.H{
		"success": false,
		"message": msg,
	})
}

// writeWriteAuthZError 把 service 层的对象级写权限拒绝映射为 403，
// 其余错误按 500 兜底。用于研发需求排期保存与维护任务保存。
func (h *Handler) writeWriteAuthZError(c *gin.Context, logMsg string, objectID uint, err error) {
	if bizErr, ok := errorx.IsBizError(err); ok && bizErr.Code == errorx.ErrCodeForbidden {
		writeForbiddenJSON(c, bizErr.Msg)
		return
	}
	if h.logger != nil {
		h.logger.Error(logMsg, zap.Error(err), zap.Uint("object_id", objectID))
	}
	c.JSON(http.StatusInternalServerError, gin.H{
		"success": false,
		"message": err.Error(),
	})
}

// writeWindowAuthZError 是版本窗口写入口的出口：对象级写权限拒绝返回 403，
// 其余错误沿用既有的 200 + success:false 形态，不改变非鉴权失败的既有语义。
func (h *Handler) writeWindowAuthZError(c *gin.Context, windowID uint64, err error, logMsg, fallbackMsg string) {
	if bizErr, ok := errorx.IsBizError(err); ok && bizErr.Code == errorx.ErrCodeForbidden {
		writeForbiddenJSON(c, bizErr.Msg)
		return
	}
	if h.logger != nil {
		h.logger.Error(logMsg, zap.Error(err), zap.Uint64("window_id", windowID))
	}
	c.JSON(http.StatusOK, gin.H{
		"success": false,
		"error":   fallbackMsg,
	})
}
