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

// WindowWorkItem identifies one unique business demand or independent story in a version window.
type WindowWorkItem struct {
	Kind string `gorm:"column:item_kind"` // demand / story
	ID   uint   `gorm:"column:item_id"`
}

const findWindowWorkItemsSQL = `
SELECT item_kind, item_id
FROM (
  SELECT 'demand' AS item_kind, dw.demand AS item_id
  FROM zt_demandwindow dw
  WHERE dw.versionWindow = ? AND dw.story = 0 AND dw.deletedAt IS NULL AND dw.demand > 0

  UNION

  SELECT CASE
           WHEN s.sourceType = 'demandpool' AND s.fromDemand > 0 THEN 'demand'
           ELSE 'story'
         END AS item_kind,
         CASE
           WHEN s.sourceType = 'demandpool' AND s.fromDemand > 0 THEN s.fromDemand
           ELSE s.id
         END AS item_id
  FROM zt_versionwindowproduct vwp
  INNER JOIN zt_planstory ps ON ps.plan = vwp.plan
  INNER JOIN zt_story s ON s.id = ps.story AND s.deleted = '0'
  WHERE vwp.versionWindow = ? AND vwp.deletedAt IS NULL AND vwp.plan IS NOT NULL
) AS window_items
WHERE item_id > 0
ORDER BY item_kind ASC, item_id ASC`

// FindWindowWorkItems merges demand-level window links with the existing
// product-plan/story chain. Demand-pool stories collapse to their originating
// business demand; independent stories retain their own story identity. UNION
// deduplicates an item reached through both source chains.
func (r *Repo) FindWindowWorkItems(ctx context.Context, windowID uint64) ([]WindowWorkItem, error) {
	if windowID == 0 {
		return []WindowWorkItem{}, nil
	}
	items := []WindowWorkItem{}
	if err := r.db.WithContext(ctx).Raw(findWindowWorkItemsSQL, windowID, windowID).Scan(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
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
