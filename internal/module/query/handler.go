package query

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"workbench/internal/constants"
	"workbench/internal/middleware"
	"workbench/internal/pkg/perm"
	"workbench/internal/pkg/render"
)

type Handler struct {
	renderer *render.Renderer
	svc      *Service
	logger   *zap.Logger
}

func NewHandler(renderer *render.Renderer, svc *Service, logger *zap.Logger) *Handler {
	return &Handler{renderer: renderer, svc: svc, logger: logger}
}

func NewFromDB(renderer *render.Renderer, db *gorm.DB, logger *zap.Logger) *Handler {
	return NewHandler(renderer, NewService(NewRepo(db)), logger)
}

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	g := rg.Group("/query")
	g.Use(middleware.ActiveNav("/query"))
	g.GET("", middleware.RequirePerm(perm.ScheduleList), h.Index)
	g.GET("/items", middleware.RequirePerm(perm.ScheduleList), h.Items)
}

func (h *Handler) Index(c *gin.Context) {
	req := reqFromContext(c)
	resp, err := h.svc.List(c.Request.Context(), req)
	if err != nil {
		if h.logger != nil {
			h.logger.Warn("load demand query failed", zap.Error(err))
		}
		resp = ListResp{Kind: req.Tab, Rows: []Row{}}
	}
	render.Page(c, http.StatusOK, constants.TEMPLATE_QUERY_INDEX, gin.H{
		"Title":     "需求查询",
		"PageTitle": "需求查询",
		"Tab":       resp.Kind,
		"Keyword":   req.Keyword,
		"Status":    req.Status,
		"Priority":  req.Priority,
		"Owner":     req.Owner,
		"System":    req.System,
		"Rows":      resp.Rows,
		"Total":     resp.Total,
	})
}

// Items 返回 JSON：保持 {rows, total} 形状，并回显 {kind, page, pageSize}。
func (h *Handler) Items(c *gin.Context) {
	resp, err := h.svc.List(c.Request.Context(), reqFromContext(c))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "需求查询暂不可用"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"kind":     resp.Kind,
		"rows":     resp.Rows,
		"total":    resp.Total,
		"page":     resp.Page,
		"pageSize": resp.PageSize,
	})
}

func reqFromContext(c *gin.Context) ListReq {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("pageSize", "15"))
	r := ListReq{
		Tab:      c.DefaultQuery("tab", "biz"),
		Keyword:  c.Query("keyword"),
		Status:   c.Query("status"),
		Priority: c.Query("priority"),
		Owner:    c.Query("owner"),
		System:   c.Query("system"),
		Stage:    c.Query("stage"),
		Page:     page,
		PageSize: size,
	}
	r.Normalize()
	return r
}
