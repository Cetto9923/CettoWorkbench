// =============================================================================
// 文件: internal/module/schedule/repo_window_milestone.go
// 模块: 排期工作台
// 类型: repo
// 职责: 实现版本窗口里程碑（zt_wb_versionwindow_milestone）数据库读写。
// 依赖: gorm, internal/model
// =============================================================================

package schedule

import (
	"context"
	"errors"

	"workbench/internal/model"
)

// WindowMilestoneRow 版本窗口基础里程碑查询结果。
type WindowMilestoneRow struct {
	WindowID     uint64 `gorm:"column:versionWindow"`
	PlanTestDone string `gorm:"column:planTestDone"`
	TestDone     string `gorm:"column:testDone"`
	AcceptDone   string `gorm:"column:acceptDone"`
}

// GetWindowMilestone 查询单个版本窗口的基础里程碑。
func (r *Repo) GetWindowMilestone(ctx context.Context, windowID uint64) (*WindowMilestoneRow, error) {
	if windowID == 0 {
		return nil, nil
	}
	const query = `
SELECT
  versionWindow,
  DATE_FORMAT(planTestDone, '%Y-%m-%d') AS planTestDone,
  DATE_FORMAT(testDone, '%Y-%m-%d') AS testDone,
  DATE_FORMAT(acceptDone, '%Y-%m-%d') AS acceptDone
FROM zt_wb_versionwindow_milestone
WHERE versionWindow = ? AND deletedAt IS NULL`

	var row WindowMilestoneRow
	if err := r.db.WithContext(ctx).Raw(query, windowID).Scan(&row).Error; err != nil {
		return nil, err
	}
	if row.WindowID == 0 {
		return nil, nil
	}
	return &row, nil
}

// GetWindowMilestones 批量查询版本窗口基础里程碑。
func (r *Repo) GetWindowMilestones(ctx context.Context, windowIDs []uint64) (map[uint64]WindowMilestoneRow, error) {
	out := make(map[uint64]WindowMilestoneRow)
	if len(windowIDs) == 0 {
		return out, nil
	}
	const query = `
SELECT
  versionWindow,
  DATE_FORMAT(planTestDone, '%Y-%m-%d') AS planTestDone,
  DATE_FORMAT(testDone, '%Y-%m-%d') AS testDone,
  DATE_FORMAT(acceptDone, '%Y-%m-%d') AS acceptDone
FROM zt_wb_versionwindow_milestone
WHERE versionWindow IN ? AND deletedAt IS NULL`

	var rows []WindowMilestoneRow
	if err := r.db.WithContext(ctx).Raw(query, windowIDs).Scan(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		if row.WindowID == 0 {
			continue
		}
		out[row.WindowID] = row
	}
	return out, nil
}

// UpsertWindowMilestone 保存版本窗口基础里程碑。
func (r *Repo) UpsertWindowMilestone(ctx context.Context, row *model.VersionWindowMilestone) error {
	if row == nil || row.WindowID == 0 {
		return errors.New("version window milestone is invalid")
	}
	const query = `
INSERT INTO zt_wb_versionwindow_milestone
  (versionWindow, planTestDone, testDone, acceptDone, createdBy, updatedBy, createdDate, updatedDate)
VALUES (?, ?, ?, ?, ?, ?, NOW(), NOW())
ON DUPLICATE KEY UPDATE
  planTestDone = VALUES(planTestDone),
  testDone = VALUES(testDone),
  acceptDone = VALUES(acceptDone),
  updatedBy = VALUES(updatedBy),
  updatedDate = NOW(),
  deletedAt = NULL`
	return r.db.WithContext(ctx).Exec(
		query,
		row.WindowID,
		row.PlanTestDone,
		row.TestDone,
		row.AcceptDone,
		row.CreatedBy,
		row.UpdatedBy,
	).Error
}
