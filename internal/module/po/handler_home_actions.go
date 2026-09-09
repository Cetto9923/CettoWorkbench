package po

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"workbench/internal/middleware"
)

func (h *Handler) homeAction(c *gin.Context, fn func(uint, string) error, success string) {
	id, err := strconv.ParseUint(strings.TrimPrefix(strings.TrimPrefix(strings.TrimSpace(c.Param("id")), "US"), "us"), 10, 64)
	if err != nil || id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "需求 ID 无效"})
		return
	}
	var req DemandActionReq
	if c.Request.ContentLength != 0 {
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "参数解析失败"})
			return
		}
	}
	req.normalize()
	if err := fn(uint(id), req.Comment); err != nil {
		status := http.StatusInternalServerError
		message := "办理失败，请稍后重试"
		switch {
		case errors.Is(err, errHomeActionNotFound):
			status, message = http.StatusNotFound, "需求不存在"
		case errors.Is(err, errHomeActionForbidden):
			status, message = http.StatusForbidden, "无权办理该需求"
		case errors.Is(err, errHomeActionConflict):
			status, message = http.StatusConflict, err.Error()
		}
		c.JSON(status, gin.H{"success": false, "message": message})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": success, "redirectUrl": "/home"})
}

func (h *Handler) ClarifyHomeDemand(c *gin.Context) {
	h.homeAction(c, func(id uint, comment string) error {
		return h.svc.ClarifyHomeDemand(c.Request.Context(), middleware.CurrentUser(c), id, comment)
	}, "澄清成功")
}
func (h *Handler) AcceptHomeDemand(c *gin.Context) {
	h.homeAction(c, func(id uint, comment string) error {
		return h.svc.AcceptHomeDemand(c.Request.Context(), middleware.CurrentUser(c), id, comment)
	}, "验收成功")
}
func (h *Handler) DeliverHomeDemand(c *gin.Context) {
	h.homeAction(c, func(id uint, comment string) error {
		return h.svc.DeliverHomeDemand(c.Request.Context(), middleware.CurrentUser(c), id, comment)
	}, "发起交付成功")
}
func (h *Handler) UrgeHomeDemand(c *gin.Context) {
	h.homeAction(c, func(id uint, comment string) error {
		return h.svc.UrgeHomeDemand(c.Request.Context(), middleware.CurrentUser(c), id, comment)
	}, "已发起催办")
}
