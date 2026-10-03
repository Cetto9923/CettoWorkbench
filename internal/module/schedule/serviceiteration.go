// =============================================================================
// 文件: internal/module/schedule/serviceiteration.go
// 模块: 排期工作台
// 类型: action
// 职责: 校验对象归属后调用禅道创建迭代，关联产品计划。
// 依赖: internal/model
// =============================================================================
package schedule

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"workbench/internal/model"
	"workbench/internal/pkg/errorx"
)

func (s *Service) CreateIteration(ctx context.Context, actor *model.User, req CreateIterationReq) (uint, error) {
	if err := s.RequireStoryWriteAccess(ctx, actor, req.StoryID); err != nil {
		return 0, err
	}
	if fields := req.Validate(); len(fields) > 0 {
		return 0, errors.New(formatFieldErrors(fields))
	}
	story, err := s.repo.GetStoryTaskDetail(ctx, req.StoryID)
	if err != nil {
		return 0, err
	}
	products, err := s.schedulingProductAccess(ctx, actor)
	if err != nil {
		return 0, err
	}
	if products != nil && !products[story.ProductID] {
		return 0, errorx.New(errorx.ErrCodeForbidden, "无权操作该产品")
	}
	projects, err := s.repo.GetProductProjects(ctx, story.ProductID)
	if err != nil {
		return 0, err
	}
	allowed := false
	for _, project := range projects {
		allowed = allowed || project.ID == req.ProjectID
	}
	if !allowed {
		return 0, errorx.New(errorx.ErrCodeForbidden, "项目不属于该产品或不可创建迭代")
	}
	plans, err := s.repo.IterationPlans(ctx, story.ProductID)
	if err != nil {
		return 0, err
	}
	body, err := iterationBody(req, story.ProductID, actor.Account, plans, time.Now())
	if err != nil {
		return 0, err
	}
	var out struct {
		ID uint `json:"id"`
	}
	path := fmt.Sprintf("/projects/%d/executions", req.ProjectID)
	if err := s.ztAPI.Do(ctx, http.MethodPost, path, body, &out); err != nil {
		return 0, fmt.Errorf("创建迭代失败，请刷新核对禅道状态：%w", err)
	}
	if out.ID == 0 {
		return 0, errors.New("禅道未返回迭代 ID，请刷新核对")
	}
	return out.ID, nil
}

func iterationBody(req CreateIterationReq, productID uint, account string, plans []MatchingPlanItem, now time.Time) (map[string]any, error) {
	begin := now.In(time.FixedZone("Asia/Shanghai", 8*3600)).Format("2006-01-02")
	start, _ := time.Parse("2006-01-02", begin)
	days := 14
	if req.Period == "4w" {
		days = 28
	}
	end := start.AddDate(0, 0, days-1).Format("2006-01-02")
	linked := [][]uint{}
	if req.PlanID > 0 {
		var plan *MatchingPlanItem
		for i := range plans {
			if plans[i].ID == req.PlanID {
				plan = &plans[i]
			}
		}
		if plan == nil {
			return nil, errorx.New(errorx.ErrCodeForbidden, "计划不属于该产品或已关闭")
		}
		if req.Period == "plan" {
			begin, end = plan.Begin, plan.End
		}
		// Main 原生入口直接下标访问 plans，JSON 对象会被解析为 stdClass。
		linked = make([][]uint, productID+1)
		for i := range linked {
			linked[i] = []uint{}
		}
		linked[productID] = []uint{req.PlanID}
	}
	return map[string]any{"name": begin + " - " + end, "begin": begin, "end": end,
		"products": []uint{productID}, "plans": linked, "PM": account, "lifetime": "short",
		"acl": "private", "percent": 100, "code": fmt.Sprintf("WB-%d-%d", req.StoryID, now.UnixNano())}, nil
}
