package schedule

import "context"

// GetTasksByStories preserves the single-object predicates and ordering in one batch.
func (r *Repo) GetTasksByStories(ctx context.Context, ids []uint) (map[uint][]ZtTaskItem, error) {
	out := make(map[uint][]ZtTaskItem)
	ids = uniqueUints(ids)
	if len(ids) == 0 {
		return out, nil
	}
	const query = `
SELECT
  story,
  id,
  name,
  type,
  pri,
  assignedTo,
  estimate,
  consumed,
  ` + "`left`" + `,
  DATE_FORMAT(estStarted, '%Y-%m-%d') AS estStarted,
  DATE_FORMAT(deadline, '%Y-%m-%d') AS deadline,
  status,
  finishedBy,
  DATE_FORMAT(finishedDate, '%Y-%m-%d') AS finishedDate,
  project,
  execution
FROM zt_task
WHERE story IN ?
  AND deleted = '0'
  AND status != 'closed'
ORDER BY id ASC`
	var rows []struct {
		ZtTaskItem
		Story uint
	}
	if err := r.db.WithContext(ctx).Raw(query, ids).Scan(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		out[row.Story] = append(out[row.Story], row.ZtTaskItem)
	}
	return out, nil
}

// GetProjectsByProducts preserves the single-object predicates and ordering in one batch.
func (r *Repo) GetProjectsByProducts(ctx context.Context, ids []uint) (map[uint][]ZtProjectOption, error) {
	out := make(map[uint][]ZtProjectOption)
	ids = uniqueUints(ids)
	if len(ids) == 0 {
		return out, nil
	}
	const query = `
SELECT pp.product, p.id, p.name, p.status, p.model
FROM zt_projectproduct pp
JOIN zt_project p ON p.id = pp.project AND p.deleted = '0' AND p.type = 'project'
WHERE pp.product IN ?
  AND p.status IN ('doing', 'wait')
ORDER BY p.id DESC`
	var rows []struct {
		ZtProjectOption
		Product uint
	}
	if err := r.db.WithContext(ctx).Raw(query, ids).Scan(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		out[row.Product] = append(out[row.Product], row.ZtProjectOption)
	}
	return out, nil
}

// GetExecutionsByProjects preserves the single-object predicates and ordering in one batch.
func (r *Repo) GetExecutionsByProjects(ctx context.Context, ids []uint) (map[uint][]ZtExecutionOption, error) {
	out := make(map[uint][]ZtExecutionOption)
	ids = uniqueUints(ids)
	if len(ids) == 0 {
		return out, nil
	}
	const query = `
SELECT parent, id, name, type, status
FROM zt_project
WHERE parent IN ?
  AND type IN ('sprint', 'stage', 'kanban')
  AND deleted = '0'
  AND status IN ('doing', 'wait')
ORDER BY id DESC`
	var rows []struct {
		ZtExecutionOption
		Parent uint
	}
	if err := r.db.WithContext(ctx).Raw(query, ids).Scan(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		out[row.Parent] = append(out[row.Parent], row.ZtExecutionOption)
	}
	return out, nil
}
