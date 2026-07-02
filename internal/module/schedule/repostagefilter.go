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

const indepStoryNoWindowSQL = `
(
  (` + indepStoryHasChildrenSQL + `
    AND NOT EXISTS (
      SELECT 1 FROM zt_story ch
      INNER JOIN zt_planstory ps ON ps.story = ch.id
      INNER JOIN zt_versionwindowproduct vwp ON vwp.plan = ps.plan AND vwp.deletedAt IS NULL
      WHERE ch.parent = s.id AND ch.deleted = '0' AND ch.type = 'story'
    )
  )
  OR (
    NOT ` + indepStoryHasChildrenSQL + `
    AND NOT EXISTS (
      SELECT 1 FROM zt_planstory ps
      INNER JOIN zt_versionwindowproduct vwp ON vwp.plan = ps.plan AND vwp.deletedAt IS NULL
      WHERE ps.story = s.id
    )
  )
)`

const indepStoryHasWindowSQL = `
(
  (` + indepStoryHasChildrenSQL + `
    AND EXISTS (
      SELECT 1 FROM zt_story ch
      INNER JOIN zt_planstory ps ON ps.story = ch.id
      INNER JOIN zt_versionwindowproduct vwp ON vwp.plan = ps.plan AND vwp.deletedAt IS NULL
      WHERE ch.parent = s.id AND ch.deleted = '0' AND ch.type = 'story'
    )
  )
  OR (
    NOT ` + indepStoryHasChildrenSQL + `
    AND EXISTS (
      SELECT 1 FROM zt_planstory ps
      INNER JOIN zt_versionwindowproduct vwp ON vwp.plan = ps.plan AND vwp.deletedAt IS NULL
      WHERE ps.story = s.id
    )
  )
)`

const indepAggregateStoryMatch = `
(
  (ch.parent = s.id AND ch.deleted = '0' AND ch.type = 'story')
  OR (
    ch.id = s.id
    AND NOT ` + indepStoryHasChildrenSQL + `
  )
)`

const indepAggregateTaskSQL = `
EXISTS (
  SELECT 1 FROM zt_story ch
  WHERE ` + indepAggregateStoryMatch + `
    AND EXISTS (
      SELECT 1 FROM zt_task t
      WHERE t.story = ch.id
        AND t.deleted = '0'
        AND t.status != 'closed'
    )
)`

const indepAggregateUnassignedTaskSQL = `
EXISTS (
  SELECT 1 FROM zt_story ch
  WHERE ` + indepAggregateStoryMatch + `
    AND EXISTS (
      SELECT 1 FROM zt_task t
      WHERE t.story = ch.id
        AND t.deleted = '0'
        AND t.status != 'closed'
        AND (t.assignedTo IS NULL OR t.assignedTo = '')
    )
)`

const indepStoryStageNoTaskSQL = `
` + indepStoryHasWindowSQL + `
AND NOT ` + indepAggregateTaskSQL

const indepStoryStageTaskUnassignedSQL = `
` + indepStoryHasWindowSQL + `
AND ` + indepAggregateTaskSQL + `
AND ` + indepAggregateUnassignedTaskSQL

const indepStoryStageTaskAssignedSQL = `
` + indepStoryHasWindowSQL + `
AND ` + indepAggregateTaskSQL + `
AND NOT ` + indepAggregateUnassignedTaskSQL

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
			conditions = append(conditions, "("+indepStoryNoWindowSQL+")")
		case StageFilterNoTask:
			conditions = append(conditions, "("+indepStoryStageNoTaskSQL+")")
		case StageFilterTaskUnassigned:
			conditions = append(conditions, "("+indepStoryStageTaskUnassignedSQL+")")
		case StageFilterTaskAssigned:
			conditions = append(conditions, "("+indepStoryStageTaskAssignedSQL+")")
		}
	}
	if len(conditions) == 0 {
		return filterClause{}
	}
	return filterClause{sql: "AND (" + strings.Join(conditions, " OR ") + ")"}
}
