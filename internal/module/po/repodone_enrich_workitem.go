package po

import (
	"context"
	"fmt"
)

// doneWorkContextRow 业务需求 / 研发需求共用的上下文行：投影列一致，仅标题列名不同。
type doneWorkContextRow struct {
	ID          int64  `gorm:"column:id"`
	Name        string `gorm:"column:name"`
	Title       string `gorm:"column:title"`
	Status      string `gorm:"column:status"`
	AssignedTo  string `gorm:"column:assignedTo"`
	ProductName string `gorm:"column:product_name"`
	PoolName    string `gorm:"column:pool_name"`
}

// doneTaskBugContextRow 任务 / Bug 共用的上下文行；PoolName 只有任务会回填。
type doneTaskBugContextRow struct {
	ID            int64  `gorm:"column:id"`
	Name          string `gorm:"column:name"`
	Title         string `gorm:"column:title"`
	Status        string `gorm:"column:status"`
	Project       int64  `gorm:"column:project"`
	Execution     int64  `gorm:"column:execution"`
	AssignedTo    string `gorm:"column:assignedTo"`
	ProjectName   string `gorm:"column:project_name"`
	ExecutionName string `gorm:"column:execution_name"`
	PoolName      string `gorm:"column:pool_name"`
}

// loadDemandStoryContexts 加载业务需求与研发需求上下文：研需的需求池须经 fromDemand 回填。
func (r *Repo) loadDemandStoryContexts(ctx context.Context, out map[string]doneObjectContext, demandIDs, storyIDs []int64) {
	if len(demandIDs) > 0 {
		var list []doneWorkContextRow
		_ = r.db.WithContext(ctx).Table("zt_demand AS d").
			Select("d.id, d.name, d.status, d.assignedTo, COALESCE(p.name, '') AS product_name, COALESCE(dp.name, '') AS pool_name").
			Joins("LEFT JOIN zt_product AS p ON p.id = d.product").
			Joins("LEFT JOIN zt_demandpool AS dp ON dp.id = d.pool AND dp.deleted = '0'").
			Where("d.id IN ?", demandIDs).
			Scan(&list).Error
		for _, d := range list {
			out[fmt.Sprintf("demand:%d", d.ID)] = doneObjectContext{
				Title:        d.Name,
				Status:       d.Status,
				ProductName:  d.ProductName,
				PoolName:     d.PoolName,
				CurrentOwner: d.AssignedTo,
			}
		}
	}
	if len(storyIDs) > 0 {
		var list []doneWorkContextRow
		_ = r.db.WithContext(ctx).Table("zt_story AS s").
			Select("s.id, s.title, s.status, s.assignedTo, COALESCE(p.name, '') AS product_name, COALESCE(dp.name, '') AS pool_name").
			Joins("LEFT JOIN zt_product AS p ON p.id = s.product").
			Joins("LEFT JOIN zt_demand AS d ON d.id = s.fromDemand AND d.deleted = '0'").
			Joins("LEFT JOIN zt_demandpool AS dp ON dp.id = d.pool AND dp.deleted = '0'").
			Where("s.id IN ?", storyIDs).
			Scan(&list).Error
		for _, s := range list {
			out[fmt.Sprintf("story:%d", s.ID)] = doneObjectContext{
				Title:        s.Title,
				Status:       s.Status,
				ProductName:  s.ProductName,
				PoolName:     s.PoolName,
				CurrentOwner: s.AssignedTo,
			}
		}
	}
}

// loadTaskBugContexts 加载任务与 Bug 上下文：任务的需求池须经 story → fromDemand 回填，Bug 不挂需求池。
func (r *Repo) loadTaskBugContexts(ctx context.Context, out map[string]doneObjectContext, taskIDs, bugIDs []int64) {
	if len(taskIDs) > 0 {
		var list []doneTaskBugContextRow
		_ = r.db.WithContext(ctx).Table("zt_task AS t").
			Select("t.id, t.name, t.status, t.project, t.execution, t.assignedTo, COALESCE(p.name, '') AS project_name, COALESCE(e.name, '') AS execution_name, COALESCE(dp.name, '') AS pool_name").
			Joins("LEFT JOIN zt_project AS p ON p.id = t.project").
			Joins("LEFT JOIN zt_project AS e ON e.id = t.execution").
			Joins("LEFT JOIN zt_story AS s ON s.id = t.story AND s.deleted = '0'").
			Joins("LEFT JOIN zt_demand AS d ON d.id = s.fromDemand AND d.deleted = '0'").
			Joins("LEFT JOIN zt_demandpool AS dp ON dp.id = d.pool AND dp.deleted = '0'").
			Where("t.id IN ?", taskIDs).
			Scan(&list).Error
		for _, t := range list {
			out[fmt.Sprintf("task:%d", t.ID)] = doneObjectContext{
				Title:         t.Name,
				Status:        t.Status,
				ProjectID:     t.Project,
				ProjectName:   t.ProjectName,
				ExecutionID:   t.Execution,
				ExecutionName: t.ExecutionName,
				PoolName:      t.PoolName,
				CurrentOwner:  t.AssignedTo,
			}
		}
	}
	if len(bugIDs) > 0 {
		var list []doneTaskBugContextRow
		_ = r.db.WithContext(ctx).Table("zt_bug AS b").
			Select("b.id, b.title, b.status, b.project, b.execution, b.assignedTo, COALESCE(p.name, '') AS project_name, COALESCE(e.name, '') AS execution_name").
			Joins("LEFT JOIN zt_project AS p ON p.id = b.project").
			Joins("LEFT JOIN zt_project AS e ON e.id = b.execution").
			Where("b.id IN ?", bugIDs).
			Scan(&list).Error
		for _, b := range list {
			out[fmt.Sprintf("bug:%d", b.ID)] = doneObjectContext{
				Title:         b.Title,
				Status:        b.Status,
				ProjectID:     b.Project,
				ProjectName:   b.ProjectName,
				ExecutionID:   b.Execution,
				ExecutionName: b.ExecutionName,
				CurrentOwner:  b.AssignedTo,
			}
		}
	}
}

// loadTodoContexts 加载待办上下文：待办无归属信息，只取标题与状态。
func (r *Repo) loadTodoContexts(ctx context.Context, out map[string]doneObjectContext, todoIDs []int64) {
	if len(todoIDs) == 0 {
		return
	}
	type tdRow struct {
		ID     int64  `gorm:"column:id"`
		Name   string `gorm:"column:name"`
		Status string `gorm:"column:status"`
	}
	var list []tdRow
	_ = r.db.WithContext(ctx).Table("zt_todo").
		Select("id, name, status").
		Where("id IN ?", todoIDs).
		Scan(&list).Error
	for _, td := range list {
		out[fmt.Sprintf("todo:%d", td.ID)] = doneObjectContext{
			Title:  td.Name,
			Status: td.Status,
		}
	}
}
