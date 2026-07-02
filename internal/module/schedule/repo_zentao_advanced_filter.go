// =============================================================================
// 文件: internal/module/schedule/repo_zentao_advanced_filter.go
// 模块: 排期工作台
// 类型: action
// 职责: 业需与独立研发需求列表高级筛选（小组/系统/阶段）SQL。
// 依赖: internal/module/schedule/form.go
// =============================================================================

package schedule

import "strings"

const bizDemandStoryKeywordTopDemandIDsSQL = `
SELECT DISTINCT CASE
  WHEN d0.parent IN (0, -1) THEN d0.id
  ELSE d0.parent
END
FROM zt_story s
INNER JOIN zt_demand d0 ON d0.id = s.fromDemand AND d0.deleted = '0'
WHERE s.deleted = '0'
  AND s.sourceType = 'demandpool'
  AND s.type = 'story'
  AND (
    CAST(s.id AS CHAR) = ?
    OR s.title LIKE ?
    OR s.assignedTo = ?
    OR s.assignedTo IN (SELECT u.account FROM zt_user u WHERE u.deleted = '0' AND u.realname LIKE ?)
    OR s.product IN (SELECT p.id FROM zt_product p WHERE p.deleted = '0' AND p.name LIKE ?)
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
(
  d.id IN (
    SELECT DISTINCT dw.demand
    FROM zt_demandwindow dw
    WHERE dw.deletedAt IS NULL
      AND dw.story = 0
      AND dw.versionWindow IN ?
  )
  OR d.id IN (
    SELECT DISTINCT c.parent
    FROM zt_demandwindow dw
    INNER JOIN zt_demand c ON c.id = dw.demand
    WHERE dw.deletedAt IS NULL
      AND dw.story = 0
      AND c.deleted = '0'
      AND c.parent > 0
      AND dw.versionWindow IN ?
  )
)`

const bizDemandWindowTypeSQL = `
(
  d.id IN (
    SELECT DISTINCT dw.demand
    FROM zt_demandwindow dw
    INNER JOIN zt_versionwindow vw ON vw.id = dw.versionWindow AND vw.deletedAt IS NULL
    WHERE dw.deletedAt IS NULL
      AND dw.story = 0
      AND vw.status = ?
  )
  OR d.id IN (
    SELECT DISTINCT c.parent
    FROM zt_demandwindow dw
    INNER JOIN zt_demand c ON c.id = dw.demand
    INNER JOIN zt_versionwindow vw ON vw.id = dw.versionWindow AND vw.deletedAt IS NULL
    WHERE dw.deletedAt IS NULL
      AND dw.story = 0
      AND c.deleted = '0'
      AND c.parent > 0
      AND vw.status = ?
  )
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

const indepStoryWindowTypeSQL = `
(
  EXISTS (
    SELECT 1 FROM zt_planstory ps
    INNER JOIN zt_versionwindowproduct vwp ON vwp.plan = ps.plan AND vwp.deletedAt IS NULL
    INNER JOIN zt_versionwindow vw ON vw.id = vwp.versionWindow AND vw.deletedAt IS NULL
    WHERE ps.story = s.id AND vw.status = ?
  )
  OR EXISTS (
    SELECT 1 FROM zt_story ch
    INNER JOIN zt_planstory ps ON ps.story = ch.id
    INNER JOIN zt_versionwindowproduct vwp ON vwp.plan = ps.plan AND vwp.deletedAt IS NULL
    INNER JOIN zt_versionwindow vw ON vw.id = vwp.versionWindow AND vw.deletedAt IS NULL
    WHERE ch.parent = s.id AND ch.deleted = '0' AND ch.type = 'story' AND vw.status = ?
  )
)`

type advancedFilterParams struct {
	groupIDs    []uint
	productIDs  []uint
	windowIDs   []uint
	stages      []string
	keyword     string
	pri         string
	windowType  string
	devOwner    string
	testOwner   string
	acceptOwner string
}

func advancedFilterParamsFromBizReq(req ListBizDemandsReq) advancedFilterParams {
	return advancedFilterParams{
		groupIDs:    ParseCommaSeparatedUints(req.Groups),
		productIDs:  ParseCommaSeparatedUints(req.Products),
		windowIDs:   ParseCommaSeparatedUints(req.Windows),
		stages:      ParseCommaSeparatedStages(req.Stages),
		keyword:     strings.TrimSpace(req.Keyword),
		pri:         NormalizePriorityFilter(req.Pri),
		windowType:  NormalizeWindowTypeFilter(req.WindowType),
		devOwner:    strings.TrimSpace(req.DevOwner),
		testOwner:   strings.TrimSpace(req.TestOwner),
		acceptOwner: strings.TrimSpace(req.AcceptOwner),
	}
}

func advancedFilterParamsFromIndepReq(req ListIndependentReq) advancedFilterParams {
	return advancedFilterParams{
		groupIDs:   ParseCommaSeparatedUints(req.Groups),
		productIDs: ParseCommaSeparatedUints(req.Products),
		windowIDs:  ParseCommaSeparatedUints(req.Windows),
		stages:     ParseCommaSeparatedStages(req.Stages),
		keyword:    strings.TrimSpace(req.Keyword),
		pri:        NormalizePriorityFilter(req.Pri),
		windowType: NormalizeWindowTypeFilter(req.WindowType),
		devOwner:   strings.TrimSpace(req.DevOwner),
		testOwner:  strings.TrimSpace(req.TestOwner),
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
		args = append(args, params.windowIDs, params.windowIDs)
	}
	if params.keyword != "" {
		exactID := extractScheduleSearchID(params.keyword)
		if exactID == "" {
			exactID = params.keyword
		}
		like := "%" + params.keyword + "%"
		parts = append(parts, `AND (
  CAST(d.id AS CHAR) = ?
  OR d.name LIKE ?
  OR d.BRA = ?
  OR d.RD = ?
  OR d.QD = ?
  OR d.accepter = ?
  OR d.BRA IN (SELECT u.account FROM zt_user u WHERE u.deleted = '0' AND u.realname LIKE ?)
  OR d.RD IN (SELECT u.account FROM zt_user u WHERE u.deleted = '0' AND u.realname LIKE ?)
  OR d.QD IN (SELECT u.account FROM zt_user u WHERE u.deleted = '0' AND u.realname LIKE ?)
  OR d.accepter IN (SELECT u.account FROM zt_user u WHERE u.deleted = '0' AND u.realname LIKE ?)
  OR CAST(NULLIF(d.mainSystem, '') AS UNSIGNED) IN (SELECT p.id FROM zt_product p WHERE p.deleted = '0' AND p.name LIKE ?)
  OR d.id IN (`+bizDemandStoryKeywordTopDemandIDsSQL+`)
)`)
		args = append(args,
			exactID, like, params.keyword, params.keyword, params.keyword, params.keyword, like, like, like, like, like,
			exactID, like, params.keyword, like, like,
		)
	}
	if params.pri != "" {
		parts = append(parts, "AND d.pri = ?")
		args = append(args, params.pri)
	}
	if params.windowType != "" {
		parts = append(parts, "AND "+bizDemandWindowTypeSQL)
		args = append(args, params.windowType, params.windowType)
	}
	if params.devOwner != "" {
		parts = append(parts, "AND d.RD = ?")
		args = append(args, params.devOwner)
	}
	if params.testOwner != "" {
		parts = append(parts, "AND d.QD = ?")
		args = append(args, params.testOwner)
	}
	if params.acceptOwner != "" {
		parts = append(parts, "AND d.accepter = ?")
		args = append(args, params.acceptOwner)
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
	if params.keyword != "" {
		exactID := extractScheduleSearchID(params.keyword)
		if exactID == "" {
			exactID = params.keyword
		}
		like := "%" + params.keyword + "%"
		parts = append(parts, `AND (
  CAST(s.id AS CHAR) = ?
  OR s.title LIKE ?
  OR s.assignedTo = ?
  OR s.assignedTo IN (SELECT u.account FROM zt_user u WHERE u.deleted = '0' AND u.realname LIKE ?)
  OR s.product IN (SELECT p.id FROM zt_product p WHERE p.deleted = '0' AND p.name LIKE ?)
)`)
		args = append(args, exactID, like, params.keyword, like, like)
	}
	if params.pri != "" {
		parts = append(parts, "AND s.pri = ?")
		args = append(args, params.pri)
	}
	if params.windowType != "" {
		parts = append(parts, "AND "+indepStoryWindowTypeSQL)
		args = append(args, params.windowType, params.windowType)
	}
	if params.devOwner != "" {
		parts = append(parts, "AND s.assignedTo = ?")
		args = append(args, params.devOwner)
	}
	if params.testOwner != "" {
		parts = append(parts, `AND EXISTS (
  SELECT 1 FROM zt_story ch
  INNER JOIN zt_task t ON t.story = ch.id AND t.deleted = '0' AND t.status != 'closed'
  WHERE (`+indepAggregateStoryMatch+`)
    AND t.type IN ('test', 'OLtest')
    AND t.assignedTo = ?
)`)
		args = append(args, params.testOwner)
	}
	if len(params.stages) > 0 {
		stageClause := buildIndepStoryStageOrClause(params.stages)
		if stageClause.sql != "" {
			parts = append(parts, stageClause.sql)
		}
	}

	return filterClause{sql: strings.Join(parts, "\n"), args: args}
}

func mergeFilterClauses(base, extra filterClause) filterClause {
	return filterClause{
		sql:  base.sql + extra.sql,
		args: append(append([]interface{}{}, base.args...), extra.args...),
	}
}
