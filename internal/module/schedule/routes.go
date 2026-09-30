// =============================================================================
// 文件: internal/module/schedule/routes.go
// 模块: 排期工作台
// 类型: action
// 职责: 排期工作台路由注册。写路径统一绑定 RequirePerm，读路径沿用既有放行口径。
//       单独成文件，避免让已超 500 行的 handler.go 继续变长。
// 依赖: internal/middleware
//       internal/pkg/perm
// =============================================================================

package schedule

import (
	"github.com/gin-gonic/gin"

	"workbench/internal/middleware"
	"workbench/internal/pkg/perm"
)

// RegisterRoutes 注册排期工作台路由（挂载在已配置登录与操作日志的中间件组上）。
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	g := rg.Group("/schedule")
	g.Use(middleware.ActiveNav("/schedule"))
	{
		g.GET("", h.Index)
		g.GET("/filter-counts", h.GetFilterCounts)
		g.GET("/matching-plans", h.GetMatchingPlans)
		g.GET("/demands/:id/scheduling", h.GetDemandScheduling)
		g.GET("/demands/:id/review-to-story-notice", h.CheckReviewToStoryNotice)
		g.POST("/demands/:id/save-scheduling", middleware.RequirePerm(perm.PoSchedule), h.SaveScheduling)
		g.GET("/stories/:id/scheduling", h.GetStoryScheduling)
		g.POST("/stories/:id/save-scheduling", middleware.RequirePerm(perm.PoSchedule), h.SaveStoryScheduling)
		g.GET("/products/:id/projects", h.GetProductProjects)
		g.GET("/projects/:id/executions", h.GetProjectExecutions)
		g.GET("/stories/:id/tasks", h.GetStoryTasks)
		g.POST("/stories/:id/save-tasks", middleware.RequirePerm(perm.PoSchedule), h.SaveStoryTasks)
		g.POST("/windows", middleware.RequirePerm(perm.PoSchedule), h.CreateWindow)
		g.GET("/windows", h.ListWindows)
		g.GET("/windows/:id", h.GetWindow)
		g.PUT("/windows/:id", middleware.RequirePerm(perm.PoSchedule), h.UpdateWindow)
		g.DELETE("/windows/:id", middleware.RequirePerm(perm.PoSchedule), h.DeleteWindow)
		g.GET("/window-options", h.WindowOptions)
	}
}
