// =============================================================================
// 文件: internal/module/build/handler.go
// 模块: 版本管理
// 类型: action
// 职责: 版本关联/解除研发需求 HTML 片段与写入接口。
// 依赖: internal/middleware
//       internal/pkg/errorx
//       internal/pkg/perm
//       internal/pkg/render
//       internal/constants
// =============================================================================

package build

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"workbench/internal/constants"
	"workbench/internal/middleware"
	"workbench/internal/pkg/errorx"
	"workbench/internal/pkg/perm"
	"workbench/internal/pkg/render"
)

// Handler 版本 HTTP。
type Handler struct {
	svc    *Service
	logger *zap.Logger
}

// NewHandler 创建 Handler。
func NewHandler(svc *Service, logger *zap.Logger) *Handler {
	return &Handler{svc: svc, logger: logger}
}

// RegisterRoutes 注册版本路由（挂载在已登录的根 group）。
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	g := rg.Group("/builds")

	g.GET("/:id/linkstory", middleware.RequirePerm(perm.BuildLinkStory), h.LinkStory)

	g.POST("/:id/linkstories", middleware.RequirePerm(perm.BuildLinkStory), h.LinkStories)
	g.POST("/:id/unlinkstories", middleware.RequirePerm(perm.BuildLinkStory), h.UnlinkStories)
}

// LinkStory GET /builds/:id/linkstory — 返回关联需求弹窗 body HTML 片段。
func (h *Handler) LinkStory(c *gin.Context) {
	id, err := parseUintParam(c.Param("id"))
	if err != nil || id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"message": "版本 ID 无效"})
		return
	}

	var req LinkStoryListReq
	if bindErr := c.ShouldBindQuery(&req); bindErr != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "参数解析失败"})
		return
	}

	resp, pager, svcErr := h.svc.LinkStory(c.Request.Context(), middleware.CurrentUser(c), id, req)
	if svcErr != nil {
		if h.logger != nil {
			h.logger.Error("build linkstory", zap.Error(svcErr), zap.Uint("buildId", id))
		}
		status, msg := linkStoryHTTPError(svcErr)
		c.JSON(status, gin.H{"message": msg})
		return
	}

	render.Fragment(c, http.StatusOK, constants.TEMPLATE_PO_LINKSTORY, "po/linkstory_body", gin.H{
		"BuildID":        resp.BuildID,
		"BaseUrl":        resp.BaseUrl,
		"Stories":        resp.Stories,
		"Pager":          pager,
		"PageSize":       resp.PageSize,
		"SearchForm":     resp.SearchForm,
		"SearchMetaJSON": SearchMetaJSON(resp.SearchForm),
		"QuerySuffix":    resp.QuerySuffix,
	})
}

// LinkStories POST /builds/:id/linkstories — 将勾选需求关联到版本（代理禅道 API）。
func (h *Handler) LinkStories(c *gin.Context) {
	id, err := parseUintParam(c.Param("id"))
	if err != nil || id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "版本 ID 无效"})
		return
	}

	var req LinkStoriesReq
	if bindErr := c.ShouldBindJSON(&req); bindErr != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "参数解析失败"})
		return
	}
	if fieldErrs := req.Validate(); len(fieldErrs) > 0 {
		c.JSON(http.StatusUnprocessableEntity, gin.H{
			"success": false,
			"message": "参数校验失败",
			"errors":  fieldErrs,
		})
		return
	}

	if svcErr := h.svc.LinkStories(c.Request.Context(), middleware.CurrentUser(c), id, req); svcErr != nil {
		if h.logger != nil {
			h.logger.Error("build linkstories", zap.Error(svcErr), zap.Uint("buildId", id))
		}
		status, msg := linkStoryHTTPError(svcErr)
		c.JSON(status, gin.H{"success": false, "message": msg})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":     true,
		"message":     "关联需求成功",
		"redirectUrl": "",
	})
}

// UnlinkStories POST /builds/:id/unlinkstories — 解除版本与研发需求关联（代理禅道 API）。
func (h *Handler) UnlinkStories(c *gin.Context) {
	id, err := parseUintParam(c.Param("id"))
	if err != nil || id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "版本 ID 无效"})
		return
	}

	var req LinkStoriesReq
	if bindErr := c.ShouldBindJSON(&req); bindErr != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "参数解析失败"})
		return
	}
	if fieldErrs := req.Validate(); len(fieldErrs) > 0 {
		c.JSON(http.StatusUnprocessableEntity, gin.H{
			"success": false,
			"message": "参数校验失败",
			"errors":  fieldErrs,
		})
		return
	}

	if svcErr := h.svc.UnlinkStories(c.Request.Context(), middleware.CurrentUser(c), id, req); svcErr != nil {
		if h.logger != nil {
			h.logger.Error("build unlinkstories", zap.Error(svcErr), zap.Uint("buildId", id))
		}
		status, msg := linkStoryHTTPError(svcErr)
		c.JSON(status, gin.H{"success": false, "message": msg})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":     true,
		"message":     "解除关联成功",
		"redirectUrl": "",
	})
}

func parseUintParam(raw string) (uint, error) {
	raw = strings.TrimSpace(raw)
	n, err := strconv.ParseUint(raw, 10, 64)
	if err != nil {
		return 0, err
	}
	return uint(n), nil
}

func linkStoryHTTPError(err error) (int, string) {
	if biz, ok := errorx.IsBizError(err); ok {
		switch biz.Code {
		case errorx.ErrCodeNotFound:
			return http.StatusNotFound, biz.Msg
		case errorx.ErrCodeForbidden:
			return http.StatusForbidden, biz.Msg
		case errorx.ErrCodeInvalidParam:
			return http.StatusBadRequest, biz.Msg
		}
		return http.StatusBadRequest, biz.Msg
	}
	return http.StatusInternalServerError, "请求失败"
}
