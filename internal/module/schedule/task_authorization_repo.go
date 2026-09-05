package schedule

import (
	"context"
	"time"
)

type taskMutationRow struct {
	ID      uint   `gorm:"column:id"`
	Story   uint   `gorm:"column:story"`
	Deleted string `gorm:"column:deleted"`
}

// ValidateStoryForTaskMutation locks the parent story and verifies its live
// ZenTao row before a task is created, edited, or closed.
func (r *Repo) ValidateStoryForTaskMutation(ctx context.Context, storyID uint) error {
	if storyID == 0 {
		return &TaskMutationError{Code: taskMutationNotFound, Message: "研发需求不存在"}
	}
	var row struct {
		ID      uint   `gorm:"column:id"`
		Deleted string `gorm:"column:deleted"`
	}
	if err := r.db.WithContext(ctx).Raw(
		`SELECT id, deleted FROM zt_story WHERE id = ? FOR UPDATE`, storyID,
	).Scan(&row).Error; err != nil {
		return err
	}
	if row.ID == 0 || row.Deleted != "0" {
		return &TaskMutationError{Code: taskMutationNotFound, Message: "研发需求不存在"}
	}
	return nil
}

// ValidateTaskOwnership locks the task row and validates its database-owned
// story relationship. The client supplied story ID is never treated as fact.
func (r *Repo) ValidateTaskOwnership(ctx context.Context, taskID, storyID uint) error {
	if taskID == 0 {
		return &TaskMutationError{Code: taskMutationNotFound, Message: "任务不存在"}
	}
	var row taskMutationRow
	if err := r.db.WithContext(ctx).Raw(
		`SELECT id, story, deleted FROM zt_task WHERE id = ? FOR UPDATE`, taskID,
	).Scan(&row).Error; err != nil {
		return err
	}
	if row.ID == 0 || row.Deleted != "0" {
		return &TaskMutationError{Code: taskMutationNotFound, Message: "任务不存在"}
	}
	if row.Story != storyID {
		return &TaskMutationError{Code: taskMutationForbidden, Message: "任务不属于当前研发需求"}
	}
	return nil
}

// UpdateTaskForStory keeps the ownership predicate on the write as a second
// guard after the locked validation read.
func (r *Repo) UpdateTaskForStory(ctx context.Context, taskID, storyID uint, updates map[string]interface{}) error {
	if taskID == 0 || storyID == 0 {
		return &TaskMutationError{Code: taskMutationNotFound, Message: "任务不存在"}
	}
	if len(updates) == 0 {
		return nil
	}
	result := r.db.WithContext(ctx).
		Table("zt_task").
		Where("id = ? AND story = ? AND deleted = '0'", taskID, storyID).
		Updates(updates)
	if result.Error != nil || result.RowsAffected > 0 {
		return result.Error
	}

	// MySQL reports zero changed rows for a valid no-op update. Re-read the
	// locked row to distinguish that case from a stale or concurrently changed
	// ownership relationship.
	var row taskMutationRow
	if err := r.db.WithContext(ctx).Raw(
		`SELECT id, story, deleted FROM zt_task WHERE id = ? FOR UPDATE`, taskID,
	).Scan(&row).Error; err != nil {
		return err
	}
	if row.ID == 0 || row.Deleted != "0" {
		return &TaskMutationError{Code: taskMutationNotFound, Message: "任务不存在"}
	}
	if row.Story != storyID {
		return &TaskMutationError{Code: taskMutationConflict, Message: "任务归属已发生变化"}
	}
	return nil
}

func (r *Repo) CloseTaskForStory(ctx context.Context, taskID, storyID uint, actor string) error {
	return r.UpdateTaskForStory(ctx, taskID, storyID, map[string]interface{}{
		"status":       "closed",
		"closedBy":     actor,
		"closedDate":   time.Now(),
		"closedReason": "done",
		"assignedTo":   "closed",
	})
}
