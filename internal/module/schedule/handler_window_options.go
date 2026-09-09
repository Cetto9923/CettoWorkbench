package schedule

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"workbench/internal/middleware"
)

// WindowOptions loads the current user's window form choices on demand.
func (h *Handler) WindowOptions(c *gin.Context) {
	data, err := h.svc.GetCreateWindowFormData(c.Request.Context(), middleware.CurrentUser(c))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "加载窗口选项失败"})
		return
	}
	products := make([]gin.H, 0, len(data.Products))
	for _, product := range data.Products {
		products = append(products, gin.H{"id": product.ID, "name": product.Name})
	}
	groups := make([]gin.H, 0, len(data.Teamgroups))
	for _, group := range data.Teamgroups {
		groups = append(groups, gin.H{"id": group.ID, "name": group.DisplayName})
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "products": products, "teamgroups": groups})
}
