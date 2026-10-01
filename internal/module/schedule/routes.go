// =============================================================================
// 文件: internal/module/schedule/routes.go
// 模块: 排期工作台
// 类型: action
// 职责: 排期工作台路由注册。写路径统一绑定 RequirePerm，读路径绑定已有 ScheduleList 权限。
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
		g.GET("", middleware.RequirePerm(perm.ScheduleList), h.Index)
		g.GET("/filter-counts", middleware.RequirePerm(perm.ScheduleList), h.GetFilterCounts)
		g.GET("/matching-plans", middleware.RequirePerm(perm.ScheduleList), h.GetMatchingPlans)
		g.GET("/demands/:id/scheduling", middleware.RequirePerm(perm.ScheduleList), h.GetDemandScheduling)
		g.GET("/demands/:id/review-to-story-notice", middleware.RequirePerm(perm.ScheduleList), h.CheckReviewToStoryNotice)
		g.POST("/demands/:id/save-scheduling", middleware.RequirePerm(perm.PoSchedule), h.SaveScheduling)
		g.GET("/stories/:id/scheduling", middleware.RequirePerm(perm.ScheduleList), h.GetStoryScheduling)
		g.POST("/stories/:id/save-scheduling", middleware.RequirePerm(perm.PoSchedule), h.SaveStoryScheduling)
		g.GET("/products/:id/projects", middleware.RequirePerm(perm.ScheduleList), h.GetProductProjects)
		g.GET("/projects/:id/executions", middleware.RequirePerm(perm.ScheduleList), h.GetProjectExecutions)
		g.GET("/stories/:id/tasks", middleware.RequirePerm(perm.ScheduleList), h.GetStoryTasks)
		g.POST("/stories/:id/save-tasks", middleware.RequirePerm(perm.PoSchedule), h.SaveStoryTasks)
		g.POST("/windows", middleware.RequirePerm(perm.PoSchedule), h.CreateWindow)
		g.GET("/windows", middleware.RequirePerm(perm.ScheduleList), h.ListWindows)
		g.GET("/windows/:id", middleware.RequirePerm(perm.ScheduleList), h.GetWindow)
		g.PUT("/windows/:id", middleware.RequirePerm(perm.PoSchedule), h.UpdateWindow)
		g.DELETE("/windows/:id", middleware.RequirePerm(perm.PoSchedule), h.DeleteWindow)
		g.GET("/window-options", middleware.RequirePerm(perm.ScheduleList), h.WindowOptions)
	}
}
