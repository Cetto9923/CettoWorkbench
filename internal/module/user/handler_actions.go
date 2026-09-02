// =============================================================================
// 文件: internal/module/user/handler_actions.go
// 模块: 用户管理
// 类型: crud
// 职责: 用户删除、状态切换与密码重置等动作型写操作。
// 依赖: internal/middleware
//
//	internal/pkg/flash
//	internal/pkg/render
//
// =============================================================================

package user

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"workbench/internal/middleware"
	"workbench/internal/pkg/flash"
	"workbench/internal/pkg/render"
)

// Delete 删除用户
func (h *Handler) Delete(c *gin.Context) {
	id, err := strconv.ParseInt(strings.TrimSpace(c.Param("id")), 10, 64)
	if err != nil || id <= 0 {
		flash.Error(c, "删除用户失败: 用户ID不合法")
		c.Redirect(http.StatusSeeOther, "/admin/users")
		return
	}

	actor := middleware.CurrentUser(c)
	err = h.svc.Delete(c.Request.Context(), actor, DeleteReq{ID: id})
	if err != nil {
		h.logger.Error("delete user failed", zap.Int64("userID", id), zap.Error(err))
		flash.Error(c, "删除用户失败，请稍后重试")
		c.Redirect(http.StatusSeeOther, "/admin/users")
		return
	}

	flash.Success(c, "删除成功")
	c.Redirect(http.StatusSeeOther, "/admin/users")
}

// ToggleStatus 切换用户启用状态。
func (h *Handler) ToggleStatus(c *gin.Context) {
	var req ToggleStatusReq
	if err := c.ShouldBindUri(&req); err != nil {
		flash.Error(c, "切换用户状态失败: 参数解析失败")
		render.Redirect(c, "/admin/users")
		return
	}
	if err := c.ShouldBind(&req); err != nil {
		flash.Error(c, "切换用户状态失败: 参数解析失败")
		render.Redirect(c, "/admin/users")
		return
	}
	actor := middleware.CurrentUser(c)
	if err := h.svc.ToggleStatus(c.Request.Context(), actor, req); err != nil {
		flash.Error(c, "切换用户状态失败: "+err.Error())
		render.Redirect(c, "/admin/users")
		return
	}
	flash.Success(c, "用户状态切换成功")
	render.Redirect(c, "/admin/users")
}

// ResetPassword 重置用户密码。
func (h *Handler) ResetPassword(c *gin.Context) {
	var req ResetPasswordReq
	if err := c.ShouldBindUri(&req); err != nil {
		flash.Error(c, "重置密码失败: 参数解析失败")
		render.Redirect(c, "/admin/users")
		return
	}
	if err := c.ShouldBind(&req); err != nil {
		flash.Error(c, "重置密码失败: 参数解析失败")
		render.Redirect(c, "/admin/users")
		return
	}

	if fieldErrs := req.Validate(); len(fieldErrs) > 0 {
		var msgs []string
		for _, item := range fieldErrs {
			if item.Message != "" {
				msgs = append(msgs, item.Message)
			}
		}
		flash.Error(c, "重置密码失败: "+strings.Join(msgs, "；"))
		render.Redirect(c, "/admin/users")
		return
	}

	actor := middleware.CurrentUser(c)
	if err := h.svc.ResetPassword(c.Request.Context(), actor, req); err != nil {
		flash.Error(c, "重置密码失败: "+err.Error())
		render.Redirect(c, "/admin/users")
		return
	}
	flash.Success(c, "密码重置成功")
	render.Redirect(c, "/admin/users")
}
