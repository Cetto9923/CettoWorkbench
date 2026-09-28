package schedule

import (
	"context"
	"workbench/internal/model"
)

// ListHomeRelatedVersionWindows unions memberships and personal demand links.
// EXISTS avoids duplicate windows when several related demands share a window.
func (r *Repo) ListHomeRelatedVersionWindows(ctx context.Context, groups []uint, demands []int, limit int) ([]model.VersionWindow, error) {
	rows := []model.VersionWindow{}
	if len(groups) == 0 && len(demands) == 0 {
		return rows, nil
	}
	if limit < 1 || limit > 4 {
		limit = 4
	}
	q := r.db.WithContext(ctx).Model(&model.VersionWindow{}).Where("releaseDate >= CURDATE()")
	if len(groups) > 0 && len(demands) > 0 {
		q = q.Where("teamgroup IN ? OR EXISTS (SELECT 1 FROM zt_demandwindow dw WHERE dw.versionWindow = zt_versionwindow.id AND dw.deletedAt IS NULL AND dw.story = 0 AND dw.demand IN ?)", groups, demands)
	} else if len(groups) > 0 {
		q = q.Where("teamgroup IN ?", groups)
	} else {
		q = q.Where("EXISTS (SELECT 1 FROM zt_demandwindow dw WHERE dw.versionWindow = zt_versionwindow.id AND dw.deletedAt IS NULL AND dw.story = 0 AND dw.demand IN ?)", demands)
	}
	err := q.Order("releaseDate ASC, id ASC").Limit(limit).Find(&rows).Error
	return rows, err
}
