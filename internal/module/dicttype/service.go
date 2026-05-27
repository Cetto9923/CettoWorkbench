// =============================================================================
// 文件: internal/module/dicttype/service.go
// 模块: 字典类型
// 类型: crud
// 职责: 实现字典类型 CRUD 业务逻辑。
// 依赖: internal/model
//       internal/module/dicttype/repo.go
// =============================================================================

package dicttype

import (
	"context"
	"errors"
	"strings"
	"time"

	"goframework/internal/model"
)

// Service 处理字典类型业务逻辑。
type Service struct {
	repo *Repo
}

// NewService 创建 Service。
func NewService(repo *Repo) *Service {
	return &Service{repo: repo}
}

// List 查询字典类型列表。
func (s *Service) List(ctx context.Context, actor *model.User, req ListReq) (ListResp, error) {
	_ = actor
	req.Normalize()
	rows, _, err := s.repo.FindAll(ctx, RepoFindAllReq{Keyword: req.Keyword})
	if err != nil {
		return ListResp{}, err
	}
	return ListResp{Items: rows}, nil
}

// GetByID 查询字典类型详情。
func (s *Service) GetByID(ctx context.Context, actor *model.User, id uint64) (*model.DictType, error) {
	_ = actor
	return s.repo.FindByID(ctx, id)
}

// Create 创建字典类型。
func (s *Service) Create(ctx context.Context, actor *model.User, req CreateReq) (CreateResp, error) {
	_ = actor
	if ok, err := s.repo.ExistsByCode(ctx, req.Code, 0); err != nil {
		return CreateResp{}, err
	} else if ok {
		return CreateResp{}, errors.New("类型编码已存在")
	}
	m := &model.DictType{
		Code:   strings.TrimSpace(req.Code),
		Name:   strings.TrimSpace(req.Name),
		Remark: strings.TrimSpace(req.Remark),
	}
	if err := s.repo.Create(ctx, m); err != nil {
		return CreateResp{}, err
	}
	return CreateResp{ID: m.ID}, nil
}

// Update 更新字典类型。
func (s *Service) Update(ctx context.Context, actor *model.User, req UpdateReq) error {
	_ = actor
	m, err := s.repo.FindByID(ctx, req.ID)
	if err != nil {
		return err
	}
	if ok, err := s.repo.ExistsByCode(ctx, req.Code, req.ID); err != nil {
		return err
	} else if ok {
		return errors.New("类型编码已存在")
	}
	m.Code = strings.TrimSpace(req.Code)
	m.Name = strings.TrimSpace(req.Name)
	m.Remark = strings.TrimSpace(req.Remark)
	m.UpdatedAt = time.Now()
	return s.repo.Update(ctx, m)
}

// Delete 删除字典类型。
func (s *Service) Delete(ctx context.Context, actor *model.User, req DeleteReq) error {
	_ = actor
	m, err := s.repo.FindByID(ctx, req.ID)
	if err != nil {
		return err
	}
	hasItems, err := s.repo.HasItems(ctx, m.Code)
	if err != nil {
		return err
	}
	if hasItems {
		return errors.New("字典类型下仍存在字典值，无法删除")
	}
	return s.repo.Delete(ctx, req.ID)
}
