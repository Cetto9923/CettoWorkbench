// =============================================================================
// 文件: internal/module/menu/service.go
// 模块: 菜单管理
// 类型: crud
// 职责: 实现菜单树查询与增删改业务逻辑。
// 依赖: internal/model
//       internal/module/menu/repo.go
// =============================================================================

package menu

import (
	"context"
	"errors"
	"strings"
	"time"

	"workbench/internal/model"
)

// MenuNode 表示菜单树节点。
type MenuNode struct {
	Menu     *model.Menu
	Children []*MenuNode
}

// Service 处理菜单业务逻辑。
type Service struct {
	repo *Repo
}

// NewService 创建 Service。
func NewService(repo *Repo) *Service {
	return &Service{repo: repo}
}

// List 返回菜单列表。
func (s *Service) List(ctx context.Context, actor *model.User, req ListReq) (ListResp, error) {
	_ = actor
	if req.Page < 1 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 1000
	}
	rows, total, err := s.repo.FindAll(ctx, RepoFindAllReq{
		Page:     req.Page,
		PageSize: req.PageSize,
	})
	if err != nil {
		return ListResp{}, err
	}
	return ListResp{
		Items: rows,
		Total: total,
	}, nil
}

// GetByID 获取菜单详情。
func (s *Service) GetByID(ctx context.Context, actor *model.User, id uint64) (*model.Menu, error) {
	_ = actor
	return s.repo.FindByID(ctx, id)
}

// Create 创建菜单。
func (s *Service) Create(ctx context.Context, actor *model.User, req CreateReq) (CreateResp, error) {
	_ = actor
	m := &model.Menu{
		ParentID: req.ParentID,
		Type:     normalizeMenuType(req.Type),
		Title:    strings.TrimSpace(req.Title),
		Icon:     strings.TrimSpace(req.Icon),
		Path:     strings.TrimSpace(req.Path),
		Perm:     strings.TrimSpace(req.Perm),
		Sort:     req.Sort,
	}
	if m.Title == "" {
		return CreateResp{}, errors.New("菜单标题不能为空")
	}
	if err := s.repo.Create(ctx, m); err != nil {
		return CreateResp{}, err
	}
	return CreateResp{ID: m.ID}, nil
}

// Update 更新菜单。
func (s *Service) Update(ctx context.Context, actor *model.User, req UpdateReq) (UpdateResp, error) {
	_ = actor
	m, err := s.repo.FindByID(ctx, req.ID)
	if err != nil {
		return UpdateResp{}, err
	}
	m.ParentID = req.ParentID
	m.Type = normalizeMenuType(req.Type)
	m.Title = strings.TrimSpace(req.Title)
	m.Icon = strings.TrimSpace(req.Icon)
	m.Path = strings.TrimSpace(req.Path)
	m.Perm = strings.TrimSpace(req.Perm)
	m.Sort = req.Sort
	m.UpdatedAt = time.Now()
	if m.Title == "" {
		return UpdateResp{}, errors.New("菜单标题不能为空")
	}
	if err := s.repo.Update(ctx, m); err != nil {
		return UpdateResp{}, err
	}
	return UpdateResp{ID: m.ID}, nil
}

// Delete 删除菜单（有子菜单则拒绝）。
func (s *Service) Delete(ctx context.Context, actor *model.User, req DeleteReq) error {
	_ = actor
	children, err := s.repo.FindChildren(ctx, req.ID)
	if err != nil {
		return err
	}
	if len(children) > 0 {
		return errors.New("存在子菜单，不能删除")
	}
	return s.repo.Delete(ctx, req.ID)
}

func buildTree(rows []model.Menu) []*MenuNode {
	nodeByID := make(map[uint64]*MenuNode, len(rows))
	roots := make([]*MenuNode, 0)
	for _, row := range rows {
		rowCopy := row
		nodeByID[row.ID] = &MenuNode{
			Menu:     &rowCopy,
			Children: make([]*MenuNode, 0),
		}
	}
	for _, row := range rows {
		node := nodeByID[row.ID]
		if row.ParentID == 0 {
			roots = append(roots, node)
			continue
		}
		parent, ok := nodeByID[row.ParentID]
		if !ok {
			roots = append(roots, node)
			continue
		}
		parent.Children = append(parent.Children, node)
	}
	return roots
}

func normalizeMenuType(raw string) string {
	value := strings.ToUpper(strings.TrimSpace(raw))
	switch value {
	case "M", "C", "F":
		return value
	default:
		return "C"
	}
}
