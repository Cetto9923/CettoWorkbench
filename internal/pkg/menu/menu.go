// =============================================================================
// 文件: internal/pkg/menu/menu.go
// 模块: 基础设施
// 类型: infra
// 职责: 加载菜单配置并按权限过滤菜单。
// 依赖: 无
// =============================================================================

package menu

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"workbench/internal/model"

	"gorm.io/gorm"
)

// ContextActiveNavKey Gin 上下文键：middleware.ActiveNav 写入的当前导航键（与 Menu.Key 同源，由菜单 path 经 KeyFromPath 得到）。
const ContextActiveNavKey = "activeNavKey"

// KeyFromPath 将菜单 path 转为与 buildMenus 中 Menu.Key 一致的键（与 model 行上 menuKey 含 path 时规则相同）。
func KeyFromPath(path string) string {
	p := strings.TrimSpace(path)
	if p == "" {
		return ""
	}
	return strings.Trim(strings.ReplaceAll(p, "/", "_"), "_")
}

// Menu 表示侧边栏菜单节点。
type Menu struct {
	Key      string `yaml:"key"`
	Title    string `yaml:"title"`
	Icon     string `yaml:"icon"`
	Path     string `yaml:"path"`
	Type     string `yaml:"type"`
	Perm     string `yaml:"perm"`
	Order    int    `yaml:"order"`
	Children []Menu `yaml:"children"`
}

// LoadFromDB 从 zt_menus 加载菜单并组装树形结构。
func LoadFromDB(ctx context.Context, db *gorm.DB) ([]Menu, error) {
	if db == nil {
		return []Menu{}, nil
	}
	var rows []model.Menu
	if err := db.WithContext(ctx).
		Model(&model.Menu{}).
		Where("deletedAt IS NULL").
		Order("sort ASC, id ASC").
		Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("find menus from db: %w", err)
	}
	return buildMenus(rows), nil
}

// Filter 按权限过滤菜单。
func Filter(menus []Menu, userPerms map[string]bool, isSuperAdmin bool) []Menu {
	if isSuperAdmin {
		out := make([]Menu, len(menus))
		copy(out, menus)
		return out
	}
	result := make([]Menu, 0, len(menus))
	for _, m := range menus {
		if m.Perm != "" && !userPerms[m.Perm] {
			continue
		}
		node := m
		if len(node.Children) > 0 {
			node.Children = Filter(node.Children, userPerms, false)
		}
		if len(node.Children) == 0 && node.Path == "" {
			continue
		}
		result = append(result, node)
	}
	return result
}

func sortMenus(menus []Menu) {
	sort.SliceStable(menus, func(i, j int) bool {
		if menus[i].Order == menus[j].Order {
			return menus[i].Key < menus[j].Key
		}
		return menus[i].Order < menus[j].Order
	})
	for i := range menus {
		if len(menus[i].Children) > 0 {
			sortMenus(menus[i].Children)
		}
	}
}

func buildMenus(rows []model.Menu) []Menu {
	if len(rows) == 0 {
		return []Menu{}
	}
	byParent := make(map[uint64][]model.Menu)
	for _, row := range rows {
		byParent[row.ParentID] = append(byParent[row.ParentID], row)
	}
	var walk func(parentID uint64) []Menu
	walk = func(parentID uint64) []Menu {
		items := byParent[parentID]
		menus := make([]Menu, 0, len(items))
		for _, item := range items {
			node := Menu{
				Key:      menuKey(item),
				Title:    item.Title,
				Icon:     item.Icon,
				Path:     item.Path,
				Type:     mapMenuType(item.Type),
				Perm:     item.Perm,
				Order:    item.Sort,
				Children: walk(item.ID),
			}
			menus = append(menus, node)
		}
		sortMenus(menus)
		return menus
	}
	return walk(0)
}

func menuKey(m model.Menu) string {
	if strings.TrimSpace(m.Path) != "" {
		return KeyFromPath(m.Path)
	}
	if m.Perm != "" {
		return strings.ReplaceAll(m.Perm, ":", "_")
	}
	return fmt.Sprintf("menu_%d", m.ID)
}

func mapMenuType(raw string) string {
	switch strings.ToUpper(strings.TrimSpace(raw)) {
	case "M":
		return "directory"
	case "F":
		return "button"
	case "A":
		return "action"
	default:
		return "menu"
	}
}

// MenuNavActive 供模板使用：当前请求的 ActiveNavKey（与 Menu.Key 一致）是否落在该节点或其子孙菜单上，用于侧栏/顶栏高亮与顶栏二级面板显隐。
func MenuNavActive(activeKey interface{}, node interface{}) bool {
	k := strings.TrimSpace(fmt.Sprintf("%v", activeKey))
	if k == "" {
		return false
	}
	m, ok := node.(Menu)
	if !ok {
		return false
	}
	return menuNavActiveByKey(k, m)
}

func menuNavActiveByKey(k string, m Menu) bool {
	if m.Type != "menu" && m.Type != "directory" {
		return false
	}
	if m.Key == k {
		return true
	}
	for i := range m.Children {
		if menuNavActiveByKey(k, m.Children[i]) {
			return true
		}
	}
	return false
}
