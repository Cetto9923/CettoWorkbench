// =============================================================================
// 文件: internal/module/cronjob/service.go
// 模块: 定时任务
// 类型: action
// 职责: 编排定时任务列表、启停、手动触发与日志查询逻辑。
// 依赖: internal/model
//       internal/pkg/cron
//       internal/pkg/pagination
// =============================================================================

package cronjob

import (
	"context"
	"errors"
	"strings"

	"gorm.io/gorm"

	"goframework/internal/model"
	cronpkg "goframework/internal/pkg/cron"
	"goframework/internal/pkg/pagination"
)

// Service 定时任务业务层。
type Service struct {
	repo    *Repo
	cronMgr *cronpkg.Manager
}

// NewService 创建 Service。
func NewService(repo *Repo, cronMgr *cronpkg.Manager) *Service {
	return &Service{
		repo:    repo,
		cronMgr: cronMgr,
	}
}

// List 查询任务列表。
func (s *Service) List(ctx context.Context, actor *model.User, req ListReq) (ListResp, error) {
	_ = actor
	if req.Page < 1 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = pagination.DefaultPageSize
	}
	items, total, err := s.repo.FindAll(ctx, RepoFindAllReq{
		Page:     req.Page,
		PageSize: req.PageSize,
	})
	if err != nil {
		return ListResp{}, err
	}
	return ListResp{
		Items: items,
		Pager: pagination.New(total, req.Page, req.PageSize),
	}, nil
}

// Toggle 启用或禁用任务。
func (s *Service) Toggle(ctx context.Context, actor *model.User, req ToggleReq) error {
	_ = actor
	job, err := s.repo.FindByID(ctx, req.ID)
	if err != nil {
		return err
	}
	if err := s.repo.UpdateEnabled(ctx, req.ID, req.IsEnabled); err != nil {
		return err
	}
	if s.cronMgr == nil {
		return nil
	}
	if err := s.cronMgr.SetEnabled(job.Name, req.IsEnabled); err != nil {
		return err
	}
	return nil
}

// Trigger 手动触发任务。
func (s *Service) Trigger(ctx context.Context, actor *model.User, req TriggerReq) error {
	_ = actor
	job, err := s.repo.FindByID(ctx, req.ID)
	if err != nil {
		return err
	}
	if !job.IsEnabled {
		return errors.New("任务已禁用，无法触发")
	}
	if s.cronMgr == nil {
		return errors.New("cron manager not initialized")
	}
	if err := s.cronMgr.Trigger(job.Name); err != nil {
		return err
	}
	return nil
}

// ListLogs 查询执行日志。
func (s *Service) ListLogs(ctx context.Context, actor *model.User, req LogListReq) (LogListResp, error) {
	_ = actor
	if req.Page < 1 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = pagination.DefaultPageSize
	}
	req.JobName = strings.TrimSpace(req.JobName)
	items, total, err := s.repo.FindLogs(ctx, RepoFindLogsReq{
		JobName:  req.JobName,
		Page:     req.Page,
		PageSize: req.PageSize,
	})
	if err != nil {
		return LogListResp{}, err
	}
	return LogListResp{
		Items: items,
		Pager: pagination.New(total, req.Page, req.PageSize),
	}, nil
}

// IsNotFound 判断是否为记录不存在。
func IsNotFound(err error) bool {
	return errors.Is(err, gorm.ErrRecordNotFound)
}
