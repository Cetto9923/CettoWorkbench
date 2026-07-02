// =============================================================================
// 文件: internal/module/schedule/repostagefilter.go
// 模块: 排期工作台
// 类型: action
// 职责: 业需与独立研发需求排期阶段筛选 SQL。
// 依赖: internal/module/schedule/form.go
// =============================================================================

package schedule

import "strings"

const bizDemandSubtreeStoryFrom = `
  (s.fromDemand = d.id OR s.fromDemand IN (
    SELECT c.id FROM zt_demand c WHERE c.parent = d.id AND c.deleted = '0'
  ))`

const bizDemandSubtreeHasDemandWindowSQL = `
EXISTS (
  SELECT 1 FROM zt_demandwindow dw
  WHERE dw.deletedAt IS NULL
    AND dw.story = 0
    AND dw.versionWindow > 0
    AND (
      dw.demand = d.id
      OR dw.demand IN (
        SELECT c.id FROM zt_demand c WHERE c.parent = d.id AND c.deleted = '0'
      )
    )
)`

const bizDemandSubtreeHasStorySQL = `
EXISTS (
  SELECT 1 FROM zt_story s
  WHERE s.deleted = '0'
    AND s.sourceType = 'demandpool'
    AND s.type = 'story'
    AND ` + bizDemandSubtreeStoryFrom + `
)`

const bizDemandMainSystemStoryFrom = `
  CAST(IFNULL(NULLIF(s.isMainSystemAssociation, ''), '0') AS SIGNED) = 1
  AND ` + bizDemandSubtreeStoryFrom

const bizDemandMainSystemStoryHasTaskSQL = `
EXISTS (
  SELECT 1 FROM zt_story s
  INNER JOIN zt_task t ON t.story = s.id
    AND t.deleted = '0'
    AND t.status != 'closed'
  WHERE s.deleted = '0'
    AND s.sourceType = 'demandpool'
    AND s.type = 'story'
    AND ` + bizDemandMainSystemStoryFrom + `
)`

const bizDemandMainSystemStoryUnassignedSQL = `
EXISTS (
  SELECT 1 FROM zt_story s
  INNER JOIN zt_task t ON t.story = s.id
    AND t.deleted = '0'
    AND t.status != 'closed'
    AND (t.assignedTo IS NULL OR t.assignedTo = '')
  WHERE s.deleted = '0'
    AND s.sourceType = 'demandpool'
    AND s.type = 'story'
    AND ` + bizDemandMainSystemStoryFrom + `
)`

const bizDemandStageNoWindowSQL = `
NOT ` + bizDemandSubtreeHasDemandWindowSQL

const bizDemandStageNoStorySQL = `
` + bizDemandSubtreeHasDemandWindowSQL + `
AND NOT ` + bizDemandSubtreeHasStorySQL

const bizDemandStageNoTaskSQL = `
` + bizDemandSubtreeHasDemandWindowSQL + `
AND ` + bizDemandSubtreeHasStorySQL + `
AND NOT ` + bizDemandMainSystemStoryHasTaskSQL

const bizDemandStageTaskUnassignedSQL = `
` + bizDemandSubtreeHasDemandWindowSQL + `
AND ` + bizDemandMainSystemStoryHasTaskSQL + `
AND ` + bizDemandMainSystemStoryUnassignedSQL

const bizDemandStageTaskAssignedSQL = `
` + bizDemandSubtreeHasDemandWindowSQL + `
AND ` + bizDemandMainSystemStoryHasTaskSQL + `
AND NOT ` + bizDemandMainSystemStoryUnassignedSQL

const indepStoryHasChildrenSQL = `
EXISTS (
  SELECT 1 FROM zt_story ch
  WHERE ch.parent = s.id AND ch.deleted = '0' AND ch.type = 'story'
)`

const indepAggregateStoryMatch = `
(
  (ch.parent = s.id AND ch.deleted = '0' AND ch.type = 'story')
  OR (
    ch.id = s.id
    AND NOT ` + indepStoryHasChildrenSQL + `
  )
)`

const indepStoryStageAggregateSQL = `
SELECT agg.top_id
FROM (
  SELECT
    x.top_id,
    SUM(CASE WHEN win.story IS NULL THEN 0 ELSE 1 END) AS window_count,
    SUM(COALESCE(task_stat.task_count, 0)) AS task_count,
    SUM(COALESCE(task_stat.unassigned_count, 0)) AS unassigned_task_count
  FROM (
    SELECT p.id AS top_id, ch.id AS story_id
    FROM zt_story ch
    INNER JOIN zt_story p ON p.id = ch.parent
      AND IFNULL(p.sourceType, '') != 'demandpool'
      AND p.parent = 0
      AND p.type = 'story'
      AND p.deleted = '0'
    WHERE ch.parent > 0
      AND ch.deleted = '0'
      AND ch.type = 'story'
    UNION ALL
    SELECT p.id AS top_id, p.id AS story_id
    FROM zt_story p
    LEFT JOIN (
      SELECT DISTINCT parent
      FROM zt_story
      WHERE parent > 0
        AND deleted = '0'
        AND type = 'story'
    ) child_parent ON child_parent.parent = p.id
    WHERE IFNULL(p.sourceType, '') != 'demandpool'
      AND p.parent = 0
      AND p.type = 'story'
      AND p.deleted = '0'
      AND child_parent.parent IS NULL
  ) x
  LEFT JOIN (
    SELECT DISTINCT ps.story
    FROM zt_planstory ps
    INNER JOIN zt_versionwindowproduct vwp
      ON vwp.plan = ps.plan AND vwp.deletedAt IS NULL
  ) win ON win.story = x.story_id
  LEFT JOIN (
    SELECT
      story,
      COUNT(*) AS task_count,
      SUM(CASE WHEN assignedTo IS NULL OR assignedTo = '' THEN 1 ELSE 0 END) AS unassigned_count
    FROM zt_task
    WHERE deleted = '0'
      AND status != 'closed'
    GROUP BY story
  ) task_stat ON task_stat.story = x.story_id
  GROUP BY x.top_id
) agg`

func buildBizDemandStageOrClause(stages []string) filterClause {
	conditions := make([]string, 0, len(stages))
	for _, stage := range stages {
		switch stage {
		case StageFilterNoStory:
			conditions = append(conditions, "("+bizDemandStageNoStorySQL+")")
		case StageFilterNoWindow:
			conditions = append(conditions, "("+bizDemandStageNoWindowSQL+")")
		case StageFilterNoTask:
			conditions = append(conditions, "("+bizDemandStageNoTaskSQL+")")
		case StageFilterTaskUnassigned:
			conditions = append(conditions, "("+bizDemandStageTaskUnassignedSQL+")")
		case StageFilterTaskAssigned:
			conditions = append(conditions, "("+bizDemandStageTaskAssignedSQL+")")
		}
	}
	if len(conditions) == 0 {
		return filterClause{}
	}
	return filterClause{sql: "AND (" + strings.Join(conditions, " OR ") + ")"}
}

func buildIndepStoryStageOrClause(stages []string) filterClause {
	conditions := make([]string, 0, len(stages))
	for _, stage := range stages {
		switch stage {
		case StageFilterNoWindow:
			conditions = append(conditions, "agg.window_count = 0")
		case StageFilterNoTask:
			conditions = append(conditions, "(agg.window_count > 0 AND agg.task_count = 0)")
		case StageFilterTaskUnassigned:
			conditions = append(conditions, "(agg.window_count > 0 AND agg.task_count > 0 AND agg.unassigned_task_count > 0)")
		case StageFilterTaskAssigned:
			conditions = append(conditions, "(agg.window_count > 0 AND agg.task_count > 0 AND agg.unassigned_task_count = 0)")
		}
	}
	if len(conditions) == 0 {
		return filterClause{}
	}
	return filterClause{sql: `AND s.id IN (
` + indepStoryStageAggregateSQL + `
  WHERE ` + strings.Join(conditions, " OR ") + `
)`}
}
