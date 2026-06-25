// =============================================================================
// 文件: internal/module/schedule/repo_zentao_advanced_filter.go
// 模块: 排期工作台
// 类型: action
// 职责: 业需与独立研发需求列表高级筛选（小组/系统/阶段）SQL。
// 依赖: internal/module/schedule/form.go
// =============================================================================

package schedule

import "strings"

const bizDemandSubtreeStoryFrom = `
  (s.fromDemand = d.id OR s.fromDemand IN (
    SELECT c.id FROM zt_demand c WHERE c.parent = d.id AND c.deleted = '0'
  ))`

const bizDemandStageNoStorySQL = `
NOT EXISTS (
  SELECT 1 FROM zt_story s
  WHERE s.deleted = '0'
    AND s.sourceType = 'demandpool'
    AND s.type = 'story'
    AND ` + bizDemandSubtreeStoryFrom + `
)`

const bizDemandStageNoWindowSQL = `
EXISTS (
  SELECT 1 FROM zt_story s
  WHERE s.deleted = '0'
    AND s.sourceType = 'demandpool'
    AND s.type = 'story'
    AND ` + bizDemandSubtreeStoryFrom + `
)
AND NOT EXISTS (
  SELECT 1 FROM zt_story s
  INNER JOIN zt_planstory ps ON ps.story = s.id
  INNER JOIN zt_versionwindowproduct vwp ON vwp.plan = ps.plan AND vwp.deletedAt IS NULL
  WHERE s.deleted = '0'
    AND s.sourceType = 'demandpool'
    AND s.type = 'story'
    AND ` + bizDemandSubtreeStoryFrom + `
)`

const bizDemandHasWindowSQL = `
EXISTS (
  SELECT 1 FROM zt_story s
  INNER JOIN zt_planstory ps ON ps.story = s.id
  INNER JOIN zt_versionwindowproduct vwp ON vwp.plan = ps.plan AND vwp.deletedAt IS NULL
  WHERE s.deleted = '0'
    AND s.sourceType = 'demandpool'
    AND s.type = 'story'
    AND ` + bizDemandSubtreeStoryFrom + `
)`

const bizDemandMainSystemStoryFrom = `
  CAST(IFNULL(NULLIF(s.isMainSystemAssociation, ''), '0') AS SIGNED) = 1
  AND ` + bizDemandSubtreeStoryFrom

const bizDemandStageNoTaskSQL = `
` + bizDemandHasWindowSQL + `
AND NOT EXISTS (
  SELECT 1 FROM zt_story s
  INNER JOIN zt_task t ON t.story = s.id AND t.deleted = '0' AND t.status != 'closed'
  WHERE s.deleted = '0'
    AND s.sourceType = 'demandpool'
    AND s.type = 'story'
    AND ` + bizDemandMainSystemStoryFrom + `
)`

const bizDemandStageTaskAssignedSQL = `
` + bizDemandHasWindowSQL + `
AND EXISTS (
  SELECT 1 FROM zt_story s
  INNER JOIN zt_task t ON t.story = s.id AND t.deleted = '0' AND t.status != 'closed'
  WHERE s.deleted = '0'
    AND s.sourceType = 'demandpool'
    AND s.type = 'story'
    AND ` + bizDemandMainSystemStoryFrom + `
)
AND NOT EXISTS (
  SELECT 1 FROM zt_story s
  INNER JOIN zt_task t ON t.story = s.id AND t.deleted = '0' AND t.status != 'closed'
    AND (t.assignedTo IS NULL OR t.assignedTo = '')
  WHERE s.deleted = '0'
    AND s.sourceType = 'demandpool'
    AND s.type = 'story'
    AND ` + bizDemandMainSystemStoryFrom + `
)`

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

const indepStoryStageNoTaskSQL = `
` + indepStoryHasWindowSQL + `
AND NOT EXISTS (
  SELECT 1 FROM zt_story ch
  INNER JOIN zt_task t ON t.story = ch.id AND t.deleted = '0' AND t.status != 'closed'
  WHERE ` + indepAggregateStoryMatch + `
)`

const indepStoryStageTaskAssignedSQL = `
` + indepStoryHasWindowSQL + `
AND EXISTS (
  SELECT 1 FROM zt_story ch
  INNER JOIN zt_task t ON t.story = ch.id AND t.deleted = '0' AND t.status != 'closed'
  WHERE ` + indepAggregateStoryMatch + `
)
AND NOT EXISTS (
  SELECT 1 FROM zt_story ch
  INNER JOIN zt_task t ON t.story = ch.id AND t.deleted = '0' AND t.status != 'closed'
    AND (t.assignedTo IS NULL OR t.assignedTo = '')
  WHERE ` + indepAggregateStoryMatch + `
)`

const indepStoryGroupWindowSQL = `
(
  EXISTS (
    SELECT 1 FROM zt_planstory ps
    INNER JOIN zt_versionwindowproduct vwp ON vwp.plan = ps.plan AND vwp.deletedAt IS NULL
    INNER JOIN zt_versionwindow vw ON vw.id = vwp.versionWindow AND vw.deletedAt IS NULL
    WHERE ps.story = s.id AND vw.teamgroup IN ?
  )
  OR EXISTS (
    SELECT 1 FROM zt_story ch
    INNER JOIN zt_planstory ps ON ps.story = ch.id
    INNER JOIN zt_versionwindowproduct vwp ON vwp.plan = ps.plan AND vwp.deletedAt IS NULL
    INNER JOIN zt_versionwindow vw ON vw.id = vwp.versionWindow AND vw.deletedAt IS NULL
    WHERE ch.parent = s.id AND ch.deleted = '0' AND ch.type = 'story' AND vw.teamgroup IN ?
  )
)`

const bizDemandWindowIDsSQL = `
EXISTS (
  SELECT 1 FROM zt_story s
  INNER JOIN zt_planstory ps ON ps.story = s.id
  INNER JOIN zt_versionwindowproduct vwp ON vwp.plan = ps.plan AND vwp.deletedAt IS NULL
  INNER JOIN zt_versionwindow vw ON vw.id = vwp.versionWindow AND vw.deletedAt IS NULL
  WHERE s.deleted = '0'
    AND s.sourceType = 'demandpool'
    AND s.type = 'story'
    AND ` + bizDemandSubtreeStoryFrom + `
    AND vw.id IN ?
)`

const indepStoryWindowIDsSQL = `
(
  EXISTS (
    SELECT 1 FROM zt_planstory ps
    INNER JOIN zt_versionwindowproduct vwp ON vwp.plan = ps.plan AND vwp.deletedAt IS NULL
    INNER JOIN zt_versionwindow vw ON vw.id = vwp.versionWindow AND vw.deletedAt IS NULL
    WHERE ps.story = s.id AND vw.id IN ?
  )
  OR EXISTS (
    SELECT 1 FROM zt_story ch
    INNER JOIN zt_planstory ps ON ps.story = ch.id
    INNER JOIN zt_versionwindowproduct vwp ON vwp.plan = ps.plan AND vwp.deletedAt IS NULL
    INNER JOIN zt_versionwindow vw ON vw.id = vwp.versionWindow AND vw.deletedAt IS NULL
    WHERE ch.parent = s.id AND ch.deleted = '0' AND ch.type = 'story' AND vw.id IN ?
  )
)`

type advancedFilterParams struct {
	groupIDs   []uint
	productIDs []uint
	windowIDs  []uint
	stages     []string
}

func advancedFilterParamsFromBizReq(req ListBizDemandsReq) advancedFilterParams {
	return advancedFilterParams{
		groupIDs:   ParseCommaSeparatedUints(req.Groups),
		productIDs: ParseCommaSeparatedUints(req.Products),
		windowIDs:  ParseCommaSeparatedUints(req.Windows),
		stages:     ParseCommaSeparatedStages(req.Stages),
	}
}

func advancedFilterParamsFromIndepReq(req ListIndependentReq) advancedFilterParams {
	return advancedFilterParams{
		groupIDs:   ParseCommaSeparatedUints(req.Groups),
		productIDs: ParseCommaSeparatedUints(req.Products),
		windowIDs:  ParseCommaSeparatedUints(req.Windows),
		stages:     ParseCommaSeparatedStages(req.Stages),
	}
}

func buildBizDemandAdvancedClause(params advancedFilterParams) filterClause {
	var parts []string
	var args []interface{}

	if len(params.groupIDs) > 0 {
		parts = append(parts, "AND CAST(NULLIF(d.teamGroup, '') AS UNSIGNED) IN ?")
		args = append(args, params.groupIDs)
	}
	if len(params.productIDs) > 0 {
		parts = append(parts, "AND d.id IN (SELECT demand FROM zt_demandclarify WHERE product IN ?)")
		args = append(args, params.productIDs)
	}
	if len(params.windowIDs) > 0 {
		parts = append(parts, "AND "+bizDemandWindowIDsSQL)
		args = append(args, params.windowIDs)
	}
	if len(params.stages) > 0 {
		stageClause := buildBizDemandStageOrClause(params.stages)
		if stageClause.sql != "" {
			parts = append(parts, stageClause.sql)
			args = append(args, stageClause.args...)
		}
	}

	return filterClause{sql: strings.Join(parts, "\n"), args: args}
}

func buildIndepStoryAdvancedClause(params advancedFilterParams) filterClause {
	var parts []string
	var args []interface{}

	if len(params.groupIDs) > 0 {
		parts = append(parts, "AND "+indepStoryGroupWindowSQL)
		args = append(args, params.groupIDs, params.groupIDs)
	}
	if len(params.productIDs) > 0 {
		parts = append(parts, "AND s.product IN ?")
		args = append(args, params.productIDs)
	}
	if len(params.windowIDs) > 0 {
		parts = append(parts, "AND "+indepStoryWindowIDsSQL)
		args = append(args, params.windowIDs, params.windowIDs)
	}
	if len(params.stages) > 0 {
		stageClause := buildIndepStoryStageOrClause(params.stages)
		if stageClause.sql != "" {
			parts = append(parts, stageClause.sql)
		}
	}

	return filterClause{sql: strings.Join(parts, "\n"), args: args}
}

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
		case StageFilterTaskAssigned:
			conditions = append(conditions, "("+indepStoryStageTaskAssignedSQL+")")
		}
	}
	if len(conditions) == 0 {
		return filterClause{}
	}
	return filterClause{sql: "AND (" + strings.Join(conditions, " OR ") + ")"}
}

func mergeFilterClauses(base, extra filterClause) filterClause {
	return filterClause{
		sql:  base.sql + extra.sql,
		args: append(append([]interface{}{}, base.args...), extra.args...),
	}
}
