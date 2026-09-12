// =============================================================================
// 文件: internal/module/agileteam/service.go
// 模块: 敏捷小组治理
// 类型: action
// 职责: 列表 / 详情 / 合并成员池编排。
// 依赖: internal/model
//       internal/pkg/errorx
// =============================================================================

package agileteam

import (
	"context"
	"errors"
	"strings"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"workbench/internal/model"
	"workbench/internal/pkg/errorx"
)

// Service 敏捷小组业务。
type Service struct {
	repo   *Repo
	logger *zap.Logger
}

// NewService 创建 Service。
func NewService(repo *Repo, logger *zap.Logger) *Service {
	return &Service{repo: repo, logger: logger}
}

func firstAccount(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	// manager 可能是逗号分隔或多账号，取第一个。
	for _, sep := range []string{",", ";", "\n", " "} {
		if strings.Contains(raw, sep) {
			parts := strings.Split(raw, sep)
			for _, p := range parts {
				p = strings.TrimSpace(p)
				if p != "" {
					return p
				}
			}
		}
	}
	return raw
}

// Detail 小组详情。
func (s *Service) Detail(ctx context.Context, actor *model.User, id uint, canConfirm, canEdit bool) (*DetailResp, error) {
	row, err := s.repo.FindTeamgroupByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errorx.New("not_found", "敏捷小组不存在")
		}
		return nil, err
	}
	formalRows, err := s.repo.ListMembers(ctx, id)
	if err != nil {
		return nil, err
	}
	pendingAdj, pendErr := s.repo.FindPendingByTeamgroup(ctx, id)
	var pending *PendingSummary
	pendingRemove := map[string]bool{}
	pendingJoin := []MemberItem{}
	if pendErr == nil && pendingAdj != nil {
		items, err := s.repo.ListItems(ctx, pendingAdj.ID)
		if err != nil {
			return nil, err
		}
		pending = buildPendingSummary(pendingAdj, items)
		accs := []string{pendingAdj.SubmittedBy}
		for _, it := range items {
			accs = append(accs, it.Account)
		}
		names, _ := s.repo.ResolveRealnames(ctx, accs)
		pending.SubmittedBy = displayName(pendingAdj.SubmittedBy, names)
		for _, it := range items {
			switch it.ActionType {
			case ActionAdd:
				pendingJoin = append(pendingJoin, MemberItem{
					Account: it.Account, Name: displayName(it.Account, names),
					Role: it.Role, Hours: it.AvailableHours,
					Status: "pendingAdd", Submitter: displayName(pendingAdj.SubmittedBy, names),
				})
			case ActionRemove:
				pendingRemove[it.Account] = true
			}
		}
		pending.AddNames = nameList(items, ActionAdd, names)
		pending.RemoveNames = nameList(items, ActionRemove, names)
		pending.ChangeNames = nameList(items, ActionRoleChange, names)
	} else if pendErr != nil && !errors.Is(pendErr, gorm.ErrRecordNotFound) {
		return nil, pendErr
	}

	formal := make([]MemberItem, 0, len(formalRows))
	for _, m := range formalRows {
		st := "formal"
		if pendingRemove[m.Account] {
			st = "pendingRemove"
		}
		formal = append(formal, MemberItem{
			Account: m.Account, Name: m.Name, Role: m.Role,
			Hours: m.Hours, JoinDate: m.Join, Status: st,
		})
	}

	histRows, err := s.repo.ListHistory(ctx, id, 50)
	if err != nil {
		return nil, err
	}
	actors := make([]string, 0, len(histRows)+2)
	actors = append(actors, row.PO, firstAccount(row.Manager))
	for _, h := range histRows {
		actors = append(actors, h.Actor)
	}
	names, _ := s.repo.ResolveRealnames(ctx, actors)
	history := make([]HistoryItem, 0, len(histRows))
	for _, h := range histRows {
		var adjID int64
		if h.AdjustmentID != nil {
			adjID = *h.AdjustmentID
		}
		history = append(history, HistoryItem{
			EventType: h.EventType, Summary: h.Summary,
			Actor: h.Actor, ActorName: displayName(h.Actor, names),
			CreatedAt: formatTime(h.CreatedDate), AdjustmentID: adjID,
		})
	}

	coach := firstAccount(row.Manager)
	po := strings.TrimSpace(row.PO)
	parentOpts, _ := s.repo.ListParentTeamOptions(ctx, id)
	return &DetailResp{
		ID: row.ID, Name: row.Name, ParentID: row.Parent, ParentName: row.ParentName,
		CoachAccount: coach, CoachName: displayName(coach, names),
		POAccount: po, POName: displayName(po, names),
		Slogan: row.Slogan, Declaration: row.Declaration, Logo: row.Logo,
		Status: row.Status, StatusLabel: statusLabel(row.Status),
		CreatedDate: row.CreatedDate, FormalCount: len(formalRows),
		Formal: formal, PendingJoin: pendingJoin, Pending: pending,
		History: history, CanConfirm: canConfirm, CanEdit: canEdit,
		ParentOptions: parentOpts,
	}, nil
}

func buildPendingSummary(adj *Adjustment, items []AdjustmentItem) *PendingSummary {
	ps := &PendingSummary{
		AdjustmentID: adj.ID, AdjustNo: adj.AdjustNo,
		SubmittedBy: adj.SubmittedBy, SubmittedAt: formatTime(adj.CreatedDate),
		Status: adj.Status, Reason: adj.Reason,
	}
	for _, it := range items {
		switch it.ActionType {
		case ActionAdd:
			ps.AddCount++
		case ActionRemove:
			ps.RemoveCount++
		case ActionRoleChange:
			ps.ChangeCount++
		}
	}
	return ps
}

func nameList(items []AdjustmentItem, action string, names map[string]string) []string {
	out := []string{}
	for _, it := range items {
		if it.ActionType == action {
			out = append(out, displayName(it.Account, names))
		}
	}
	return out
}

func displayName(account string, names map[string]string) string {
	if n, ok := names[account]; ok && n != "" {
		return n
	}
	return account
}

// PendingAddGroupIDs 返回账号作为待确认新增成员所在的小组。
func (s *Service) PendingAddGroupIDs(ctx context.Context, account string) ([]uint, error) {
	return s.repo.ListPendingAddGroupIDsForAccount(ctx, account)
}

// TeamgroupsByIDs 批量查小组基本信息。
func (s *Service) TeamgroupsByIDs(ctx context.Context, ids []uint) ([]TeamgroupRow, error) {
	return s.repo.FindTeamgroupsByIDs(ctx, ids)
}

// MergedMembersForGroups 看板用：正式成员 + pending 新增，并标记 pending 移除。
func (s *Service) MergedMembersForGroups(ctx context.Context, teamgroupIDs []uint) (map[uint][]MemberItem, error) {
	out := map[uint][]MemberItem{}
	if len(teamgroupIDs) == 0 {
		return out, nil
	}
	formalByID, err := s.repo.ListMembersByGroupIDs(ctx, teamgroupIDs)
	if err != nil {
		return nil, err
	}
	pendingAdd, err := s.repo.ListPendingAddMembers(ctx, teamgroupIDs)
	if err != nil {
		return nil, err
	}
	pendingRemove, err := s.repo.ListPendingRemoveAccounts(ctx, teamgroupIDs)
	if err != nil {
		return nil, err
	}
	accs := []string{}
	for _, list := range pendingAdd {
		for _, it := range list {
			accs = append(accs, it.Account)
		}
	}
	names, _ := s.repo.ResolveRealnames(ctx, accs)

	for _, gid := range teamgroupIDs {
		list := make([]MemberItem, 0)
		seen := map[string]bool{}
		for _, m := range formalByID[gid] {
			st := "formal"
			if pendingRemove[gid] != nil && pendingRemove[gid][m.Account] {
				st = "pendingRemove"
			}
			list = append(list, MemberItem{
				Account: m.Account, Name: m.Name, Role: m.Role,
				Hours: m.Hours, JoinDate: m.Join, Status: st,
			})
			seen[m.Account] = true
		}
		for _, it := range pendingAdd[gid] {
			if seen[it.Account] {
				continue
			}
			list = append(list, MemberItem{
				Account: it.Account, Name: displayName(it.Account, names),
				Role: it.Role, Hours: it.AvailableHours,
				Status: "pendingAdd", Submitter: it.CreatedBy,
			})
		}
		out[gid] = list
	}
	return out, nil
}

// SearchCandidates 按账号/姓名搜索可加入成员。
func (s *Service) SearchCandidates(ctx context.Context, actor *model.User, req CandidateSearchReq) ([]CandidateItem, error) {
	req.Normalize()
	if _, err := requireActorAccount(actor); err != nil {
		return nil, err
	}
	return s.repo.SearchUsers(ctx, req.Q, 20)
}
