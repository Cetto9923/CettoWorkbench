// =============================================================================
// 文件: internal/module/schedule/repo_history.go
// 模块: 排期工作台
// 类型: action
// 职责: 读取业务需求排期、研发需求或任务编辑前字段，并把字段差异交给 pkg/ztaction 写入禅道历史。
// 依赖: internal/pkg/ztaction
// =============================================================================

package schedule

import (
	"context"
	"errors"
	"strings"

	"workbench/internal/pkg/ztaction"
)

type storyHistorySnapshot struct {
	ID             uint   `gorm:"column:id"`
	Title          string `gorm:"column:title"`
	AssignedTo     string `gorm:"column:assignedTo"`
	Product        uint   `gorm:"column:product"`
	EstimateLaunch string `gorm:"column:estimateLaunch"`
	DevelopFinish  string `gorm:"column:developFinish"`
	TestFinish     string `gorm:"column:testFinish"`
	VerifyFinish   string `gorm:"column:verifyFinish"`
}

// FindStoryHistorySnapshot 读取编辑前的研发需求字段，供对比变更。
func (r *Repo) FindStoryHistorySnapshot(ctx context.Context, storyID uint) (*storyHistorySnapshot, error) {
	if storyID == 0 {
		return nil, errors.New("story id is invalid")
	}
	var row storyHistorySnapshot
	err := r.db.WithContext(ctx).Raw(`
SELECT
  id,
  title,
  assignedTo,
  product,
  IFNULL(DATE_FORMAT(estimateLaunch, '%Y-%m-%d'), '') AS estimateLaunch,
  IFNULL(DATE_FORMAT(developFinish, '%Y-%m-%d'), '') AS developFinish,
  IFNULL(DATE_FORMAT(testFinish, '%Y-%m-%d'), '') AS testFinish,
  IFNULL(DATE_FORMAT(verifyFinish, '%Y-%m-%d'), '') AS verifyFinish
FROM zt_story
WHERE id = ? AND deleted = '0'
LIMIT 1`, storyID).Scan(&row).Error
	if err != nil {
		return nil, err
	}
	if row.ID == 0 {
		return nil, errors.New("研发需求不存在")
	}
	return &row, nil
}

type taskHistorySnapshot struct {
	ID         uint    `gorm:"column:id"`
	Name       string  `gorm:"column:name"`
	Type       string  `gorm:"column:type"`
	Pri        int     `gorm:"column:pri"`
	AssignedTo string  `gorm:"column:assignedTo"`
	Estimate   float64 `gorm:"column:estimate"`
	Left       float64 `gorm:"column:leftHours"`
	EstStarted string  `gorm:"column:estStarted"`
	Deadline   string  `gorm:"column:deadline"`
	Execution  uint    `gorm:"column:execution"`
	Project    uint    `gorm:"column:project"`
}

// FindTaskHistorySnapshot 读取编辑前的任务字段，供对比变更。
func (r *Repo) FindTaskHistorySnapshot(ctx context.Context, taskID uint) (*taskHistorySnapshot, error) {
	if taskID == 0 {
		return nil, errors.New("task id is invalid")
	}
	var row taskHistorySnapshot
	err := r.db.WithContext(ctx).Raw(`
SELECT
  id,
  name,
  type,
  pri,
  assignedTo,
  estimate,
  `+"`left`"+` AS leftHours,
  IFNULL(DATE_FORMAT(estStarted, '%Y-%m-%d'), '') AS estStarted,
  IFNULL(DATE_FORMAT(deadline, '%Y-%m-%d'), '') AS deadline,
  execution,
  project
FROM zt_task
WHERE id = ? AND deleted = '0'
LIMIT 1`, taskID).Scan(&row).Error
	if err != nil {
		return nil, err
	}
	if row.ID == 0 {
		return nil, errors.New("任务不存在")
	}
	return &row, nil
}

type demandSchedulingSnapshot struct {
	ID             uint   `gorm:"column:id"`
	Product        string `gorm:"column:product"`
	QD             string `gorm:"column:QD"`
	RD             string `gorm:"column:RD"`
	EstimateLaunch string `gorm:"column:estimateLaunch"`
	DevelopFinish  string `gorm:"column:developFinish"`
	TestFinish     string `gorm:"column:testFinish"`
	VerifyFinish   string `gorm:"column:verifyFinish"`
}

// FindDemandSchedulingSnapshot 读取业需排期字段的旧值，供对比变更。
func (r *Repo) FindDemandSchedulingSnapshot(ctx context.Context, demandID uint) (*demandSchedulingSnapshot, error) {
	if demandID == 0 {
		return nil, errors.New("demand id is invalid")
	}
	var row demandSchedulingSnapshot
	err := r.db.WithContext(ctx).Raw(`
SELECT
  id,
  product,
  QD,
  RD,
  IFNULL(DATE_FORMAT(estimateLaunch, '%Y-%m-%d'), '') AS estimateLaunch,
  IFNULL(DATE_FORMAT(developFinish, '%Y-%m-%d'), '') AS developFinish,
  IFNULL(DATE_FORMAT(testFinish, '%Y-%m-%d'), '') AS testFinish,
  IFNULL(DATE_FORMAT(verifyFinish, '%Y-%m-%d'), '') AS verifyFinish
FROM zt_demand
WHERE id = ? AND deleted = '0'
LIMIT 1`, demandID).Scan(&row).Error
	if err != nil {
		return nil, err
	}
	if row.ID == 0 {
		return nil, errors.New("业务需求不存在")
	}
	return &row, nil
}

// LogEditedHistory 有字段变化时写禅道 Edited 动作及 zt_history。
func (r *Repo) LogEditedHistory(ctx context.Context, objectType string, objectID uint, actor string, productID uint, projectID uint, executionID uint, changes []ztaction.Change) error {
	return r.LogEditedRecord(ctx, ztaction.Record{
		ObjectType: objectType,
		ObjectID:   objectID,
		ProductID:  productID,
		Project:    projectID,
		Execution:  executionID,
		Actor:      actor,
	}, changes)
}

// LogEditedRecord 使用调用方组装好的动作信息写 Edited 及字段差异。
func (r *Repo) LogEditedRecord(ctx context.Context, rec ztaction.Record, changes []ztaction.Change) error {
	return ztaction.LogEdited(ctx, r.db, rec, changes)
}

func normalizeHistoryDate(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" || strings.HasPrefix(raw, "0000-00-00") {
		return ""
	}
	if len(raw) >= 10 {
		return raw[:10]
	}
	return raw
}
