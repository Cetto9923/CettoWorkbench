// =============================================================================
// 文件: internal/module/backup/handler.go
// 模块: 数据库备份
// 类型: action
// 职责: 处理备份记录列表、手动备份与文件下载请求。
// 依赖: internal/middleware
//       internal/pkg/flash
//       internal/pkg/perm
//       internal/pkg/render
// =============================================================================

package backup

import (
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"workbench/internal/constants"
	"workbench/internal/middleware"
	"workbench/internal/pkg/flash"
	"workbench/internal/pkg/perm"
	"workbench/internal/pkg/render"
)

// Handler 备份模块 Handler。
type Handler struct {
	logger *zap.Logger
	svc    *Service
}

// NewHandler 创建 Handler。
func NewHandler(logger *zap.Logger, svc *Service) *Handler {
	return &Handler{
		logger: logger,
		svc:    svc,
	}
}

// RegisterRoutes 注册模块路由。
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	g := rg.Group("/backups")
	g.Use(middleware.ActiveNav("/admin/backups"))
	{
		g.GET("", middleware.RequirePerm(perm.BackupList), h.List)
		g.POST("", middleware.RequirePerm(perm.BackupCreate), h.Create)
		g.GET("/:id/download", middleware.RequirePerm(perm.BackupDownload), h.Download)
	}
}

// List 备份记录列表页。
func (h *Handler) List(c *gin.Context) {
	var req ListReq
	if err := c.ShouldBindQuery(&req); err != nil {
		render.Error(c, http.StatusBadRequest, "参数解析失败", err)
		return
	}
	actor := middleware.CurrentUser(c)
	resp, err := h.svc.List(c.Request.Context(), actor, req)
	if err != nil {
		h.logger.Error("list backup records failed", zap.Error(err))
		render.Error(c, http.StatusInternalServerError, "获取备份记录失败", err)
		return
	}
	render.Page(c, http.StatusOK, constants.TEMPLATE_BACKUP_LIST, gin.H{
		"Title":     "数据库备份",
		"PageTitle": "数据库备份",
		"Records":   resp.Items,
		"Pager":     resp.Pager,
	})
}

// Create 手动触发备份。
func (h *Handler) Create(c *gin.Context) {
	actor := middleware.CurrentUser(c)
	resp, err := h.svc.Create(c.Request.Context(), actor, CreateReq{})
	if err != nil {
		flash.Error(c, "创建备份失败: "+err.Error())
		c.Redirect(http.StatusSeeOther, "/admin/backups")
		return
	}
	flash.Success(c, "备份创建成功: "+resp.Filename)
	c.Redirect(http.StatusSeeOther, "/admin/backups")
}

// Download 下载备份文件。
func (h *Handler) Download(c *gin.Context) {
	id, ok := parseID(c.Param("id"))
	if !ok {
		render.Error(c, http.StatusBadRequest, "无效的备份 ID", nil)
		return
	}
	actor := middleware.CurrentUser(c)
	fullpath, filename, err := h.svc.ResolveDownloadFile(c.Request.Context(), actor, DownloadReq{ID: id})
	if err != nil {
		if IsNotFound(err) {
			render.Error(c, http.StatusNotFound, "备份记录不存在", err)
		} else {
			render.Error(c, http.StatusInternalServerError, "解析备份文件失败", err)
		}
		return
	}
	if _, statErr := os.Stat(fullpath); statErr != nil {
		render.Error(c, http.StatusNotFound, "备份文件不存在", statErr)
		return
	}
	c.FileAttachment(fullpath, filename)
}

func parseID(raw string) (uint64, bool) {
	id, err := strconv.ParseUint(strings.TrimSpace(raw), 10, 64)
	if err != nil || id == 0 {
		return 0, false
	}
	return id, true
}
