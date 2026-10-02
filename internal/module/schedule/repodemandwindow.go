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
SELECT window_id, item_kind, item_id FROM (
 SELECT dw.versionWindow AS window_id, 'demand' AS item_kind, CASE WHEN d.parent > 0 THEN d.parent ELSE d.id END AS item_id
 FROM zt_demandwindow dw JOIN zt_demand d ON d.id = dw.demand AND d.deleted = '0'
 WHERE dw.versionWindow IN ? AND dw.story = 0 AND dw.deletedAt IS NULL
 UNION
 SELECT dw.versionWindow, CASE WHEN s.fromDemand > 0 THEN 'demand' ELSE 'story' END,
 CASE WHEN s.fromDemand > 0 THEN CASE WHEN d.parent > 0 THEN d.parent ELSE d.id END
      ELSE CASE WHEN s.parent > 0 THEN s.parent ELSE s.id END END
 FROM zt_demandwindow dw JOIN zt_story s ON s.id = dw.story AND s.deleted = '0'
 LEFT JOIN zt_demand d ON d.id = s.fromDemand AND d.deleted = '0'
 WHERE dw.versionWindow IN ? AND dw.deletedAt IS NULL AND (s.fromDemand = 0 OR d.id IS NOT NULL)
 UNION
 SELECT vwp.versionWindow, CASE WHEN s.fromDemand > 0 THEN 'demand' ELSE 'story' END,
 CASE WHEN s.fromDemand > 0 THEN CASE WHEN d.parent > 0 THEN d.parent ELSE d.id END
      ELSE CASE WHEN s.parent > 0 THEN s.parent ELSE s.id END END
 FROM zt_versionwindowproduct vwp JOIN zt_planstory ps ON ps.plan = vwp.plan
 JOIN zt_story s ON s.id = ps.story AND s.deleted = '0'
 LEFT JOIN zt_demand d ON d.id = s.fromDemand AND d.deleted = '0'
 WHERE vwp.versionWindow IN ? AND vwp.deletedAt IS NULL AND (s.fromDemand = 0 OR d.id IS NOT NULL)
) AS window_items WHERE item_id > 0 ORDER BY item_kind, item_id`

// FindWindowWorkItems merges demand-level window links with the existing
// product-plan/story chain. Demand-pool stories collapse to their originating
// business demand; independent stories retain their own story identity. UNION
// deduplicates an item reached through both source chains.
func (r *Repo) FindWindowWorkItems(ctx context.Context, windowID uint64) ([]WindowWorkItem, error) {
	if windowID == 0 {
		return []WindowWorkItem{}, nil
	}
	items := []WindowWorkItem{}
	if err := r.db.WithContext(ctx).Raw(findWindowWorkItemsSQL, []uint64{windowID}, []uint64{windowID}, []uint64{windowID}).Scan(&items).Error; err != nil {
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

// FindDemandWindowBoundIDs 查询哪些业务需求（包含子需求及父需求）直接在 zt_demandwindow 关联了窗口。
func (r *Repo) FindDemandWindowBoundIDs(ctx context.Context, demandIDs []uint) (map[uint]bool, error) {
	if len(demandIDs) == 0 {
		return map[uint]bool{}, nil
	}
	const query = `
SELECT DISTINCT d.id
FROM zt_demand d
WHERE d.id IN ?
  AND EXISTS (
    SELECT 1 FROM zt_demandwindow dw
    WHERE dw.deletedAt IS NULL AND dw.story = 0 AND dw.versionWindow > 0
      AND (
        dw.demand = d.id
        OR dw.demand IN (SELECT c.id FROM zt_demand c WHERE c.parent = d.id AND c.deleted = '0')
        OR (d.parent > 0 AND dw.demand = d.parent)
      )
  )`
	var matchedIDs []uint
	if err := r.db.WithContext(ctx).Raw(query, demandIDs).Scan(&matchedIDs).Error; err != nil {
		return nil, err
	}
	out := make(map[uint]bool, len(matchedIDs))
	for _, id := range matchedIDs {
		out[id] = true
	}
	return out, nil
}
