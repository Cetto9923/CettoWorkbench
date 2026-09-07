package po

import (
	"context"
	"strings"
	"time"

	"gorm.io/gorm"
)

// homeFocusQuery intersects the existing stage scopes with the homepage focus.
// Focus facts currently belong to demands; independent stories have no confirmed
// deadline/blocking/hang mapping in the homepage KPI contract.
func (r *Repo) homeFocusQuery(ctx context.Context, account string, req DemandsReq) *gorm.DB {
	parts := []string{}
	args := []interface{}{}
	today := time.Now().Format("2006-01-02")
	for index, stage := range valueStreamStages {
		if stage.status == "all" || (req.Status != "all" && stage.status != req.Status) {
			continue
		}
		q := r.roleDemandScope(ctx, account, mysqlStageFilters[stage.status])
		switch req.Focus {
		case "today":
			q = q.Where("deadline IS NOT NULL AND deadline != '0000-00-00' AND deadline <= ?", today)
		case "overdue":
			q = q.Where("deadline IS NOT NULL AND deadline != '0000-00-00' AND deadline < ?", today)
		case "blocked":
			q = q.Where("status = ?", "refuse")
		case "suspended":
			q = q.Where("hang = ?", "1")
		}
		var rows []struct{ ID int }
		stmt := q.Select("id, ? AS stage_index", index).Session(&gorm.Session{DryRun: true}).Find(&rows).Statement
		parts = append(parts, stmt.SQL.String())
		args = append(args, stmt.Vars...)
	}
	if len(parts) == 0 {
		// 无匹配阶段时返回空候选集，避免非法 SQL。
		return r.db.WithContext(ctx).Table("(SELECT NULL AS id, 0 AS stage_index WHERE 1 = 0) AS candidates").
			Select("id, stage_index")
	}
	return r.db.WithContext(ctx).Table("("+strings.Join(parts, " UNION ALL ")+") AS candidates", args...).
		Select("id, MIN(stage_index) AS stage_index").Group("id")
}

func (r *Repo) FindHomeFocus(ctx context.Context, account string, req DemandsReq) ([]itemRef, int, error) {
	if strings.TrimSpace(account) == "" {
		return nil, 0, nil
	}
	base := r.homeFocusQuery(ctx, account, req)
	var total int64
	if err := r.db.WithContext(ctx).Table("(?) AS focused", base).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []struct {
		ID         int
		StageIndex int
	}
	err := r.db.WithContext(ctx).Table("(?) AS focused", base).
		Select("id, stage_index").Order("stage_index ASC, id DESC").
		Offset((req.Page - 1) * req.PageSize).Limit(req.PageSize).Scan(&rows).Error
	if err != nil {
		return nil, 0, err
	}
	refs := make([]itemRef, 0, len(rows))
	for _, row := range rows {
		refs = append(refs, itemRef{kind: "demand", id: row.ID, stageStatus: valueStreamStages[row.StageIndex].status})
	}
	return refs, int(total), nil
}
