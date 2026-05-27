// =============================================================================
// 文件: internal/module/dictitem/service.go
// 模块: 字典值
// 类型: crud
// 职责: 实现字典值 CRUD 和 JSON 查询业务逻辑。
// 依赖: internal/model
//       internal/module/dictitem/repo.go
// =============================================================================

package dictitem

import (
	"context"
	"errors"
	"strings"
	"time"

	"goframework/internal/model"
)

// Service 处理字典值业务逻辑。
type Service struct {
	repo *Repo
}

// NewService 创建 Service。
func NewService(repo *Repo) *Service {
	return &Service{repo: repo}
}

// List 查询字典值列表。
func (s *Service) List(ctx context.Context, actor *model.User, req ListReq) (ListResp, error) {
	_ = actor
	req.Normalize()
	rows, _, err := s.repo.FindAll(ctx, RepoFindAllReq{TypeCode: req.TypeCode})
	if err != nil {
		return ListResp{}, err
	}
	return ListResp{Items: rows}, nil
}

// GetByID 查询字典值详情。
func (s *Service) GetByID(ctx context.Context, actor *model.User, id uint64) (*model.DictItem, error) {
	_ = actor
	return s.repo.FindByID(ctx, id)
}

// Create 创建字典值。
func (s *Service) Create(ctx context.Context, actor *model.User, req CreateReq) (CreateResp, error) {
	_ = actor
	m := &model.DictItem{
		TypeCode: strings.TrimSpace(req.TypeCode),
		Label:    strings.TrimSpace(req.Label),
		Value:    strings.TrimSpace(req.Value),
		Sort:     req.Sort,
	}
	if m.TypeCode == "" {
		return CreateResp{}, errors.New("字典类型编码不能为空")
	}
	if err := s.repo.Create(ctx, m); err != nil {
		return CreateResp{}, err
	}
	return CreateResp{ID: m.ID}, nil
}

// Update 更新字典值。
func (s *Service) Update(ctx context.Context, actor *model.User, req UpdateReq) error {
	_ = actor
	m, err := s.repo.FindByID(ctx, req.ID)
	if err != nil {
		return err
	}
	m.TypeCode = strings.TrimSpace(req.TypeCode)
	m.Label = strings.TrimSpace(req.Label)
	m.Value = strings.TrimSpace(req.Value)
	m.Sort = req.Sort
	m.UpdatedAt = time.Now()
	if m.TypeCode == "" {
		return errors.New("字典类型编码不能为空")
	}
	return s.repo.Update(ctx, m)
}

// Delete 删除字典值。
func (s *Service) Delete(ctx context.Context, actor *model.User, req DeleteReq) error {
	_ = actor
	return s.repo.Delete(ctx, req.ID)
}
