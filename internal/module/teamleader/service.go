// =============================================================================
// 文件: internal/module/teamleader/service.go
// 模块: 团队长工作台
// 类型: service
// 职责: 团队层级拓扑组装、对象级鉴权控制与双重集合去重计算。
// 依赖: internal/model
//       internal/pkg/errorx
// =============================================================================

package teamleader

import (
	"context"
	"fmt"
	"strings"

	"workbench/internal/model"
	"workbench/internal/pkg/errorx"
)

// Service 团队长业务逻辑层。
type Service struct {
	repo *Repo
}

// NewService 创建 Service 实例。
func NewService(repo *Repo) *Service {
	return &Service{repo: repo}
}

// GetTeamHierarchy 获取团队与下属小组层级视图，实施严格的对象级权限与双重去重计算。
func (s *Service) GetTeamHierarchy(ctx context.Context, actor *model.User, req TeamHierarchyReq) (*TeamHierarchyResp, error) {
	if actor == nil || strings.TrimSpace(actor.Account) == "" {
		return nil, errorx.New(errorx.ErrCodeForbidden, "请先登录")
	}

	authorizedTeams, err := s.resolveAuthorizedTeams(ctx, actor)
	if err != nil {
		return nil, err
	}
	if len(authorizedTeams) == 0 {
		return &TeamHierarchyResp{
			CurrentTeam:     nil,
			SubGroups:       []SubGroupDTO{},
			AuthorizedTeams: []TeamOptionDTO{},
			MyRole:          "guest",
		}, nil
	}

	targetTeamID, err := s.validateAndSelectTeamID(req.TeamID, authorizedTeams)
	if err != nil {
		return nil, err
	}

	targetTeam, err := s.repo.FindParentTeamByID(ctx, targetTeamID)
	if err != nil {
		return nil, errorx.Wrap(errorx.ErrCodeInternal, "查询团队信息失败", err)
	}

	subGroupRaws, err := s.repo.FindSubGroupsByParentID(ctx, targetTeamID)
	if err != nil {
		return nil, errorx.Wrap(errorx.ErrCodeInternal, "查询子小组失败", err)
	}

	groupIDs, groupNames := extractGroupMeta(subGroupRaws)
	membersRaw, err := s.repo.FindMembersByGroupIDs(ctx, groupIDs)
	if err != nil {
		return nil, errorx.Wrap(errorx.ErrCodeInternal, "查询小组成员失败", err)
	}

	accountsToResolve := collectAccountsForNameResolution(targetTeam, subGroupRaws, membersRaw)
	nameMap, err := s.repo.ResolveRealnames(ctx, accountsToResolve)
	if err != nil {
		return nil, errorx.Wrap(errorx.ErrCodeInternal, "解析用户姓名失败", err)
	}

	teamSummary, subGroupDTOs := s.assembleHierarchy(actor.Account, actor.IsSuperAdmin, targetTeam, subGroupRaws, membersRaw, groupNames, nameMap)
	myRole := s.determineMyRole(actor.Account, actor.IsSuperAdmin, targetTeam, subGroupRaws, membersRaw)

	return &TeamHierarchyResp{
		CurrentTeam:     teamSummary,
		SubGroups:       subGroupDTOs,
		AuthorizedTeams: authorizedTeams,
		MyRole:          myRole,
	}, nil
}

// resolveAuthorizedTeams 解析当前账号有权查看的三级团队列表。
func (s *Service) resolveAuthorizedTeams(ctx context.Context, actor *model.User) ([]TeamOptionDTO, error) {
	allParents, err := s.repo.FindParentTeams(ctx)
	if err != nil {
		return nil, errorx.Wrap(errorx.ErrCodeInternal, "加载团队列表失败", err)
	}
	if actor.IsSuperAdmin {
		out := make([]TeamOptionDTO, 0, len(allParents))
		for _, p := range allParents {
			out = append(out, TeamOptionDTO{ID: p.ID, Name: p.Name})
		}
		return out, nil
	}

	userTeamIDs, err := s.repo.FindUserParentTeamIDsByAccount(ctx, actor.Account)
	if err != nil {
		return nil, errorx.Wrap(errorx.ErrCodeInternal, "检查用户关联团队失败", err)
	}
	idSet := make(map[uint]bool, len(userTeamIDs))
	for _, id := range userTeamIDs {
		idSet[id] = true
	}

	out := make([]TeamOptionDTO, 0, len(userTeamIDs))
	for _, p := range allParents {
		if idSet[p.ID] {
			out = append(out, TeamOptionDTO{ID: p.ID, Name: p.Name})
		}
	}
	return out, nil
}

// validateAndSelectTeamID 校验请求 teamId 是否在授权列表内，防止越权；teamId 为 0 时自动选首个。
func (s *Service) validateAndSelectTeamID(reqTeamID uint, authorized []TeamOptionDTO) (uint, error) {
	if reqTeamID == 0 {
		return authorized[0].ID, nil
	}
	for _, it := range authorized {
		if it.ID == reqTeamID {
			return reqTeamID, nil
		}
	}
	return 0, errorx.New(errorx.ErrCodeForbidden, "无权查看该团队")
}

func extractGroupMeta(subGroups []TeamgroupRaw) ([]uint, map[uint]string) {
	ids := make([]uint, 0, len(subGroups))
	names := make(map[uint]string, len(subGroups))
	for _, g := range subGroups {
		ids = append(ids, g.ID)
		names[g.ID] = g.Name
	}
	return ids, names
}

func collectAccountsForNameResolution(parent *TeamgroupRaw, subGroups []TeamgroupRaw, members []TeamMemberRaw) []string {
	seen := make(map[string]bool)
	collect := func(acc string) {
		acc = strings.TrimSpace(acc)
		if acc != "" && !seen[acc] {
			seen[acc] = true
		}
	}
	if parent != nil {
		collect(parent.Manager)
		collect(parent.PO)
	}
	for _, g := range subGroups {
		collect(g.Manager)
		collect(g.PO)
	}
	for _, m := range members {
		collect(m.Account)
	}
	out := make([]string, 0, len(seen))
	for acc := range seen {
		out = append(out, acc)
	}
	return out
}

// assembleHierarchy 执行核心集合去重算法与层级结构组装，并实施小组级成员明细访问控制。
func (s *Service) assembleHierarchy(
	actorAccount string,
	isSuperAdmin bool,
	parent *TeamgroupRaw,
	subGroups []TeamgroupRaw,
	members []TeamMemberRaw,
	groupNames map[uint]string,
	nameMap map[string]string,
) (*TeamSummaryDTO, []SubGroupDTO) {
	// 集合去重步骤 1: (groupId, account) 去重，过滤底层可能存在的重复记录
	seenGroupAccount := make(map[string]bool)
	cleanedGroupMembers := make(map[uint][]TeamMemberRaw)
	// 集合去重步骤 2: 构建 (account -> 参与子小组集合)，用于跨组标记
	accountGroupMap := make(map[string]map[uint]bool)

	for _, m := range members {
		key := fmt.Sprintf("%d_%s", m.GroupID, m.Account)
		if seenGroupAccount[key] {
			continue
		}
		seenGroupAccount[key] = true
		cleanedGroupMembers[m.GroupID] = append(cleanedGroupMembers[m.GroupID], m)

		if accountGroupMap[m.Account] == nil {
			accountGroupMap[m.Account] = make(map[uint]bool)
		}
		accountGroupMap[m.Account][m.GroupID] = true
	}

	// 集合去重步骤 3: 团队级唯一人员去重
	uniqueAccounts := make(map[string]bool)
	for acc := range accountGroupMap {
		uniqueAccounts[acc] = true
	}

	// 判断当前用户是否具备全团队管理权限 (超管 / 团队长 / 团队PO)
	hasTeamAdmin := isSuperAdmin ||
		(parent != nil && (parent.Manager == actorAccount || parent.PO == actorAccount))

	// 组装各个子小组 DTO
	subDTOs := make([]SubGroupDTO, 0, len(subGroups))
	for _, g := range subGroups {
		rawMembers := cleanedGroupMembers[g.ID]
		totalGroupCount := len(rawMembers)

		// 检查当前账号是否对当前子小组有明细查看权 (全团队管理权 / 该小组长 / 该小组PO / 该小组成员)
		canViewDetail := hasTeamAdmin ||
			g.Manager == actorAccount ||
			g.PO == actorAccount ||
			accountGroupMap[actorAccount][g.ID]

		memberDTOs := make([]TeamMemberDTO, 0)
		if canViewDetail {
			memberDTOs = make([]TeamMemberDTO, 0, totalGroupCount)
			for _, rm := range rawMembers {
				isMulti := len(accountGroupMap[rm.Account]) > 1
				otherNames := make([]string, 0)
				if isMulti {
					for gid := range accountGroupMap[rm.Account] {
						if gid != g.ID {
							otherNames = append(otherNames, groupNames[gid])
						}
					}
				}
				memberDTOs = append(memberDTOs, TeamMemberDTO{
					Account:         rm.Account,
					Name:            resolveName(rm.Account, nameMap),
					Role:            rm.Role,
					IsMultiGroup:    isMulti,
					OtherGroupNames: otherNames,
				})
			}
		}

		leaderName := resolveName(g.Manager, nameMap)
		if strings.TrimSpace(g.Manager) == "" {
			leaderName = "未配置"
		}
		poName := resolveName(g.PO, nameMap)
		if strings.TrimSpace(g.PO) == "" {
			poName = "未配置"
		}

		subDTOs = append(subDTOs, SubGroupDTO{
			ID:          g.ID,
			Name:        g.Name,
			GroupLeader: PersonDTO{Account: g.Manager, Name: leaderName},
			PO:          PersonDTO{Account: g.PO, Name: poName},
			ScrumMaster: PersonDTO{}, // 严格遵守业务规则：禅道无独立SM来源时置空，由前端统一展示“未配置”，禁止将小组长冒充SM
			MemberCount: totalGroupCount,
			Members:     memberDTOs,
		})
	}

	teamSummary := &TeamSummaryDTO{
		ID:                 parent.ID,
		Name:               parent.Name,
		Leader:             PersonDTO{Account: parent.Manager, Name: resolveName(parent.Manager, nameMap)},
		PO:                 PersonDTO{Account: parent.PO, Name: resolveName(parent.PO, nameMap)},
		TotalUniqueMembers: len(uniqueAccounts),
	}

	return teamSummary, subDTOs
}

func resolveName(acc string, nameMap map[string]string) string {
	acc = strings.TrimSpace(acc)
	if acc == "" {
		return ""
	}
	if name, ok := nameMap[acc]; ok && strings.TrimSpace(name) != "" {
		return name
	}
	return acc
}

func (s *Service) determineMyRole(
	account string,
	isSuperAdmin bool,
	parent *TeamgroupRaw,
	subGroups []TeamgroupRaw,
	members []TeamMemberRaw,
) string {
	if isSuperAdmin {
		return "admin"
	}
	if parent != nil && parent.Manager == account {
		return "team_leader"
	}
	if parent != nil && parent.PO == account {
		return "po"
	}
	for _, g := range subGroups {
		if g.Manager == account {
			return "group_leader"
		}
		if g.PO == account {
			return "po"
		}
	}
	for _, m := range members {
		if m.Account == account {
			return "member"
		}
	}
	return "viewer"
}
