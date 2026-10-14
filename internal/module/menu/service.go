// =============================================================================
// 文件: internal/module/menu/service.go
// 模块: 菜单管理
// 类型: crud
// 职责: 实现菜单树查询、增删改与用户固定工作台菜单业务逻辑。
// 依赖: internal/model
//       internal/module/menu/repo.go
//       internal/pkg/errorx
// =============================================================================

package menu

import (
	"context"
	"errors"
	"strings"
	"time"

	"workbench/internal/model"
	"workbench/internal/pkg/errorx"
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

// ListUserMenus 返回当前用户可见的固定菜单（title/path 来自 zt_menus）。
func (s *Service) ListUserMenus(ctx context.Context, actor *model.User, perms map[string]bool) (ListUserMenuResp, error) {
	if actor == nil || strings.TrimSpace(actor.Account) == "" {
		return ListUserMenuResp{}, errorx.New(errorx.ErrCodeForbidden, "请先登录")
	}
	rows, err := s.repo.FindUserMenusByAccount(ctx, actor.Account)
	if err != nil {
		return ListUserMenuResp{}, err
	}
	items := make([]UserMenuItem, 0, len(rows))
	for _, row := range rows {
		if !menuPermAllowed(perms, actor.IsSuperAdmin, row.Perm) {
			continue
		}
		items = append(items, UserMenuItem{
			ID:     row.ID,
			MenuID: row.MenuID,
			Title:  row.Title,
			Path:   row.Path,
			Icon:   row.Icon,
			Sort:   row.Sort,
		})
	}
	return ListUserMenuResp{Items: items}, nil
}

// PinUserMenu 按 path 解析 zt_menus 后固定到当前用户。
func (s *Service) PinUserMenu(ctx context.Context, actor *model.User, perms map[string]bool, req PinUserMenuReq) (PinUserMenuResp, error) {
	if actor == nil || strings.TrimSpace(actor.Account) == "" {
		return PinUserMenuResp{}, errorx.New(errorx.ErrCodeForbidden, "请先登录")
	}
	path := strings.TrimSpace(req.Path)
	m, err := s.resolveMenuByPath(ctx, path, perms, actor.IsSuperAdmin)
	if err != nil {
		return PinUserMenuResp{}, err
	}

	existing, findErr := s.repo.FindUserMenuByAccountMenuID(ctx, actor.Account, m.ID)
	if findErr != nil && !isNotFound(findErr) {
		return PinUserMenuResp{}, findErr
	}

	maxSort, err := s.repo.MaxUserMenuSort(ctx, actor.Account)
	if err != nil {
		return PinUserMenuResp{}, err
	}
	nextSort := maxSort + 1

	if existing != nil {
		return PinUserMenuResp{Item: UserMenuItem{
			ID: existing.ID, MenuID: m.ID, Title: m.Title, Path: m.Path, Icon: m.Icon, Sort: existing.Sort,
		}}, nil
	}

	now := time.Now()
	row := &model.UserMenu{
		CreatedBy:   actor.Account,
		CreatedDate: now,
		UpdatedBy:   actor.Account,
		UpdatedDate: now,
		Account: actor.Account,
		MenuID:  m.ID,
		Sort:    nextSort,
	}
	id, err := s.repo.CreateUserMenu(ctx, row)
	if err != nil {
		return PinUserMenuResp{}, err
	}
	return PinUserMenuResp{Item: UserMenuItem{
		ID: uint64(id), MenuID: m.ID, Title: m.Title, Path: m.Path, Icon: m.Icon, Sort: nextSort,
	}}, nil
}

// UnpinUserMenu 取消固定。
func (s *Service) UnpinUserMenu(ctx context.Context, actor *model.User, req UnpinUserMenuReq) error {
	if actor == nil || strings.TrimSpace(actor.Account) == "" {
		return errorx.New(errorx.ErrCodeForbidden, "请先登录")
	}
	if req.MenuID == 0 {
		return errorx.New(errorx.ErrCodeInvalidParam, "菜单 ID 无效")
	}
	err := s.repo.DeleteUserMenu(ctx, actor.Account, req.MenuID)
	if isNotFound(err) {
		return errorx.New(errorx.ErrCodeNotFound, "未固定该菜单")
	}
	return err
}

// ReorderUserMenus 按 menuIds 顺序重写 sort；未出现在请求中的已固定项保持相对顺序排在末尾。
func (s *Service) ReorderUserMenus(ctx context.Context, actor *model.User, perms map[string]bool, req ReorderUserMenuReq) (ReorderUserMenuResp, error) {
	if actor == nil || strings.TrimSpace(actor.Account) == "" {
		return ReorderUserMenuResp{}, errorx.New(errorx.ErrCodeForbidden, "请先登录")
	}
	existing, err := s.repo.FindUserMenuIDsByAccount(ctx, actor.Account)
	if err != nil {
		return ReorderUserMenuResp{}, err
	}
	owned := make(map[uint64]struct{}, len(existing))
	for _, id := range existing {
		owned[id] = struct{}{}
	}
	inReq := make(map[uint64]struct{}, len(req.MenuIDs))
	for _, id := range req.MenuIDs {
		if _, ok := owned[id]; !ok {
			return ReorderUserMenuResp{}, errorx.New(errorx.ErrCodeInvalidParam, "含有未固定的菜单")
		}
		inReq[id] = struct{}{}
	}
	// 最终顺序 = 请求顺序 + 未提及的已固定项（保持原相对序）
	ordered := make([]uint64, 0, len(existing))
	ordered = append(ordered, req.MenuIDs...)
	for _, id := range existing {
		if _, ok := inReq[id]; !ok {
			ordered = append(ordered, id)
		}
	}
	// 列表按 sort DESC 展示：数组首位 sort 最大
	n := len(ordered)
	sorts := make(map[uint64]int, n)
	for i, id := range ordered {
		sorts[id] = n - i
	}
	if err := s.repo.UpdateUserMenuSorts(ctx, actor.Account, actor.Account, sorts); err != nil {
		return ReorderUserMenuResp{}, err
	}
	list, err := s.ListUserMenus(ctx, actor, perms)
	if err != nil {
		return ReorderUserMenuResp{}, err
	}
	return ReorderUserMenuResp{Items: list.Items}, nil
}

func (s *Service) resolveMenuByPath(ctx context.Context, path string, perms map[string]bool, isSuperAdmin bool) (*model.Menu, error) {
	path = strings.TrimSpace(path)
	if path == "" || path[0] != '/' || path == "/home" {
		return nil, errorx.New(errorx.ErrCodeInvalidParam, "当前页面不可固定")
	}
	menus, err := s.repo.FindPinnableMenus(ctx)
	if err != nil {
		return nil, err
	}
	var best *model.Menu
	bestLen := -1
	for i := range menus {
		m := &menus[i]
		mp := strings.TrimSpace(m.Path)
		if mp == "" {
			continue
		}
		if !menuPermAllowed(perms, isSuperAdmin, m.Perm) {
			continue
		}
		if path == mp || strings.HasPrefix(path, mp+"/") {
			if len(mp) > bestLen {
				best = m
				bestLen = len(mp)
			}
		}
	}
	if best == nil {
		return nil, errorx.New(errorx.ErrCodeNotFound, "未找到可固定的菜单")
	}
	return best, nil
}

func menuPermAllowed(perms map[string]bool, isSuperAdmin bool, permCode string) bool {
	if isSuperAdmin {
		return true
	}
	code := strings.TrimSpace(permCode)
	if code == "" {
		return true
	}
	if perms == nil {
		return false
	}
	return perms[code]
}
