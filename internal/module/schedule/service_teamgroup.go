// =============================================================================
// 文件: internal/module/schedule/service_teamgroup.go
// 模块: 排期工作台
// 类型: action
// 职责: 敏捷小组及关联展示名称查询。
// 依赖: context, fmt, strings, internal/module/schedule/repo.go
// =============================================================================

package schedule

import (
	"context"
	"fmt"
	"strings"
)

// GetUserTeamgroups 查询用户所属敏捷小组并拼接展示名称。
func (s *Service) GetUserTeamgroups(ctx context.Context, account string) ([]TeamgroupOption, error) {
	account = strings.TrimSpace(account)
	if account == "" {
		return []TeamgroupOption{}, nil
	}

	isAdmin, err := s.repo.IsAdmin(ctx, account)
	if err != nil {
		return nil, err
	}

	var groups []ZtTeamgroup
	if isAdmin {
		groups, err = s.repo.ListAllTeamgroups(ctx)
	} else {
		groups, err = s.repo.GetUserTeamgroups(ctx, account)
	}
	if err != nil {
		return nil, err
	}
	if len(groups) == 0 {
		return []TeamgroupOption{}, nil
	}

	parentIDs := make([]uint, 0)
	parentSeen := make(map[uint]struct{})
	for _, group := range groups {
		if group.Parent == 0 {
			continue
		}
		if _, ok := parentSeen[group.Parent]; ok {
			continue
		}
		parentSeen[group.Parent] = struct{}{}
		parentIDs = append(parentIDs, group.Parent)
	}

	parentNameByID := make(map[uint]string, len(parentIDs))
	if len(parentIDs) > 0 {
		parents, err := s.repo.FindTeamgroupsByIDs(ctx, parentIDs)
		if err != nil {
			return nil, err
		}
		for _, parent := range parents {
			parentNameByID[parent.ID] = strings.TrimSpace(parent.Name)
		}
	}

	options := make([]TeamgroupOption, 0, len(groups))
	for _, group := range groups {
		displayName := strings.TrimSpace(group.Name)
		if group.Parent > 0 {
			parentName := parentNameByID[group.Parent]
			if parentName != "" {
				displayName = fmt.Sprintf("%s / %s", parentName, displayName)
			}
		}
		options = append(options, TeamgroupOption{
			ID:          group.ID,
			DisplayName: displayName,
		})
	}
	return options, nil
}
