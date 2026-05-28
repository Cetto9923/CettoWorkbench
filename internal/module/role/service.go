// =============================================================================
// 文件: internal/module/role/service.go
// 模块: 角色管理
// 类型: crud
// 职责: 角色 CRUD 与权限绑定替换的业务规则。
// 依赖: internal/model
//       internal/module/role/repo.go
// =============================================================================

package role

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"workbench/internal/model"
	"workbench/internal/pkg/perm"
)

// Service 处理角色业务逻辑。
type Service struct {
	repo *Repo
}

// NewService 创建 Service。
func NewService(repo *Repo) *Service {
	return &Service{repo: repo}
}

// List 分页查询角色列表。
func (s *Service) List(ctx context.Context, actor *model.User, req ListReq) (ListResp, error) {
	_ = actor
	req.Normalize()
	items, total, err := s.repo.FindAll(ctx, RepoFindAllReq{
		Keyword:  req.Keyword,
		Page:     req.Page,
		PageSize: req.PageSize,
	})
	if err != nil {
		return ListResp{}, err
	}
	return ListResp{Items: items, Total: total}, nil
}

// GetByID 按 ID 获取角色。
func (s *Service) GetByID(ctx context.Context, actor *model.User, id int64) (*model.Role, error) {
	_ = actor
	return s.repo.FindByID(ctx, id)
}

// Create 创建自定义角色。
func (s *Service) Create(ctx context.Context, actor *model.User, req CreateReq) (CreateResp, error) {
	tenantID := int64(0)
	if exists, err := s.repo.ExistsByName(ctx, tenantID, req.Name, 0); err != nil {
		return CreateResp{}, err
	} else if exists {
		return CreateResp{}, errors.New("角色名称已存在")
	}

	uid := int64(0)
	if actor != nil {
		uid = actor.ID
	}

	m := &model.Role{
		Code:      fmt.Sprintf("r%x", time.Now().UnixNano()),
		Name:      strings.TrimSpace(req.Name),
		Remark:    strings.TrimSpace(req.Remark),
		IsBuiltin: false,
		IsActive:  true,
		SortOrder: 30,
	}
	m.CreatedBy = strconv.FormatInt(uid, 10)
	m.UpdatedBy = strconv.FormatInt(uid, 10)

	if err := s.repo.Create(ctx, m); err != nil {
		return CreateResp{}, err
	}
	return CreateResp{ID: m.ID}, nil
}

// Update 更新非内置角色的名称与备注。
func (s *Service) Update(ctx context.Context, actor *model.User, req UpdateReq) error {
	role, err := s.repo.FindByID(ctx, req.ID)
	if err != nil {
		return err
	}
	if role.IsBuiltin {
		return errors.New("内置角色不可修改")
	}

	tenantID := int64(0)
	if exists, err := s.repo.ExistsByName(ctx, tenantID, req.Name, req.ID); err != nil {
		return err
	} else if exists {
		return errors.New("角色名称已存在")
	}

	uid := int64(0)
	if actor != nil {
		uid = actor.ID
	}

	role.Name = strings.TrimSpace(req.Name)
	role.Remark = strings.TrimSpace(req.Remark)
	role.UpdatedBy = strconv.FormatInt(uid, 10)
	role.UpdatedDate = time.Now()

	return s.repo.Update(ctx, role)
}

// Delete 删除非内置且无用户关联的角色。
func (s *Service) Delete(ctx context.Context, actor *model.User, req DeleteReq) error {
	_ = actor
	role, err := s.repo.FindByID(ctx, req.ID)
	if err != nil {
		return err
	}
	if role.IsBuiltin {
		return errors.New("内置角色不可删除")
	}
	has, err := s.repo.HasUsers(ctx, req.ID)
	if err != nil {
		return err
	}
	if has {
		return errors.New("角色下仍有用户，无法删除")
	}
	return s.repo.Delete(ctx, req.ID)
}

// GetPermissions 查询角色已绑定的权限码。
func (s *Service) GetPermissions(ctx context.Context, actor *model.User, roleID int64) ([]string, error) {
	_ = actor
	if _, err := s.repo.FindByID(ctx, roleID); err != nil {
		return nil, err
	}
	return s.repo.ListPermissions(ctx, roleID)
}

// AssignPermsFormData 组装权限分配页所需菜单树及勾选状态。
func (s *Service) AssignPermsFormData(ctx context.Context, actor *model.User, req AssignPermsFormDataReq) (AssignPermsFormDataResp, error) {
	_ = actor
	selected, err := s.repo.ListPermissions(ctx, req.RoleID)
	if err != nil {
		return AssignPermsFormDataResp{}, err
	}
	allPerms := perm.Configurable()
	menus, err := s.repo.ListAllMenus(ctx)
	if err != nil {
		return AssignPermsFormDataResp{}, err
	}
	tree := buildPermTreeFromMenus(menus, allPerms, selected)
	return AssignPermsFormDataResp{PermTree: tree}, nil
}

// ReplacePermissions 全量替换角色权限绑定。
func (s *Service) ReplacePermissions(ctx context.Context, actor *model.User, roleID int64, permCodes []string) error {
	_ = actor
	if _, err := s.repo.FindByID(ctx, roleID); err != nil {
		return err
	}
	return s.repo.ReplacePermissions(ctx, roleID, permCodes)
}

func buildPermTreeFromMenus(menus []model.Menu, allPerms []perm.PermInfo, selected []string) []*PermTreeNode {
	selectedSet := make(map[string]bool, len(selected))
	for _, c := range selected {
		c = strings.TrimSpace(c)
		if c != "" {
			selectedSet[c] = true
		}
	}

	byParent := make(map[uint64][]model.Menu)
	for _, m := range menus {
		byParent[m.ParentID] = append(byParent[m.ParentID], m)
	}
	for _, list := range byParent {
		sort.SliceStable(list, func(i, j int) bool {
			if list[i].Sort != list[j].Sort {
				return list[i].Sort < list[j].Sort
			}
			return list[i].ID < list[j].ID
		})
	}

	permsByModule := make(map[string][]perm.PermInfo)
	for _, p := range allPerms {
		mod := strings.TrimSpace(p.Module)
		if mod == "" {
			mod = moduleKeyFromPermCode(p.Code.String())
		}
		permsByModule[mod] = append(permsByModule[mod], p)
	}
	for mod, list := range permsByModule {
		sort.SliceStable(list, func(i, j int) bool {
			return strings.Compare(list[i].Code.String(), list[j].Code.String()) < 0
		})
		permsByModule[mod] = list
	}

	used := make(map[string]bool)
	roots := byParent[0]
	out := make([]*PermTreeNode, 0, len(roots))
	for i := range roots {
		out = append(out, menuToPermTreeNode(&roots[i], byParent, permsByModule, used, selectedSet, 0, ""))
	}

	var orphanPerms []perm.PermInfo
	for mod, list := range permsByModule {
		if used[mod] || len(list) == 0 {
			continue
		}
		orphanPerms = append(orphanPerms, list...)
	}
	if len(orphanPerms) > 0 {
		sort.SliceStable(orphanPerms, func(i, j int) bool {
			return strings.Compare(orphanPerms[i].Code.String(), orphanPerms[j].Code.String()) < 0
		})
		other := &PermTreeNode{
			ID:                "m-other",
			Name:              "其他权限",
			Depth:             0,
			SortOrder:         0,
			NumericID:         0,
			HasBranchChildren: false,
		}
		for _, p := range orphanPerms {
			other.Children = append(other.Children, permLeafNode(p, selectedSet, other.ID, other.SortOrder))
		}
		out = append(out, other)
	}
	sortPermTreeNodes(out)
	out = pruneEmptyLeaves(out)
	out = promoteOnlyChild(out)
	return out
}

func menuToPermTreeNode(m *model.Menu, byParent map[uint64][]model.Menu, permsByModule map[string][]perm.PermInfo, used map[string]bool, selected map[string]bool, depth int, parentMenuID string) *PermTreeNode {
	kids := byParent[m.ID]
	node := &PermTreeNode{
		ID:                fmt.Sprintf("m-%d", m.ID),
		ParentID:          parentMenuID,
		Name:              m.Title,
		Depth:             depth,
		SortOrder:         m.Sort,
		NumericID:         int64(m.ID),
		HasBranchChildren: len(kids) > 0,
	}
	for i := range kids {
		node.Children = append(node.Children, menuToPermTreeNode(&kids[i], byParent, permsByModule, used, selected, depth+1, node.ID))
	}
	mod := moduleKeyFromMenuPerm(m.Perm)
	if mod != "" && !used[mod] {
		if list, ok := permsByModule[mod]; ok && len(list) > 0 {
			used[mod] = true
			for _, p := range list {
				node.Children = append(node.Children, permLeafNode(p, selected, node.ID, node.SortOrder))
			}
		}
	}
	return node
}

func permLeafNode(p perm.PermInfo, selected map[string]bool, parentMenuID string, parentSort int) *PermTreeNode {
	code := p.Code.String()
	safeID := "p-" + strings.ReplaceAll(strings.ReplaceAll(code, ":", "-"), "/", "-")
	return &PermTreeNode{
		ID:        safeID,
		ParentID:  parentMenuID,
		Name:      p.Name,
		Code:      code,
		Checked:   selected[code],
		SortOrder: parentSort,
		NumericID: 0,
	}
}

// pruneEmptyLeaves 递归删除空叶子菜单节点。
// 空叶子定义：菜单节点（Code 为空）且没有菜单子节点也没有权限码子节点。
func pruneEmptyLeaves(nodes []*PermTreeNode) []*PermTreeNode {
	if len(nodes) == 0 {
		return nodes
	}
	result := make([]*PermTreeNode, 0, len(nodes))
	for _, n := range nodes {
		n.Children = pruneEmptyLeaves(n.Children)
		hasBranch := hasBranchChild(n)
		hasPerm := hasPermChild(n)
		if n.Code == "" && !hasBranch && !hasPerm {
			continue
		}
		n.HasBranchChildren = hasBranch
		result = append(result, n)
	}
	return result
}

// promoteOnlyChild 将独生子菜单节点的权限码提升到父节点。
// 仅在父节点无权限码、仅有一个菜单子节点且该菜单子节点不再包含菜单子节点时进行提升。
func promoteOnlyChild(nodes []*PermTreeNode) []*PermTreeNode {
	if len(nodes) == 0 {
		return nodes
	}
	for _, n := range nodes {
		n.Children = promoteOnlyChild(n.Children)

		var branchChildren []*PermTreeNode
		var permChildren []*PermTreeNode
		for _, c := range n.Children {
			if c.Code == "" {
				branchChildren = append(branchChildren, c)
			} else {
				permChildren = append(permChildren, c)
			}
		}

		if n.Code == "" && len(permChildren) == 0 && len(branchChildren) == 1 {
			only := branchChildren[0]
			if !hasBranchChild(only) {
				n.Children = only.Children
				for _, c := range n.Children {
					c.ParentID = n.ID
					c.Depth = n.Depth + 1
				}
				n.HasBranchChildren = hasBranchChild(n)
				continue
			}
		}

		n.HasBranchChildren = hasBranchChild(n)
	}
	return nodes
}

func hasBranchChild(n *PermTreeNode) bool {
	for _, c := range n.Children {
		if c.Code == "" {
			return true
		}
	}
	return false
}

func hasPermChild(n *PermTreeNode) bool {
	for _, c := range n.Children {
		if c.Code != "" {
			return true
		}
	}
	return false
}

func sortPermTreeNodes(nodes []*PermTreeNode) {
	if len(nodes) == 0 {
		return
	}
	sort.SliceStable(nodes, func(i, j int) bool {
		a := nodes[i]
		b := nodes[j]
		aPriority := permCodePriority(a.Code)
		bPriority := permCodePriority(b.Code)
		if aPriority != bPriority {
			return aPriority < bPriority
		}
		if a.SortOrder != b.SortOrder {
			return a.SortOrder < b.SortOrder
		}
		if a.NumericID != b.NumericID {
			return a.NumericID < b.NumericID
		}
		return strings.Compare(a.ID, b.ID) < 0
	})
	for _, n := range nodes {
		sortPermTreeNodes(n.Children)
	}
}

func permCodePriority(code string) int {
	code = strings.TrimSpace(code)
	if code == "" {
		return 1
	}
	if strings.HasSuffix(code, ":list") || strings.HasSuffix(code, ":query") {
		return 0
	}
	return 1
}

func moduleKeyFromMenuPerm(perm string) string {
	perm = strings.TrimSpace(perm)
	if perm == "" {
		return ""
	}
	mod, _, ok := strings.Cut(perm, ":")
	if !ok {
		return ""
	}
	return strings.TrimSpace(mod)
}

func moduleKeyFromPermCode(code string) string {
	mod, _, ok := strings.Cut(strings.TrimSpace(code), ":")
	if !ok {
		return ""
	}
	return strings.TrimSpace(mod)
}
