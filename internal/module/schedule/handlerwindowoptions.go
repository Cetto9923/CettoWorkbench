// =============================================================================
// 文件: internal/module/schedule/handlerwindowoptions.go
// 模块: 排期工作台
// 类型: action
// 职责: 首页内联排期「新建版本窗口」按需加载产品与敏捷小组下拉选项
// 依赖: internal/middleware
// =============================================================================

package schedule

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"workbench/internal/middleware"
)

// windowOptionItem 下拉选项（id + 展示名）。
type windowOptionItem struct {
	ID   uint   `json:"id"`
	Name string `json:"name"`
}

// WindowOptions 返回当前用户新建版本窗口可选的产品与敏捷小组。
// GET /schedule/window-options；数据源复用 Main 的 GetCreateWindowFormData，不另起查询口径。
func (h *Handler) WindowOptions(c *gin.Context) {
	data, err := h.svc.GetCreateWindowFormData(c.Request.Context(), middleware.CurrentUser(c))
	if err != nil {
		if h.logger != nil {
			h.logger.Error("schedule window options", zap.Error(err))
		}
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "加载窗口选项失败"})
		return
	}
	products := make([]windowOptionItem, 0, len(data.Products))
	for _, product := range data.Products {
		products = append(products, windowOptionItem{ID: product.ID, Name: product.Name})
	}
	groups := make([]windowOptionItem, 0, len(data.Teamgroups))
	for _, group := range data.Teamgroups {
		groups = append(groups, windowOptionItem{ID: group.ID, Name: group.DisplayName})
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "products": products, "teamgroups": groups})
}
