// =============================================================================
// 文件: internal/module/schedule/repo_scheduling_write.go
// 模块: 排期工作台
// 类型: action
// 职责: 排期一体化「确认并同步」写库操作(写函数 + 写入配套类型 + helper)。
// 依赖: internal/model
// =============================================================================

package schedule

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"workbench/internal/model"
)

// ZtStoryInsert 禅道 zt_story 写入字段。
type ZtStoryInsert struct {
	Product                 uint
	Title                   string
	AssignedTo              string
	Estimate                float64
	FromDemand              uint
	IsMainSystemAssociation string
	OpenedBy                string
}

// ZtStorySpec 禅道 zt_storyspec 写入字段。
type ZtStorySpec struct {
	Story   uint
	Version int
	Title   string
	Spec    string
}

// ZtTaskInsert 禅道 zt_task 写入字段。
type ZtTaskInsert struct {
	Name       string
	Type       string
	Pri        int
	Story      uint
	Project    uint
	Execution  uint
	AssignedTo string
	Estimate   float64
	EstStarted string
	Deadline   string
	OpenedBy   string
}

// ZtTaskSpec 禅道 zt_taskspec 写入字段。
type ZtTaskSpec struct {
	Task       uint
	Version    int
	Name       string
	EstStarted string
	Deadline   string
}

type ztStoryCreateRow struct {
	ID                      uint      `gorm:"column:id;primaryKey;autoIncrement"`
	Product                 uint      `gorm:"column:product"`
	Branch                  string    `gorm:"column:branch"`
	Module                  uint      `gorm:"column:module"`
	Plan                    string    `gorm:"column:plan"`
	Source                  string    `gorm:"column:source"`
	SourceNote              string    `gorm:"column:sourceNote"`
	Title                   string    `gorm:"column:title"`
	Type                    string    `gorm:"column:type"`
	Pri                     int       `gorm:"column:pri"`
	Grade                   int       `gorm:"column:grade"`
	Estimate                float64   `gorm:"column:estimate"`
	Status                  string    `gorm:"column:status"`
	Stage                   string    `gorm:"column:stage"`
	SourceType              string    `gorm:"column:sourceType"`
	FromDemand              uint      `gorm:"column:fromDemand"`
	Version                 int       `gorm:"column:version"`
	OpenedBy                string    `gorm:"column:openedBy"`
	OpenedDate              time.Time `gorm:"column:openedDate"`
	AssignedTo              string    `gorm:"column:assignedTo"`
	IsMainSystemAssociation string    `gorm:"column:isMainSystemAssociation"`
	VerifyPlan              string    `gorm:"column:verifyPlan"`
	Deleted                 string    `gorm:"column:deleted"`
}

func (ztStoryCreateRow) TableName() string { return "zt_story" }

type ztStorySpecRow struct {
	Story   uint   `gorm:"column:story;primaryKey"`
	Version int    `gorm:"column:version;primaryKey"`
	Title   string `gorm:"column:title"`
	Spec    string `gorm:"column:spec"`
}

func (ztStorySpecRow) TableName() string { return "zt_storyspec" }

type ztPlanStoryRow struct {
	Plan  uint `gorm:"column:plan;primaryKey"`
	Story uint `gorm:"column:story;primaryKey"`
	Order int  `gorm:"column:order"`
}

func (ztPlanStoryRow) TableName() string { return "zt_planstory" }

type ztTaskCreateRow struct {
	ID         uint      `gorm:"column:id;primaryKey;autoIncrement"`
	Name       string    `gorm:"column:name"`
	Type       string    `gorm:"column:type"`
	Pri        int       `gorm:"column:pri"`
	Story      uint      `gorm:"column:story"`
	Project    uint      `gorm:"column:project"`
	Execution  uint      `gorm:"column:execution"`
	AssignedTo string    `gorm:"column:assignedTo"`
	Estimate   float64   `gorm:"column:estimate"`
	Consumed   float64   `gorm:"column:consumed"`
	Left       float64   `gorm:"column:left"`
	EstStarted string    `gorm:"column:estStarted"`
	Deadline   string    `gorm:"column:deadline"`
	Status     string    `gorm:"column:status"`
	OpenedBy   string    `gorm:"column:openedBy"`
	OpenedDate time.Time `gorm:"column:openedDate"`
	Version    int       `gorm:"column:version"`
	Deleted    string    `gorm:"column:deleted"`
}

func (ztTaskCreateRow) TableName() string { return "zt_task" }

type ztTaskSpecRow struct {
	Task       uint   `gorm:"column:task;primaryKey"`
	Version    int    `gorm:"column:version;primaryKey"`
	Name       string `gorm:"column:name"`
	EstStarted string `gorm:"column:estStarted"`
	Deadline   string `gorm:"column:deadline"`
}

func (ztTaskSpecRow) TableName() string { return "zt_taskspec" }

type ztActionRow struct {
	ID         uint      `gorm:"column:id;primaryKey;autoIncrement"`
	ObjectType string    `gorm:"column:objectType"`
	ObjectID   uint      `gorm:"column:objectID"`
	Product    string    `gorm:"column:product"`
	Project    uint      `gorm:"column:project"`
	Execution  uint      `gorm:"column:execution"`
	Actor      string    `gorm:"column:actor"`
	Action     string    `gorm:"column:action"`
	Date       time.Time `gorm:"column:date"`
	Comment    string    `gorm:"column:comment"`
	Extra      string    `gorm:"column:extra"`
}

func (ztActionRow) TableName() string { return "zt_action" }

// UpdateWindowProductPlanID 更新窗口-产品关联的计划 ID。
func (r *Repo) UpdateWindowProductPlanID(ctx context.Context, id uint64, planID uint, account string) error {
	return r.db.WithContext(ctx).
		Model(&model.VersionWindowProduct{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"plan":       planID,
			"planSynced": uint8(1),
			"updatedBy":  account,
		}).Error
}

// CreateStory 创建研发需求。
func (r *Repo) CreateStory(ctx context.Context, story *ZtStoryInsert) (uint, error) {
	if story == nil {
		return 0, errors.New("story is nil")
	}
	now := time.Now()
	row := ztStoryCreateRow{
		Product:                 story.Product,
		Branch:                  "0",
		Module:                  0,
		Plan:                    "",
		Source:                  "",
		SourceNote:              "",
		Title:                   strings.TrimSpace(story.Title),
		Type:                    "story",
		Pri:                     3,
		Grade:                   1,
		Estimate:                story.Estimate,
		Status:                  "active",
		Stage:                   "planned",
		SourceType:              "demandpool",
		FromDemand:              story.FromDemand,
		Version:                 1,
		OpenedBy:                story.OpenedBy,
		OpenedDate:              now,
		AssignedTo:              strings.TrimSpace(story.AssignedTo),
		IsMainSystemAssociation: story.IsMainSystemAssociation,
		VerifyPlan:              "",
		Deleted:                 "0",
	}
	if err := r.db.WithContext(ctx).Create(&row).Error; err != nil {
		return 0, err
	}
	return row.ID, nil
}

// CreateStorySpec 创建研发需求描述。
func (r *Repo) CreateStorySpec(ctx context.Context, spec *ZtStorySpec) error {
	if spec == nil {
		return errors.New("story spec is nil")
	}
	row := ztStorySpecRow{
		Story:   spec.Story,
		Version: spec.Version,
		Title:   strings.TrimSpace(spec.Title),
		Spec:    spec.Spec,
	}
	return r.db.WithContext(ctx).Create(&row).Error
}

// CreatePlanStory 关联计划与研发需求。
func (r *Repo) CreatePlanStory(ctx context.Context, planID uint, storyID uint) error {
	if planID == 0 || storyID == 0 {
		return errors.New("plan or story id is invalid")
	}
	row := ztPlanStoryRow{Plan: planID, Story: storyID, Order: 0}
	return r.db.WithContext(ctx).Create(&row).Error
}

// UpdateStory 更新研发需求字段。
func (r *Repo) UpdateStory(ctx context.Context, storyID uint, updates map[string]interface{}) error {
	if storyID == 0 {
		return errors.New("story id is invalid")
	}
	if len(updates) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).
		Table("zt_story").
		Where("id = ? AND deleted = '0'", storyID).
		Updates(updates).Error
}

// CloseStory 关闭研发需求。
func (r *Repo) CloseStory(ctx context.Context, storyID uint, actor string) error {
	return r.UpdateStory(ctx, storyID, map[string]interface{}{
		"status":       "closed",
		"closedBy":     actor,
		"closedDate":   time.Now(),
		"closedReason": "done",
	})
}

// CreateTask 创建任务。
func (r *Repo) CreateTask(ctx context.Context, task *ZtTaskInsert) (uint, error) {
	if task == nil {
		return 0, errors.New("task is nil")
	}
	now := time.Now()
	row := ztTaskCreateRow{
		Name:       strings.TrimSpace(task.Name),
		Type:       strings.TrimSpace(task.Type),
		Pri:        task.Pri,
		Story:      task.Story,
		Project:    task.Project,
		Execution:  task.Execution,
		AssignedTo: strings.TrimSpace(task.AssignedTo),
		Estimate:   task.Estimate,
		Consumed:   0,
		Left:       task.Estimate,
		EstStarted: nullableDateValue(task.EstStarted),
		Deadline:   nullableDateValue(task.Deadline),
		Status:     "wait",
		OpenedBy:   task.OpenedBy,
		OpenedDate: now,
		Version:    1,
		Deleted:    "0",
	}
	if err := r.db.WithContext(ctx).Create(&row).Error; err != nil {
		return 0, err
	}
	return row.ID, nil
}

// CreateTaskSpec 创建任务描述。
func (r *Repo) CreateTaskSpec(ctx context.Context, spec *ZtTaskSpec) error {
	if spec == nil {
		return errors.New("task spec is nil")
	}
	row := ztTaskSpecRow{
		Task:       spec.Task,
		Version:    spec.Version,
		Name:       strings.TrimSpace(spec.Name),
		EstStarted: nullableDateValue(spec.EstStarted),
		Deadline:   nullableDateValue(spec.Deadline),
	}
	return r.db.WithContext(ctx).Create(&row).Error
}

// UpdateTask 更新任务字段。
func (r *Repo) UpdateTask(ctx context.Context, taskID uint, updates map[string]interface{}) error {
	if taskID == 0 {
		return errors.New("task id is invalid")
	}
	if len(updates) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).
		Table("zt_task").
		Where("id = ? AND deleted = '0'", taskID).
		Updates(updates).Error
}

// CloseTask 关闭任务。
func (r *Repo) CloseTask(ctx context.Context, taskID uint, actor string) error {
	return r.UpdateTask(ctx, taskID, map[string]interface{}{
		"status":       "closed",
		"closedBy":     actor,
		"closedDate":   time.Now(),
		"closedReason": "done",
		"assignedTo":   "closed",
	})
}

// CreateAction 创建禅道操作日志。
func (r *Repo) CreateAction(ctx context.Context, objectType string, objectID uint, action string, actor string, productID uint, projectID uint, executionID uint, extra string) error {
	productField := ",0,"
	if productID > 0 {
		productField = fmt.Sprintf(",%d,", productID)
	}
	row := ztActionRow{
		ObjectType: objectType,
		ObjectID:   objectID,
		Product:    productField,
		Project:    projectID,
		Execution:  executionID,
		Actor:      actor,
		Action:     action,
		Date:       time.Now(),
		Comment:    "",
		Extra:      extra,
	}
	return r.db.WithContext(ctx).Create(&row).Error
}

// UpdateDemandScheduling 更新业需排期字段。
func (r *Repo) UpdateDemandScheduling(ctx context.Context, demandID uint, updates map[string]interface{}) error {
	if demandID == 0 {
		return errors.New("demand id is invalid")
	}
	if len(updates) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).
		Table("zt_demand").
		Where("id = ? AND deleted = '0'", demandID).
		Updates(updates).Error
}

// SaveDemandLevelWindow 保存业需级窗口关联（硬删除旧记录后 INSERT 单行）。
// 用 Unscoped 绕开 gorm 软删除以释放 uk_demand_story 唯一键位，避免残留行撞键；
// 实际调用方 service 将本方法包在事务内以保证两步原子性。
func (r *Repo) SaveDemandLevelWindow(ctx context.Context, demandID uint, windowID uint64, account string) error {
	if err := r.db.WithContext(ctx).
		Unscoped().
		Where("demand = ? AND story = 0", demandID).
		Delete(&model.DemandWindow{}).Error; err != nil {
		return err
	}
	return r.db.WithContext(ctx).Create(&model.DemandWindow{
		DemandID:  demandID,
		StoryID:   0,
		WindowID:  windowID,
		CreatedBy: account,
		UpdatedBy: account,
	}).Error
}

// nullableDateValue 把空字符串转为禅道兼容占位 "0000-00-00"。供 CreateTask/CreateTaskSpec 共用。
func nullableDateValue(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "0000-00-00"
	}
	return raw
}

// nullableSchedulingDate 把空字符串转为 nil 以便 GORM 写入 NULL。供排期保存共用。
func nullableSchedulingDate(raw string) interface{} {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	return raw
}
