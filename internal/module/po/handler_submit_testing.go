package po

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"workbench/internal/constants"
	"workbench/internal/middleware"
	"workbench/internal/pkg/errorx"
	"workbench/internal/pkg/render"
)

func (h *Handler) SubmitTestView(c *gin.Context) {
	demandID := (&DemandDetailReq{ID: c.Param("id")}).ExtractDemandID()
	if demandID == 0 {
		render.Error(c, http.StatusBadRequest, "需求 ID 无效", nil)
		return
	}
	page, err := h.svc.SubmitTestPage(c.Request.Context(), middleware.CurrentUser(c), demandID)
	if err != nil {
		handleSubmitTestPageError(c, h, err)
		return
	}
	render.Page(c, http.StatusOK, constants.TEMPLATE_PO_SUBMIT_TEST, gin.H{
		"Title":       "提测办理",
		"PageTitle":   "提测办理",
		"CurrentPath": "/demands/submit-test",
		"SubmitTest":  page,
	})
}

func handleSubmitTestPageError(c *gin.Context, h *Handler, err error) {
	status, message := submitTestErrorStatus(err)
	if status == http.StatusInternalServerError && h != nil && h.logger != nil {
		h.logger.Error("po submit-test page", zap.Error(err))
	}
	render.Error(c, status, message, err)
}

func submitTestErrorStatus(err error) (int, string) {
	if bizErr, ok := errorx.IsBizError(err); ok {
		switch bizErr.Code {
		case errorx.ErrCodeNotFound:
			return http.StatusNotFound, "业务需求不存在"
		case errorx.ErrCodeForbidden:
			return http.StatusForbidden, "无权办理该需求的提测"
		case errorx.ErrCodeInvalidParam:
			return http.StatusBadRequest, "需求 ID 无效"
		}
	}
	switch {
	case errors.Is(err, errSubmitTestNotFound):
		return http.StatusNotFound, "业务需求不存在"
	case errors.Is(err, errSubmitTestForbidden):
		return http.StatusForbidden, "无权办理该需求的提测"
	default:
		return http.StatusInternalServerError, "提测办理失败"
	}
}
