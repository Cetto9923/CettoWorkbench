// =============================================================================
// 文件: internal/module/testtask/handler_write_authz.go
// 模块: 提测办理
// 类型: action
// 职责: 提测写入口的对象级鉴权 403 出口。响应体与第 3 轮排期业需排期保存
//       保持一致：{"success":false,"message":"…"}。
// 依赖: internal/pkg/errorx
// =============================================================================

package testtask

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"workbench/internal/pkg/errorx"
)

// writeWriteAuthZError 把 service 层的对象级写权限拒绝映射为 403，
// 其余错误沿用既有的 contextHTTPError 出口，不改变非鉴权失败的既有语义。
func (h *Handler) writeWriteAuthZError(c *gin.Context, demandID uint, err error, logMsg string) {
	if bizErr, ok := errorx.IsBizError(err); ok && bizErr.Code == errorx.ErrCodeForbidden {
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"message": bizErr.Msg,
		})
		return
	}
	if h.logger != nil {
		h.logger.Error(logMsg, zap.Error(err), zap.Uint("demand_id", demandID))
	}
	status, msg := contextHTTPError(err)
	c.JSON(status, gin.H{"success": false, "message": msg})
}
