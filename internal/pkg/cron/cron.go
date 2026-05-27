// =============================================================================
// 文件: internal/pkg/cron/cron.go
// 模块: 工具组件
// 类型: pkg
// 职责: 封装定时任务注册、调度与执行日志落库。
// 依赖: internal/model
//       github.com/robfig/cron/v3
// =============================================================================

package cron

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	cronlib "github.com/robfig/cron/v3"
	"gorm.io/gorm"

	"workbrench/internal/model"
)

// Manager 封装 robfig/cron 的任务管理能力。
type Manager struct {
	db      *gorm.DB
	engine  *cronlib.Cron
	mu      sync.Mutex
	entries map[string]cronlib.EntryID
	jobs    map[string]registeredJob
}

type registeredJob struct {
	spec string
	fn   func()
}

// New 创建定时任务管理器。
func New(db *gorm.DB) *Manager {
	return &Manager{
		db:      db,
		engine:  cronlib.New(),
		entries: make(map[string]cronlib.EntryID),
		jobs:    make(map[string]registeredJob),
	}
}

// Register 注册任务，任务是否启用以数据库 zt_cron_jobs 配置为准。
func (m *Manager) Register(name, spec string, fn func()) error {
	if m == nil || m.db == nil {
		return errors.New("cron manager is not initialized")
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return errors.New("cron job name is required")
	}
	if fn == nil {
		return errors.New("cron job function is nil")
	}

	m.mu.Lock()
	m.jobs[name] = registeredJob{
		spec: strings.TrimSpace(spec),
		fn:   fn,
	}
	m.mu.Unlock()

	job, err := m.ensureJob(name, spec)
	if err != nil {
		return err
	}
	if !job.IsEnabled {
		m.removeEntry(name)
		return nil
	}
	runSpec := strings.TrimSpace(job.Spec)
	if runSpec == "" {
		runSpec = strings.TrimSpace(spec)
	}
	if runSpec == "" {
		return fmt.Errorf("cron job %q spec is empty", name)
	}
	return m.addOrUpdateEntry(name, runSpec, fn)
}

// SetEnabled 设置任务启用状态，并实时更新调度器注册状态。
func (m *Manager) SetEnabled(name string, enabled bool) error {
	if m == nil || m.db == nil {
		return errors.New("cron manager is not initialized")
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return errors.New("cron job name is required")
	}

	if err := m.db.Model(&model.CronJob{}).
		Where("name = ?", name).
		Update("isEnabled", enabled).Error; err != nil {
		return fmt.Errorf("update cron job %q enabled failed: %w", name, err)
	}

	if !enabled {
		m.removeEntry(name)
		return nil
	}

	var row model.CronJob
	if err := m.db.Where("name = ?", name).First(&row).Error; err != nil {
		return fmt.Errorf("query cron job %q failed: %w", name, err)
	}

	m.mu.Lock()
	job, ok := m.jobs[name]
	m.mu.Unlock()
	if !ok || job.fn == nil {
		return nil
	}
	spec := strings.TrimSpace(row.Spec)
	if spec == "" {
		spec = strings.TrimSpace(job.spec)
	}
	if spec == "" {
		return fmt.Errorf("cron job %q spec is empty", name)
	}
	return m.addOrUpdateEntry(name, spec, job.fn)
}

// Trigger 手动触发任务并记录执行日志。
func (m *Manager) Trigger(name string) error {
	if m == nil {
		return errors.New("cron manager is not initialized")
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return errors.New("cron job name is required")
	}
	m.mu.Lock()
	job, ok := m.jobs[name]
	m.mu.Unlock()
	if !ok || job.fn == nil {
		return fmt.Errorf("cron job %q is not registered", name)
	}
	m.wrap(name, job.fn)()
	return nil
}

func (m *Manager) addOrUpdateEntry(name, spec string, fn func()) error {
	entryID, err := m.engine.AddFunc(spec, m.wrap(name, fn))
	if err != nil {
		return fmt.Errorf("register cron job %q failed: %w", name, err)
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	if oldID, ok := m.entries[name]; ok {
		m.engine.Remove(oldID)
	}
	m.entries[name] = entryID
	return nil
}

func (m *Manager) removeEntry(name string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if oldID, ok := m.entries[name]; ok {
		m.engine.Remove(oldID)
		delete(m.entries, name)
	}
}

// Start 启动调度器。
func (m *Manager) Start() {
	if m == nil || m.engine == nil {
		return
	}
	m.engine.Start()
}

// Stop 停止调度器。
func (m *Manager) Stop() {
	if m == nil || m.engine == nil {
		return
	}
	ctx := m.engine.Stop()
	<-ctx.Done()
}

func (m *Manager) ensureJob(name, spec string) (*model.CronJob, error) {
	var job model.CronJob
	err := m.db.Where("name = ?", name).First(&job).Error
	if err == nil {
		return &job, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("query cron job %q failed: %w", name, err)
	}

	job = model.CronJob{
		Name:      name,
		Spec:      strings.TrimSpace(spec),
		IsEnabled: true,
		Remark:    "builtin",
	}
	if err := m.db.Create(&job).Error; err != nil {
		return nil, fmt.Errorf("create cron job %q failed: %w", name, err)
	}
	return &job, nil
}

func (m *Manager) wrap(name string, fn func()) func() {
	return func() {
		startAt := time.Now()
		logRow := model.CronLog{
			JobName: name,
			StartAt: startAt,
			Success: false,
		}
		_ = m.db.Create(&logRow).Error

		success := true
		message := ""
		defer func() {
			endAt := time.Now()
			if rec := recover(); rec != nil {
				success = false
				message = fmt.Sprintf("panic: %v", rec)
			}

			updates := map[string]any{
				"endAt":   endAt,
				"success": success,
				"message": message,
			}
			if logRow.ID > 0 {
				_ = m.db.Model(&model.CronLog{}).
					Where("id = ?", logRow.ID).
					Updates(updates).Error
			}
			_ = m.db.Model(&model.CronJob{}).
				Where("name = ?", name).
				Updates(map[string]any{"lastRunAt": endAt}).Error
		}()

		fn()
	}
}
