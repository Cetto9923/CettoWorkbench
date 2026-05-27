// =============================================================================
// 文件: internal/module/backup/service.go
// 模块: 数据库备份
// 类型: action
// 职责: 实现备份记录查询、手动备份与下载路径解析。
// 依赖: internal/config
//       internal/model
//       internal/pkg/backup
//       internal/pkg/pagination
// =============================================================================

package backup

import (
	"context"
	"errors"
	"path/filepath"
	"strings"

	"gorm.io/gorm"

	"goframework/internal/config"
	"goframework/internal/model"
	backuppkg "goframework/internal/pkg/backup"
	"goframework/internal/pkg/pagination"
)

// Service 备份业务层。
type Service struct {
	repo *Repo
	cfg  *config.Config
	db   *gorm.DB
}

// NewService 创建 Service。
func NewService(repo *Repo, cfg *config.Config, db *gorm.DB) *Service {
	return &Service{
		repo: repo,
		cfg:  cfg,
		db:   db,
	}
}

// List 查询备份记录列表。
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

// Create 手动执行数据库备份。
func (s *Service) Create(ctx context.Context, actor *model.User, req CreateReq) (CreateResp, error) {
	_ = ctx
	_ = actor
	_ = req
	filename, err := backuppkg.Run(backuppkg.Config{
		DB:       s.db,
		Host:     s.cfg.Database.Host,
		Port:     s.cfg.Database.Port,
		User:     s.cfg.Database.User,
		Password: s.cfg.Database.Password,
		DBName:   s.cfg.Database.DBName,
		Dir:      s.cfg.Backup.Dir,
		KeepDays: s.cfg.Backup.KeepDays,
	})
	if err != nil {
		return CreateResp{}, err
	}
	return CreateResp{Filename: filename}, nil
}

// ResolveDownloadFile 根据记录 ID 解析可下载文件路径。
func (s *Service) ResolveDownloadFile(ctx context.Context, actor *model.User, req DownloadReq) (string, string, error) {
	_ = actor
	row, err := s.repo.FindByID(ctx, req.ID)
	if err != nil {
		return "", "", err
	}
	filename := strings.TrimSpace(row.Filename)
	if filename == "" {
		return "", "", errors.New("备份文件名为空")
	}
	baseDir := strings.TrimSpace(s.cfg.Backup.Dir)
	if baseDir == "" {
		baseDir = "backups"
	}
	cleanName := filepath.Base(filename)
	fullpath := filepath.Join(baseDir, cleanName)
	return fullpath, cleanName, nil
}

// IsNotFound 判断是否为记录不存在。
func IsNotFound(err error) bool {
	return errors.Is(err, gorm.ErrRecordNotFound)
}
