// =============================================================================
// 文件: internal/module/schedule/handler_demand_write.go
// 模块: 排期工作台
// 类型: action
// 职责: 业需与独立研发需求排期保存接口（JSON POST）。
// 依赖: internal/middleware
//
//	internal/pkg/zentao
//
// =============================================================================

package schedule

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"workbench/internal/middleware"
	"workbench/internal/pkg/zentao"
)

// SaveScheduling 保存排期一体化弹窗数据并同步禅道（JSON）。
func (h *Handler) SaveScheduling(c *gin.Context) {
	demandID, ok := parseDemandID(c)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "参数错误",
		})
		return
	}

	var req SaveSchedulingReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "参数错误",
		})
		return
	}
	if errs := req.Validate(); len(errs) > 0 {
		c.JSON(http.StatusUnprocessableEntity, gin.H{
			"success": false,
			"message": formatFieldErrors(errs),
			"errors":  errs,
		})
		return
	}

	actor := middleware.CurrentUser(c)
	if err := h.svc.SaveScheduling(c.Request.Context(), actor, demandID, &req); err != nil {
		// 业务前置校验拦截：零写入，前端弹警告框引导去禅道维护。
		var notice *ProductAccessNoticeError
		if errors.As(err, &notice) {
			for i := range notice.Products {
				notice.Products[i].ViewURL = zentao.ProductViewURLWithBase(h.zentaoURL, notice.Products[i].ID)
			}
			c.JSON(http.StatusOK, gin.H{
				"success":  false,
				"code":     "PRODUCT_ACCESS_NOTICE",
				"products": notice.Products,
			})
			return
		}
		var businessErr *SchedulingBusinessError
		if errors.As(err, &businessErr) {
			c.JSON(http.StatusUnprocessableEntity, gin.H{
				"success": false,
				"message": businessErr.Error(),
			})
			return
		}
		if h.logger != nil {
			h.logger.Error("save demand scheduling failed",
				zap.Error(err),
				zap.Uint("demand_id", demandID),
			)
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// SaveStoryScheduling 保存独立研发需求排期并同步禅道（JSON）。
func (h *Handler) SaveStoryScheduling(c *gin.Context) {
	storyID, ok := parseStoryID(c)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "参数错误",
		})
		return
	}

	var req SaveSchedulingReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "参数错误",
		})
		return
	}
	if errs := req.Validate(); len(errs) > 0 {
		c.JSON(http.StatusUnprocessableEntity, gin.H{
			"success": false,
			"message": formatFieldErrors(errs),
			"errors":  errs,
		})
		return
	}

	actor := middleware.CurrentUser(c)
	if err := h.svc.SaveStoryScheduling(c.Request.Context(), actor, storyID, &req); err != nil {
		// 业务前置校验拦截（PRODUCT_ACCESS_NOTICE）：零写入，前端弹警告引导去禅道。
		var notice *ProductAccessNoticeError
		if errors.As(err, &notice) {
			for i := range notice.Products {
				notice.Products[i].ViewURL = zentao.ProductViewURLWithBase(h.zentaoURL, notice.Products[i].ID)
			}
			c.JSON(http.StatusOK, gin.H{
				"success":  false,
				"code":     "PRODUCT_ACCESS_NOTICE",
				"products": notice.Products,
			})
			return
		}
		if h.logger != nil {
			h.logger.Error("save story scheduling failed",
				zap.Error(err),
				zap.Uint("story_id", storyID),
			)
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}
