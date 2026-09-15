// =============================================================================
// 文件: internal/module/testtask/handler.go
// 模块: 提测办理
// 类型: action
// 职责: 提测上下文、产品执行列表与创建版本 HTTP 接口。
// 依赖: internal/pkg/errorx
// =============================================================================

package testtask

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"workbench/internal/middleware"
	"workbench/internal/pkg/errorx"
	"workbench/internal/pkg/perm"
)

// Handler 提测办理 HTTP。
type Handler struct {
	svc    *Service
	logger *zap.Logger
}

// NewHandler 创建 Handler。
func NewHandler(svc *Service, logger *zap.Logger) *Handler {
	return &Handler{svc: svc, logger: logger}
}

// RegisterRoutes 注册提测路由（挂载在已登录的根 group）。
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	g := rg.Group("")
	g.GET("/demands/:id/testtask", middleware.RequirePerm(perm.PoHomeList), h.GetContext)
	g.GET("/products/:id/executions", middleware.RequirePerm(perm.PoHomeList), h.ListProductExecutions)
	g.GET("/products/:id/builds", middleware.RequirePerm(perm.PoHomeList), h.ListProductBuilds)

	g.POST("/demands/:id/testtask/builds", middleware.RequirePerm(perm.PoHomeList), h.CreateBuilds)
	g.POST("/demands/:id/testtask/tasks", middleware.RequirePerm(perm.PoHomeList), h.CreateTesttasks)
}

// ListProductBuilds GET /products/:id/builds。
func (h *Handler) ListProductBuilds(c *gin.Context) {
	id, err := parseUintParam(c.Param("id"))
	if err != nil || id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"message": "产品 ID 无效"})
		return
	}
	items, svcErr := h.svc.ListProductBuilds(c.Request.Context(), middleware.CurrentUser(c), id)
	if svcErr != nil {
		if h.logger != nil {
			h.logger.Error("testtask product builds", zap.Error(svcErr), zap.Uint("productId", id))
		}
		status, msg := contextHTTPError(svcErr)
		c.JSON(status, gin.H{"success": false, "message": msg})
		return
	}
	if items == nil {
		items = []BuildOption{}
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": items})
}

// GetContext GET /demands/:id/testtask — 返回当前需求上下文 JSON。
func (h *Handler) GetContext(c *gin.Context) {
	id, err := parseDemandID(c.Param("id"))
	if err != nil || id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"message": "需求 ID 无效"})
		return
	}

	resp, svcErr := h.svc.GetContext(c.Request.Context(), middleware.CurrentUser(c), id)
	if svcErr != nil {
		if h.logger != nil {
			h.logger.Error("testtask context", zap.Error(svcErr), zap.Uint("id", id))
		}
		status, msg := contextHTTPError(svcErr)
		c.JSON(status, gin.H{"message": msg})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    resp,
	})
}

// ListProductExecutions GET /products/:id/executions — 产品下所属执行下拉。
func (h *Handler) ListProductExecutions(c *gin.Context) {
	id, err := parseUintParam(c.Param("id"))
	if err != nil || id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"message": "产品 ID 无效"})
		return
	}

	list, svcErr := h.svc.ListProductExecutions(c.Request.Context(), middleware.CurrentUser(c), id)
	if svcErr != nil {
		if h.logger != nil {
			h.logger.Error("testtask product executions", zap.Error(svcErr), zap.Uint("productId", id))
		}
		status, msg := contextHTTPError(svcErr)
		c.JSON(status, gin.H{"message": msg})
		return
	}
	if list == nil {
		list = []ExecutionOption{}
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    list,
	})
}

// CreateBuilds POST /demands/:id/testtask/builds — 将创建新版本同步到禅道。
func (h *Handler) CreateBuilds(c *gin.Context) {
	id, err := parseDemandID(c.Param("id"))
	if err != nil || id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"message": "需求 ID 无效"})
		return
	}

	var req CreateBuildsReq
	if bindErr := c.ShouldBindJSON(&req); bindErr != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "参数解析失败"})
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

	resp, svcErr := h.svc.CreateBuilds(c.Request.Context(), middleware.CurrentUser(c), id, req)
	if svcErr != nil {
		if h.logger != nil {
			h.logger.Error("testtask create builds", zap.Error(svcErr), zap.Uint("id", id))
		}
		status, msg := contextHTTPError(svcErr)
		c.JSON(status, gin.H{"success": false, "message": msg})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "版本保存成功",
		"data":    resp,
	})
}

// CreateTesttasks POST /demands/:id/testtask/tasks — 将非联调测试单同步到禅道。
//
// 区分两类失败：
//   - 业务校验失败（参数、归属校验）：返回 400 + message，不进入远程写入。
//   - 业务规则通过但部分条目远程失败：Service 返回 *PartialTesttaskError，
//     Handler 通过 errors.As 取回已成功 ID + 失败原因，HTTP 207 + succeeded + cause；
//     前端必须显式提示并禁止自动重发（避免重复创建）。
func (h *Handler) CreateTesttasks(c *gin.Context) {
	id, err := parseDemandID(c.Param("id"))
	if err != nil || id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"message": "需求 ID 无效"})
		return
	}

	var req CreateTesttasksReq
	if bindErr := c.ShouldBindJSON(&req); bindErr != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "参数解析失败"})
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

	resp, svcErr := h.svc.CreateTesttasks(c.Request.Context(), middleware.CurrentUser(c), id, req)
	if svcErr != nil {
		var partial *PartialTesttaskError
		if errors.As(svcErr, &partial) {
			if h.logger != nil {
				h.logger.Warn("testtask create partial",
					zap.Error(svcErr),
					zap.Uint("id", id),
					zap.Int("succeeded", len(partial.Succeeded)),
					zap.String("cause", partial.Cause),
				)
			}
			if resp == nil {
				resp = &CreateTesttasksResp{Tasks: partial.Succeeded}
			} else if resp.Tasks == nil {
				resp.Tasks = partial.Succeeded
			}
			tasks := resp.Tasks
			if tasks == nil {
				tasks = []CreateTesttaskResult{}
			}
			c.JSON(http.StatusMultiStatus, gin.H{
				"success":   false,
				"message":   partial.Cause,
				"cause":     partial.Cause,
				"succeeded": tasks,
				"data":      gin.H{"tasks": tasks},
			})
			return
		}
		if h.logger != nil {
			h.logger.Error("testtask create tasks", zap.Error(svcErr), zap.Uint("id", id))
		}
		status, msg := contextHTTPError(svcErr)
		c.JSON(status, gin.H{"success": false, "message": msg})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "测试单保存成功",
		"data":    resp,
	})
}

func parseDemandID(raw string) (uint, error) {
	raw = strings.TrimSpace(raw)
	if len(raw) >= 2 && strings.EqualFold(raw[:2], "US") {
		raw = raw[2:]
	}
	if strings.HasPrefix(raw, "#") {
		raw = strings.TrimPrefix(raw, "#")
	}
	return parseUintParam(raw)
}

func parseUintParam(raw string) (uint, error) {
	raw = strings.TrimSpace(raw)
	n, err := strconv.ParseUint(raw, 10, 64)
	if err != nil {
		return 0, err
	}
	return uint(n), nil
}

func contextHTTPError(err error) (int, string) {
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
