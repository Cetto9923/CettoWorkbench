// =============================================================================
// 文件: internal/module/schedule/service_window.go
// 模块: 排期工作台
// 类型: action
// 职责: 版本窗口卡片与维护列表查询。
// 依赖: internal/model
//       internal/module/schedule/repo.go
//       go.uber.org/zap
// =============================================================================

package schedule

import (
	"context"
	"fmt"
	"math"
	"strings"

	"go.uber.org/zap"

	"workbench/internal/model"
)

var windowCardToneClasses = []string{"red", "blue", "green", "purple"}

// ListWindowCards 查询版本窗口概览卡片数据。
func (s *Service) ListWindowCards(ctx context.Context, actor *model.User) ([]WindowCard, error) {
	account := actorAccount(actor)
	windows, _, err := s.repo.FindAll(ctx)
	if err != nil {
		return nil, err
	}
	if len(windows) == 0 {
		return []WindowCard{}, nil
	}

	teamgroupIDs := make([]uint, 0, len(windows))
	seen := make(map[uint]struct{}, len(windows))
	for _, window := range windows {
		if window.TeamgroupID == 0 {
			continue
		}
		if _, ok := seen[window.TeamgroupID]; ok {
			continue
		}
		seen[window.TeamgroupID] = struct{}{}
		teamgroupIDs = append(teamgroupIDs, window.TeamgroupID)
	}

	teamgroupNameByID := make(map[uint]string, len(teamgroupIDs))
	if len(teamgroupIDs) > 0 {
		groups, err := s.repo.FindTeamgroupsByIDs(ctx, teamgroupIDs)
		if err != nil {
			return nil, err
		}
		parentIDs := make([]uint, 0)
		parentSeen := make(map[uint]struct{})
		for _, group := range groups {
			teamgroupNameByID[group.ID] = strings.TrimSpace(group.Name)
			if group.Parent == 0 {
				continue
			}
			if _, ok := parentSeen[group.Parent]; ok {
				continue
			}
			parentSeen[group.Parent] = struct{}{}
			parentIDs = append(parentIDs, group.Parent)
		}
		if len(parentIDs) > 0 {
			parents, err := s.repo.FindTeamgroupsByIDs(ctx, parentIDs)
			if err != nil {
				return nil, err
			}
			parentNameByID := make(map[uint]string, len(parents))
			for _, parent := range parents {
				parentNameByID[parent.ID] = strings.TrimSpace(parent.Name)
			}
			for _, group := range groups {
				name := strings.TrimSpace(group.Name)
				if group.Parent > 0 {
					if parentName := parentNameByID[group.Parent]; parentName != "" {
						name = fmt.Sprintf("%s / %s", parentName, name)
					}
				}
				teamgroupNameByID[group.ID] = name
			}
		}
	}

	cards := make([]WindowCard, 0, len(windows))
	for i, window := range windows {
		start := window.ReleaseDate
		if window.StartDate != nil {
			start = *window.StartDate
		}
		capacityHours, err := s.CalcCapacity(
			ctx,
			start.Format("2006-01-02"),
			window.ReleaseDate.Format("2006-01-02"),
			int(window.GroupSize),
		)
		if err != nil {
			return nil, err
		}

		consumed, err := s.repo.GetWindowConsumedHours(ctx, window.ID)
		if err != nil {
			return nil, err
		}
		demandCount, err := s.repo.GetWindowDemandCount(ctx, window.ID)
		if err != nil {
			return nil, err
		}

		usedHours := int(math.Round(consumed))
		remainingHours := capacityHours - usedHours
		usedPercent := 0
		if capacityHours > 0 {
			usedPercent = usedHours * 100 / capacityHours
		}

		canEdit, canDelete, hasLinkedDemands := computeWindowPermissions(window.CreatedBy, account, demandCount)

		cards = append(cards, WindowCard{
			ID:               window.ID,
			ShortName:        window.Name,
			Range:            formatWindowDateRange(start, window.ReleaseDate),
			ToneClass:        windowCardToneClasses[i%len(windowCardToneClasses)],
			AgileGroup:       teamgroupNameByID[window.TeamgroupID],
			DemandCount:      demandCount,
			CapacityHours:    capacityHours,
			UsedHours:        usedHours,
			RemainingHours:   remainingHours,
			BlockedCount:     0,
			UsedPercent:      usedPercent,
			CanEdit:          canEdit,
			CanDelete:        canDelete,
			HasLinkedDemands: hasLinkedDemands,
		})
	}
	return cards, nil
}

// HomeVersionWindowCard PO 首页版本窗口卡片展示数据。
type HomeVersionWindowCard struct {
	Name         string
	AgileGroup   string
	Range        string
	DemandCount  int
	DevCount     int
	TestCount    int
	DeliverCount int
}

// ListHomeVersionWindows 查询 PO 首页近期版本窗口（最多 4 条，按用户敏捷小组过滤）。
func (s *Service) ListHomeVersionWindows(ctx context.Context, actor *model.User) ([]HomeVersionWindowCard, error) {
	account := actorAccount(actor)
	if account == "" {
		s.logHomeVersionWindows(account, nil, "account empty", 0, nil)
		return []HomeVersionWindowCard{}, nil
	}

	teamgroups, err := s.GetUserTeamgroups(ctx, account)
	if err != nil {
		return nil, err
	}
	teamgroupIDs := make([]uint, 0, len(teamgroups))
	for _, group := range teamgroups {
		if group.ID == 0 {
			continue
		}
		teamgroupIDs = append(teamgroupIDs, group.ID)
	}
	if len(teamgroupIDs) == 0 {
		s.logHomeVersionWindows(account, teamgroupIDs, "deletedAt IS NULL AND releaseDate>=CURDATE() AND teamgroup IN (...)", 0, nil)
		return []HomeVersionWindowCard{}, nil
	}

	windows, err := s.repo.ListUpcomingVersionWindowsForTeamgroups(ctx, teamgroupIDs, 4)
	if err != nil {
		return nil, err
	}
	s.logHomeVersionWindows(account, teamgroupIDs, "deletedAt IS NULL AND releaseDate>=CURDATE() AND teamgroup IN (...)", len(windows), windows)
	if len(windows) == 0 {
		return []HomeVersionWindowCard{}, nil
	}

	nameByID, err := s.loadTeamgroupDisplayNames(ctx, windows)
	if err != nil {
		return nil, err
	}

	cards := make([]HomeVersionWindowCard, 0, len(windows))
	for _, window := range windows {
		start := window.ReleaseDate
		if window.StartDate != nil {
			start = *window.StartDate
		}
		stats, err := s.repo.GetWindowStageStats(ctx, window.ID)
		if err != nil {
			return nil, err
		}
		cards = append(cards, HomeVersionWindowCard{
			Name:         window.Name,
			AgileGroup:   nameByID[window.TeamgroupID],
			Range:        formatWindowDateRange(start, window.ReleaseDate),
			DemandCount:  stats.DemandCount,
			DevCount:     stats.DevCount,
			TestCount:    stats.TestCount,
			DeliverCount: stats.DeliverCount,
		})
	}
	return cards, nil
}

func (s *Service) logHomeVersionWindows(account string, teamgroupIDs []uint, sqlCondition string, resultCount int, windows []model.VersionWindow) {
	if s.logger == nil {
		return
	}
	fields := []zap.Field{
		zap.String("account", account),
		zap.Uint64s("teamgroup_ids", uintsToUint64s(teamgroupIDs)),
		zap.String("sql_condition", sqlCondition),
		zap.Int("result_count", resultCount),
	}
	if len(windows) > 0 {
		ids := make([]uint64, 0, len(windows))
		names := make([]string, 0, len(windows))
		for _, window := range windows {
			ids = append(ids, window.ID)
			names = append(names, window.Name)
		}
		fields = append(fields, zap.Uint64s("window_ids", ids), zap.Strings("window_names", names))
	}
	s.logger.Info("home version windows query", fields...)
}

func uintsToUint64s(values []uint) []uint64 {
	out := make([]uint64, len(values))
	for i, value := range values {
		out[i] = uint64(value)
	}
	return out
}

func (s *Service) loadTeamgroupDisplayNames(ctx context.Context, windows []model.VersionWindow) (map[uint]string, error) {
	teamgroupIDs := make([]uint, 0, len(windows))
	seen := make(map[uint]struct{}, len(windows))
	for _, window := range windows {
		if window.TeamgroupID == 0 {
			continue
		}
		if _, ok := seen[window.TeamgroupID]; ok {
			continue
		}
		seen[window.TeamgroupID] = struct{}{}
		teamgroupIDs = append(teamgroupIDs, window.TeamgroupID)
	}
	if len(teamgroupIDs) == 0 {
		return map[uint]string{}, nil
	}

	groups, err := s.repo.FindTeamgroupsByIDs(ctx, teamgroupIDs)
	if err != nil {
		return nil, err
	}
	nameByID := make(map[uint]string, len(groups))
	parentIDs := make([]uint, 0)
	parentSeen := make(map[uint]struct{})
	for _, group := range groups {
		nameByID[group.ID] = strings.TrimSpace(group.Name)
		if group.Parent == 0 {
			continue
		}
		if _, ok := parentSeen[group.Parent]; ok {
			continue
		}
		parentSeen[group.Parent] = struct{}{}
		parentIDs = append(parentIDs, group.Parent)
	}
	if len(parentIDs) > 0 {
		parents, err := s.repo.FindTeamgroupsByIDs(ctx, parentIDs)
		if err != nil {
			return nil, err
		}
		parentNameByID := make(map[uint]string, len(parents))
		for _, parent := range parents {
			parentNameByID[parent.ID] = strings.TrimSpace(parent.Name)
		}
		for _, group := range groups {
			name := strings.TrimSpace(group.Name)
			if group.Parent > 0 {
				if parentName := parentNameByID[group.Parent]; parentName != "" {
					name = fmt.Sprintf("%s / %s", parentName, name)
				}
			}
			nameByID[group.ID] = name
		}
	}
	return nameByID, nil
}

// ListWindows 查询版本窗口维护列表。
func (s *Service) ListWindows(ctx context.Context, actor *model.User) (ListWindowsResp, error) {
	account := actorAccount(actor)
	windows, _, err := s.repo.FindAll(ctx)
	if err != nil {
		return ListWindowsResp{}, err
	}
	if len(windows) == 0 {
		return ListWindowsResp{Windows: []WindowListItem{}}, nil
	}

	items := make([]WindowListItem, 0, len(windows))
	for _, window := range windows {
		start := window.ReleaseDate
		if window.StartDate != nil {
			start = *window.StartDate
		}
		capacityHours, err := s.CalcCapacity(
			ctx,
			start.Format("2006-01-02"),
			window.ReleaseDate.Format("2006-01-02"),
			int(window.GroupSize),
		)
		if err != nil {
			return ListWindowsResp{}, err
		}
		consumed, err := s.repo.GetWindowConsumedHours(ctx, window.ID)
		if err != nil {
			return ListWindowsResp{}, err
		}
		demandCount, err := s.repo.GetWindowDemandCount(ctx, window.ID)
		if err != nil {
			return ListWindowsResp{}, err
		}
		usedHours := int(math.Round(consumed))
		remainingHours := capacityHours - usedHours
		usedPercent := 0
		if capacityHours > 0 {
			usedPercent = usedHours * 100 / capacityHours
		}
		canEdit, canDelete, hasLinkedDemands := computeWindowPermissions(window.CreatedBy, account, demandCount)

		items = append(items, WindowListItem{
			ID:               window.ID,
			Name:             window.Name,
			ReleaseDate:      window.ReleaseDate.Format("2006-01-02"),
			Range:            formatWindowDateRange(start, window.ReleaseDate),
			DemandCount:      demandCount,
			CapacityHours:    capacityHours,
			UsedHours:        usedHours,
			RemainingHours:   remainingHours,
			BlockedCount:     0,
			UsedPercent:      usedPercent,
			CanEdit:          canEdit,
			CanDelete:        canDelete,
			HasLinkedDemands: hasLinkedDemands,
		})
	}
	return ListWindowsResp{Windows: items}, nil
}
