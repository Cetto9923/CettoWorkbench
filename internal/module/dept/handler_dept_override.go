// =============================================================================
// 文件: internal/module/dept/handler_dept_override.go
// 模块: 部门管理
// 类型: handler
// 职责: 科技本部部门负责人补缺（zt_wb_dept_manager_override）管理界面与 API。
// =============================================================================

package dept

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"workbench/internal/constants"
	"workbench/internal/middleware"
	"workbench/internal/pkg/render"
)

// ListOverrides 渲染部门负责人补缺管理页面
func (h *Handler) ListOverrides(c *gin.Context) {
	h.bindRenderer(c)
	ctx := c.Request.Context()

	items, err := h.svc.repo.ListOverrides(ctx)
	if err != nil {
		h.logger.Error("list dept overrides failed", zap.Error(err))
		render.Error(c, http.StatusInternalServerError, "获取部门补缺列表失败", err)
		return
	}

	depts, err := h.svc.repo.ListTechDepts(ctx)
	if err != nil {
		h.logger.Warn("list tech depts for dropdown failed", zap.Error(err))
	}

	render.Page(c, http.StatusOK, constants.TEMPLATE_DEPT_OVERRIDES, gin.H{
		"Title":      "部门负责人补缺管理",
		"PageTitle":  "部门负责人补缺管理（科技部本部及子孙部门）",
		"Items":      items,
		"TechDepts":  depts,
		"TotalCount": len(items),
	})
}

// SaveOverride 处理表单提交创建/修改补缺记录
func (cHandler *Handler) SaveOverride(c *gin.Context) {
	deptStr := strings.TrimSpace(c.PostForm("dept"))
	account := strings.TrimSpace(c.PostForm("account"))
	remark := strings.TrimSpace(c.PostForm("remark"))

	deptID, err := strconv.ParseUint(deptStr, 10, 32)
	if err != nil || deptID == 0 || account == "" {
		render.Error(c, http.StatusBadRequest, "部门ID和负责人账号不能为空", nil)
		return
	}

	actor := middleware.CurrentUser(c)
	operator := ""
	if actor != nil {
		operator = actor.Account
	}

	if err := cHandler.svc.repo.SaveOverride(c.Request.Context(), uint(deptID), account, remark, operator); err != nil {
		cHandler.logger.Error("save dept override failed", zap.Error(err))
		render.Error(c, http.StatusInternalServerError, "保存补缺记录失败", err)
		return
	}

	c.Redirect(http.StatusSeeOther, "/admin/dept-overrides")
}

// DeleteOverride 处理删除补缺记录请求
func (cHandler *Handler) DeleteOverride(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil || id == 0 {
		render.Error(c, http.StatusBadRequest, "无效的记录 ID", nil)
		return
	}

	actor := middleware.CurrentUser(c)
	operator := ""
	if actor != nil {
		operator = actor.Account
	}

	if err := cHandler.svc.repo.DeleteOverride(c.Request.Context(), id, operator); err != nil {
		cHandler.logger.Error("delete dept override failed", zap.Error(err))
		render.Error(c, http.StatusInternalServerError, "删除补缺记录失败", err)
		return
	}

	c.Redirect(http.StatusSeeOther, "/admin/dept-overrides")
}

// APIListOverrides JSON 接口：查询所有补缺记录
func (h *Handler) APIListOverrides(c *gin.Context) {
	items, err := h.svc.repo.ListOverrides(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": items})
}

// APISaveOverride JSON 接口：创建或更新补缺记录
func (h *Handler) APISaveOverride(c *gin.Context) {
	var req struct {
		Dept    uint   `json:"dept"`
		Account string `json:"account"`
		Remark  string `json:"remark"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "参数解析失败: " + err.Error()})
		return
	}
	if req.Dept == 0 || strings.TrimSpace(req.Account) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "部门ID与账号不能为空"})
		return
	}

	actor := middleware.CurrentUser(c)
	operator := ""
	if actor != nil {
		operator = actor.Account
	}

	if err := h.svc.repo.SaveOverride(c.Request.Context(), req.Dept, req.Account, req.Remark, operator); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "保存成功"})
}

// APIDeleteOverride JSON 接口：删除补缺记录
func (h *Handler) APIDeleteOverride(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "无效记录 ID"})
		return
	}

	actor := middleware.CurrentUser(c)
	operator := ""
	if actor != nil {
		operator = actor.Account
	}

	if err := h.svc.repo.DeleteOverride(c.Request.Context(), id, operator); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "删除成功"})
}
