package po

import "context"

type childExecutionCounts struct {
	DemandID     uint `gorm:"column:demand_id"`
	Stories      int
	Tasks        int
	BlockingBugs int `gorm:"column:blocking_bugs"`
}

// FindChildExecutionCounts aggregates each fact separately so task/bug joins
// cannot multiply counts. Query count is constant for any number of children.
func (r *DemandDetailRepo) FindChildExecutionCounts(ctx context.Context, ids []uint) (map[uint]childExecutionCounts, error) {
	out := make(map[uint]childExecutionCounts)
	if len(ids) == 0 {
		return out, nil
	}
	var rows []childExecutionCounts
	err := r.db.WithContext(ctx).Raw(`
SELECT s.fromDemand AS demand_id, COUNT(*) AS stories
FROM zt_story s WHERE s.fromDemand IN ? AND s.deleted = '0'
GROUP BY s.fromDemand`, ids).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		out[row.DemandID] = row
	}
	rows = nil
	err = r.db.WithContext(ctx).Raw(`
SELECT s.fromDemand AS demand_id, COUNT(*) AS tasks
FROM zt_story s JOIN zt_task t ON t.story = s.id AND t.deleted = '0'
WHERE s.fromDemand IN ? AND s.deleted = '0'
GROUP BY s.fromDemand`, ids).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		v := out[row.DemandID]
		v.Tasks = row.Tasks
		out[row.DemandID] = v
	}
	rows = nil
	err = r.db.WithContext(ctx).Raw(`
SELECT s.fromDemand AS demand_id, COUNT(*) AS blocking_bugs
FROM zt_story s JOIN zt_bug b ON b.story = s.id AND b.deleted = '0'
WHERE s.fromDemand IN ? AND s.deleted = '0' AND b.status = 'active' AND b.severity IN (1, 2)
GROUP BY s.fromDemand`, ids).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		v := out[row.DemandID]
		v.BlockingBugs = row.BlockingBugs
		out[row.DemandID] = v
	}
	return out, nil
}
