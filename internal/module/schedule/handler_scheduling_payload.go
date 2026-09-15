package schedule

import "github.com/gin-gonic/gin"

func newDemandSchedulingPayload() gin.H {
	return gin.H{
		"success":            true,
		"involvedProducts":   []ZtProductOption{},
		"productProjects":    gin.H{},
		"projectExecutions":  gin.H{},
		"stories":            []DemandSchedulingStoryItem{},
		"userStories":        []UserStoryItem{},
		"storyDefaults":      []DemandSchedulingStoryDefault{},
		"windowProductPlans": []SchedulingWindowProductPlan{},
		"productPlans":       gin.H{},
		"windows":            []SchedulingWindowOption{},
		"users":              []SchedulingUserOption{},
	}
}
