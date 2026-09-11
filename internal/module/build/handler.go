// =============================================================================
// 文件: internal/module/build/handler.go
// 模块: 版本管理
// 类型: action
// 职责: 版本关联研发需求 HTML 片段接口。
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
		"BuildID":  resp.BuildID,
		"BaseUrl":  resp.BaseUrl,
		"Stories":  resp.Stories,
		"Pager":    pager,
		"PageSize": resp.PageSize,
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
