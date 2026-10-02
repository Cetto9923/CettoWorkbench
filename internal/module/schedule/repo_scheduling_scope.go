// =============================================================================
// 文件: internal/module/schedule/repo_scheduling_scope.go
// 模块: 排期工作台
// 类型: repo
// 职责: 批量读取排期写入对象归属。
// 依赖: gorm
// =============================================================================

package schedule

import (
	"context"
	"gorm.io/gorm/clause"
	"workbench/internal/model"
)

func (r *Repo) SchedulingStories(ctx context.Context, demandID, storyID uint) ([]ZtStory, error) {
	var rows []ZtStory
	err := r.db.WithContext(ctx).Raw(`SELECT id, product, fromDemand FROM zt_story
WHERE deleted = '0' AND ((? > 0 AND fromDemand = ?) OR (? = 0 AND id = ?))`, demandID, demandID, demandID, storyID).Scan(&rows).Error
	return rows, err
}

func (r *Repo) SchedulingTaskOwners(ctx context.Context, ids []uint) (map[uint]uint, error) {
	out := map[uint]uint{}
	if len(ids) == 0 {
		return out, nil
	}
	var rows []struct {
		ID    uint
		Story uint
	}
	if err := r.db.WithContext(ctx).Raw(`SELECT id, story FROM zt_task WHERE id IN ? AND deleted = '0'`, ids).Scan(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		out[row.ID] = row.Story
	}
	return out, nil
}

func (r *Repo) ExecutionMatchesProduct(ctx context.Context, executionID, productID uint) (bool, error) {
	projectID, err := r.GetProjectIDByExecution(ctx, executionID)
	if err != nil {
		return false, err
	}
	var count int64
	err = r.db.WithContext(ctx).Table("zt_projectproduct").Where("project = ? AND product = ?", projectID, productID).Count(&count).Error
	return count > 0, err
}

func (r *Repo) LockWindow(ctx context.Context, id uint64) error {
	var window model.VersionWindow
	return r.db.WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).First(&window, id).Error
}
