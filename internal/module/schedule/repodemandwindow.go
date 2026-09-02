// =============================================================================
// 文件: internal/module/schedule/repodemandwindow.go
// 模块: 排期工作台
// 类型: action
// 职责: 业务需求级版本窗口只读查询。
// 依赖: internal/module/schedule/form.go
// =============================================================================

package schedule

import (
	"context"
	"strings"
)

type demandWindowRow struct {
	DemandID   uint   `gorm:"column:demand"`
	WindowID   uint   `gorm:"column:windowID"`
	WindowName string `gorm:"column:windowName"`
}

// FindDemandWindowMappings 查业务需求关联的业需级版本窗口（story=0，每 demand 取最新一条）。
func (r *Repo) FindDemandWindowMappings(ctx context.Context, demandIDs []uint) (map[uint]DemandWindowRef, error) {
	if len(demandIDs) == 0 {
		return map[uint]DemandWindowRef{}, nil
	}

	const query = `
SELECT dw.demand, dw.versionWindow AS windowID, vw.name AS windowName
FROM zt_demandwindow dw
INNER JOIN zt_versionwindow vw ON vw.id = dw.versionWindow AND vw.deletedAt IS NULL
WHERE dw.demand IN ?
  AND dw.story = 0
  AND dw.deletedAt IS NULL
  AND dw.versionWindow > 0
ORDER BY dw.demand ASC, dw.updatedDate DESC, dw.id DESC`

	var rows []demandWindowRow
	if err := r.db.WithContext(ctx).Raw(query, demandIDs).Scan(&rows).Error; err != nil {
		return nil, err
	}
	out := make(map[uint]DemandWindowRef, len(rows))
	for _, row := range rows {
		if _, exists := out[row.DemandID]; exists {
			continue
		}
		out[row.DemandID] = DemandWindowRef{
			DemandID:   row.DemandID,
			WindowID:   row.WindowID,
			WindowName: strings.TrimSpace(row.WindowName),
		}
	}
	return out, nil
}
