// =============================================================================
// 文件: internal/module/kanban/repo_createissue.go
// 模块: 工作看板
// 类型: action
// 职责: 提问题弹窗项目/项目团队查询（对齐禅道 simpleCreate 参与项目口径）。
// 依赖: 无
// =============================================================================

package kanban

import (
	"context"
	"strconv"
	"strings"
)

type projectNameRow struct {
	ID     uint   `gorm:"column:id"`
	Name   string `gorm:"column:name"`
	Status string `gorm:"column:status"`
}

// ListParticipateTaskProjectIDs 任务参与项目（对齐禅道 getParticipateProjects）。
func (r *Repo) ListParticipateTaskProjectIDs(ctx context.Context, accounts []string) ([]uint, error) {
	if r == nil || r.db == nil || len(accounts) == 0 {
		return []uint{}, nil
	}

	seen := map[uint]struct{}{}
	out := make([]uint, 0)
	add := func(ids []uint) {
		for _, id := range ids {
			if id == 0 {
				continue
			}
			if _, ok := seen[id]; ok {
				continue
			}
			seen[id] = struct{}{}
			out = append(out, id)
		}
	}

	var openIDs []uint
	err := r.db.WithContext(ctx).
		Table("zt_task").
		Select("DISTINCT project").
		Where("status IN ? AND assignedTo IN ? AND IFNULL(mode, '') <> ? AND deleted = ?",
			[]string{"wait", "doing", "pause", "cancel"}, accounts, "multi", "0").
		Pluck("project", &openIDs).Error
	if err != nil {
		return nil, err
	}
	add(openIDs)

	var doneIDs []uint
	err = r.db.WithContext(ctx).
		Table("zt_task").
		Select("DISTINCT project").
		Where("status IN ? AND finishedBy IN ? AND deleted = ?",
			[]string{"done", "closed"}, accounts, "0").
		Pluck("project", &doneIDs).Error
	if err != nil {
		return nil, err
	}
	add(doneIDs)

	var multiIDs []uint
	err = r.db.WithContext(ctx).
		Table("zt_task AS t").
		Select("DISTINCT t.project").
		Joins("INNER JOIN zt_taskteam AS tt ON tt.task = t.id AND tt.account IN ?", accounts).
		Where("t.mode = ? AND t.status IN ? AND t.deleted = ?",
			"multi", []string{"wait", "doing", "pause", "cancel"}, "0").
		Pluck("project", &multiIDs).Error
	if err != nil {
		return nil, err
	}
	add(multiIDs)

	return out, nil
}

// ListIssueOwnerProjectIDs 提出人参与的问题所属项目（对齐 getParticipateIssuesProjectsByUsers）。
func (r *Repo) ListIssueOwnerProjectIDs(ctx context.Context, accounts []string) ([]uint, error) {
	if r == nil || r.db == nil || len(accounts) == 0 {
		return []uint{}, nil
	}

	var raw []string
	err := r.db.WithContext(ctx).
		Table("zt_issue").
		Select("DISTINCT project").
		Where("owner IN ? AND deleted = ?", accounts, "0").
		Pluck("project", &raw).Error
	if err != nil {
		return nil, err
	}

	out := make([]uint, 0, len(raw))
	seen := map[uint]struct{}{}
	for _, s := range raw {
		id, parseErr := strconv.ParseUint(strings.TrimSpace(s), 10, 64)
		if parseErr != nil || id == 0 {
			continue
		}
		uid := uint(id)
		if _, ok := seen[uid]; ok {
			continue
		}
		seen[uid] = struct{}{}
		out = append(out, uid)
	}
	return out, nil
}

// FindOpenProjectsByIDs 按 ID 取未关闭项目（deleted=0 且 status!=closed）
func (r *Repo) FindOpenProjectsByIDs(ctx context.Context, ids []uint) ([]projectNameRow, error) {
	if r == nil || r.db == nil || len(ids) == 0 {
		return []projectNameRow{}, nil
	}

	var rows []projectNameRow
	err := r.db.WithContext(ctx).
		Table("zt_project").
		Select("id, name, status").
		Where("id IN ? AND deleted = ? AND status <> ?", ids, "0", "closed").
		Order("id DESC").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	if rows == nil {
		return []projectNameRow{}, nil
	}
	return rows, nil
}

// ListProjectTeamAccounts 项目团队成员账号（zt_team type=project，未删除用户）。
func (r *Repo) ListProjectTeamAccounts(ctx context.Context, projectID uint) ([]string, error) {
	if r == nil || r.db == nil || projectID == 0 {
		return []string{}, nil
	}

	var accounts []string
	err := r.db.WithContext(ctx).
		Table("zt_team AS t1").
		Joins("INNER JOIN zt_user AS t2 ON t1.account = t2.account").
		Where("t1.root = ? AND t1.type = ? AND t2.deleted = ?", projectID, "project", "0").
		Order("t1.id ASC").
		Pluck("t1.account", &accounts).Error
	if err != nil {
		return nil, err
	}
	if accounts == nil {
		return []string{}, nil
	}
	out := make([]string, 0, len(accounts))
	seen := map[string]struct{}{}
	for _, a := range accounts {
		a = strings.TrimSpace(a)
		if a == "" {
			continue
		}
		if _, ok := seen[a]; ok {
			continue
		}
		seen[a] = struct{}{}
		out = append(out, a)
	}
	return out, nil
}
