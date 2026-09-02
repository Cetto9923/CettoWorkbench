// =============================================================================
// 文件: internal/module/user/handler_export.go
// 模块: 用户管理
// 类型: crud
// 职责: 用户列表导出及导出行/表头装配。
// 依赖: internal/middleware
//
//	internal/model
//	internal/pkg/fileexport
//	internal/pkg/flash
//
// =============================================================================

package user

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"workbench/internal/middleware"
	"workbench/internal/model"
	"workbench/internal/pkg/fileexport"
	"workbench/internal/pkg/flash"
)

// Export 导出用户列表并触发浏览器下载。
func (h *Handler) Export(c *gin.Context) {
	var req ExportReq
	if err := c.ShouldBind(&req); err != nil {
		flash.Error(c, "导出失败: 参数解析失败")
		c.Redirect(http.StatusSeeOther, "/admin/users")
		return
	}
	if fieldErrs := req.Validate(); len(fieldErrs) > 0 {
		flash.Error(c, "导出失败: 参数不合法")
		c.Redirect(http.StatusSeeOther, "/admin/users")
		return
	}

	actor := middleware.CurrentUser(c)
	users, err := h.svc.Export(c.Request.Context(), actor, req)
	if err != nil {
		h.logger.Error("export users failed", zap.Error(err))
		flash.Error(c, "导出失败，请稍后重试")
		c.Redirect(http.StatusSeeOther, "/admin/users")
		return
	}

	fileName := fileexport.BuildFileName(req.FileName, req.FileType, "用户列表")
	if err := fileexport.WriteTable(c.Writer, fileName, req.FileType, exportHeaders(), usersToStringRows(users)); err != nil {
		h.logger.Error("export users failed", zap.Error(err))
		c.String(http.StatusInternalServerError, "导出失败")
		return
	}
}

// exportHeaders 导出表头。
func exportHeaders() []string {
	return []string{"ID", "账号", "邮箱", "姓名", "性别", "状态", "最后登录时间"}
}

// usersToStringRows 将用户列表转为二维字符串数组,供 fileexport 写入。
func usersToStringRows(users []model.User) [][]string {
	rows := make([][]string, 0, len(users))
	for _, item := range users {
		rows = append(rows, []string{
			strconv.FormatInt(item.ID, 10),
			item.Account,
			item.Email,
			item.DisplayName,
			genderLabel(item.Gender),
			activeLabel(item.IsActive),
			formatTime(item.LastLoginDateDB),
		})
	}
	return rows
}
