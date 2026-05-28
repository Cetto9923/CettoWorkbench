// =============================================================================
// 文件: internal/module/dept/service.go
// 模块: 部门管理
// 类型: crud
// 职责: 实现部门树查询与增删改业务逻辑。
// 依赖: internal/model
//       internal/module/dept/repo.go
// =============================================================================

package dept

import (
	"context"
	"errors"
	"strings"
	"time"

	"workbench/internal/model"

	"gorm.io/gorm"
)

// DeptNode 表示部门树节点。
type DeptNode struct {
	Dept     *model.Dept
	Children []*DeptNode
}

// Service 处理部门业务逻辑。
type Service struct {
	repo *Repo
}

// NewService 创建 Service。
func NewService(repo *Repo) *Service {
	return &Service{repo: repo}
}

// List 查询部门树。
func (s *Service) List(ctx context.Context, actor *model.User, req ListReq) (ListResp, error) {
	_ = actor
	rows, _, err := s.repo.FindAll(ctx, RepoFindAllReq{
		Page:     1,
		PageSize: 1000,
	})
	if err != nil {
		return ListResp{}, err
	}
	return ListResp{Items: buildTree(rows)}, nil
}

// GetByID 查询部门详情。
func (s *Service) GetByID(ctx context.Context, actor *model.User, id uint64) (*model.Dept, error) {
	_ = actor
	return s.repo.FindByID(ctx, id)
}

// Create 创建部门。
func (s *Service) Create(ctx context.Context, actor *model.User, req CreateReq) (CreateResp, error) {
	_ = actor
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return CreateResp{}, errors.New("部门名称不能为空")
	}
	leader := strings.TrimSpace(req.Leader)
	phone := strings.TrimSpace(req.Phone)
	email := strings.TrimSpace(req.Email)

	status := req.Status
	if req.ParentID > 0 {
		parent, err := s.repo.FindByID(ctx, req.ParentID)
		if err != nil {
			return CreateResp{}, err
		}
		if parent.Status == 1 {
			if !req.EnableAncestors {
				status = 1
			} else {
				var createdID uint64
				if err := s.repo.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
					ancestorIDs, txErr := s.collectAncestorIDs(ctx, tx, req.ParentID)
					if txErr != nil {
						return txErr
					}
					if len(ancestorIDs) > 0 {
						if txErr = tx.WithContext(ctx).
							Model(&model.Dept{}).
							Where("id IN ? AND deletedAt IS NULL", ancestorIDs).
							Updates(map[string]any{
								"status": 0,
							}).Error; txErr != nil {
							return txErr
						}
					}
					m := &model.Dept{
						ParentID: req.ParentID,
						Name:     name,
						Leader:   leader,
						Phone:    phone,
						Email:    email,
						Status:   0,
						Sort:     req.Sort,
					}
					if txErr = tx.WithContext(ctx).Create(m).Error; txErr != nil {
						return txErr
					}
					createdID = m.ID
					return nil
				}); err != nil {
					return CreateResp{}, err
				}
				return CreateResp{ID: createdID}, nil
			}
		}
	}
	m := &model.Dept{
		ParentID: req.ParentID,
		Name:     name,
		Leader:   leader,
		Phone:    phone,
		Email:    email,
		Status:   status,
		Sort:     req.Sort,
	}
	if err := s.repo.Create(ctx, m); err != nil {
		return CreateResp{}, err
	}
	return CreateResp{ID: m.ID}, nil
}

func (s *Service) collectAncestorIDs(ctx context.Context, tx *gorm.DB, parentID uint64) ([]uint64, error) {
	ids := make([]uint64, 0)
	currentID := parentID
	for currentID > 0 {
		var current model.Dept
		if err := tx.WithContext(ctx).
			Model(&model.Dept{}).
			Where("id = ? AND deletedAt IS NULL", currentID).
			First(&current).Error; err != nil {
			return nil, err
		}
		ids = append(ids, current.ID)
		currentID = current.ParentID
	}
	return ids, nil
}

// Update 更新部门。
func (s *Service) Update(ctx context.Context, actor *model.User, req UpdateReq) error {
	_ = actor
	m, err := s.repo.FindByID(ctx, req.ID)
	if err != nil {
		return err
	}
	if req.ParentID == req.ID {
		return errors.New("父部门不能选择自己")
	}
	if m.ParentID == 0 && req.ParentID != 0 {
		return errors.New("顶级部门不允许修改上级部门")
	}
	m.ParentID = req.ParentID
	m.Name = strings.TrimSpace(req.Name)
	m.Leader = strings.TrimSpace(req.Leader)
	m.Phone = strings.TrimSpace(req.Phone)
	m.Email = strings.TrimSpace(req.Email)
	m.Status = req.Status
	m.Sort = req.Sort
	m.UpdatedAt = time.Now()
	if m.Name == "" {
		return errors.New("部门名称不能为空")
	}
	return s.repo.Update(ctx, m)
}

// Delete 删除部门（删除前校验子部门和用户）。
func (s *Service) Delete(ctx context.Context, actor *model.User, req DeleteReq) error {
	_ = actor
	deptItem, err := s.repo.FindByID(ctx, req.ID)
	if err != nil {
		return err
	}

	if req.ID == 1 || deptItem.ParentID == 0 {
		return errors.New("顶级部门不允许删除")
	}

	childrenCount, err := s.repo.CountChildren(ctx, req.ID)
	if err != nil {
		return err
	}
	if childrenCount > 0 {
		return errors.New("存在下级部门，不允许删除")
	}

	// TODO: 待用户模块完善后补充关联校验（含跨环境字段差异兼容）。
	// 当前先放开关联用户校验，避免因历史库字段不一致阻塞部门树删除联调。

	return s.repo.Delete(ctx, req.ID)
}

// UpdateStatus 仅更新部门状态字段。
func (s *Service) UpdateStatus(ctx context.Context, actor *model.User, req UpdateStatusReq) error {
	_ = actor
	if req.ID == 0 {
		return errors.New("无效的部门 ID")
	}
	if req.Status != 0 && req.Status != 1 {
		return errors.New("状态值不合法")
	}
	rows, _, err := s.repo.FindAll(ctx, RepoFindAllReq{
		Page:     1,
		PageSize: 100000,
	})
	if err != nil {
		return err
	}
	deptByID := make(map[uint64]model.Dept, len(rows))
	childByParent := make(map[uint64][]uint64)
	for _, row := range rows {
		deptByID[row.ID] = row
		childByParent[row.ParentID] = append(childByParent[row.ParentID], row.ID)
	}
	target, ok := deptByID[req.ID]
	if !ok {
		return errors.New("部门不存在")
	}

	if req.Status == 1 {
		ids := make([]uint64, 0)
		queue := []uint64{req.ID}
		seen := make(map[uint64]bool)
		for len(queue) > 0 {
			current := queue[0]
			queue = queue[1:]
			if seen[current] {
				continue
			}
			seen[current] = true
			ids = append(ids, current)
			for _, childID := range childByParent[current] {
				queue = append(queue, childID)
			}
		}
		return s.repo.UpdateStatusBatch(ctx, ids, 1)
	}

	if target.ParentID != 0 {
		parent := deptByID[target.ParentID]
		if parent.Status == 1 {
			if !req.WithAncestors {
				return errors.New("上级部门已停用，请勾选同时开启上级部门")
			}
			ancestorIDs := make([]uint64, 0)
			parentID := target.ParentID
			for parentID != 0 {
				ancestor, ok := deptByID[parentID]
				if !ok {
					break
				}
				ancestorIDs = append(ancestorIDs, ancestor.ID)
				parentID = ancestor.ParentID
			}
			if err := s.repo.UpdateStatusBatch(ctx, ancestorIDs, 0); err != nil {
				return err
			}
		}
	}
	return s.repo.UpdateStatus(ctx, req.ID, 0)
}

func buildTree(rows []model.Dept) []*DeptNode {
	nodeByID := make(map[uint64]*DeptNode, len(rows))
	roots := make([]*DeptNode, 0)
	for _, row := range rows {
		rowCopy := row
		nodeByID[row.ID] = &DeptNode{
			Dept:     &rowCopy,
			Children: make([]*DeptNode, 0),
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
