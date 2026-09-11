// =============================================================================
// 文件: internal/module/po/repo_weekly.go
// 模块: PO 工作台
// 类型: action
// 职责: 项目周报基础数据读：项目壳、最新周报、提交信号、部门树
// 依赖: gorm.io/gorm
// =============================================================================

package po

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"time"
)

type projectWeeklyProjectRow struct {
	ID         uint   `gorm:"column:id"`
	Code       string `gorm:"column:code"`
	Name       string `gorm:"column:name"`
	PM         string `gorm:"column:pm"`
	PMName     string `gorm:"column:pm_name"`
	PMDeptID   uint   `gorm:"column:pm_dept_id"`
	PMDeptName string `gorm:"column:pm_dept_name"`
	Begin      string `gorm:"column:begin"`
	Status     string `gorm:"column:status"`
	Model      string `gorm:"column:model"`
	Source     string `gorm:"column:source"`
	SortGroup  int    `gorm:"column:sort_group"`
}

type projectWeeklyTeamRow struct {
	ID       uint   `gorm:"column:id"`
	Name     string `gorm:"column:name"`
	ParentID uint   `gorm:"column:parent"`
	Grade    int    `gorm:"column:grade"`
	Path     string `gorm:"column:path"`
	Order    int    `gorm:"column:order"`
}

type projectWeeklyReportRow struct {
	ID                   uint   `gorm:"column:id"`
	Project              uint   `gorm:"column:project"`
	WeekStart            string `gorm:"column:weekStart"`
	Staff                int    `gorm:"column:staff"`
	Workload             string `gorm:"column:workload"`
	OverallSituation     int    `gorm:"column:overallSituation"`
	OverallSituationDesc string `gorm:"column:overallSituationDesc"`
}

type projectWeeklyCountRow struct {
	ProjectID uint `gorm:"column:project_id"`
	Cnt       int  `gorm:"column:cnt"`
}

const projectWeeklyProjectSelect = `
SELECT DISTINCT p.id AS id,
       COALESCE(p.code, '') AS code,
       COALESCE(p.name, '') AS name,
       COALESCE(p.PM, '') AS pm,
       COALESCE(u.realname, p.PM, '') AS pm_name,
       COALESCE(u.dept, 0) AS pm_dept_id,
       COALESCE(d.name, '') AS pm_dept_name,
       COALESCE(DATE_FORMAT(p.begin, '%Y-%m-%d'), '') AS begin,
       COALESCE(p.status, '') AS status,
       COALESCE(p.model, '') AS model
FROM zt_project AS p
LEFT JOIN zt_user AS u ON u.account = p.PM AND u.deleted = '0'
LEFT JOIN zt_dept AS d ON d.id = u.dept
`

func projectWeeklyTeamFilterSQL(teamIDs []uint, args *[]any) string {
	ids := uniquePositiveUints(teamIDs)
	if len(ids) == 0 {
		return ""
	}
	placeholders := make([]string, 0, len(ids))
	for _, id := range ids {
		placeholders = append(placeholders, "?")
		*args = append(*args, id)
	}
	return `
  AND EXISTS (
    SELECT 1 FROM zt_dept AS sel
    WHERE sel.id IN (` + strings.Join(placeholders, ",") + `)
      AND u.dept IS NOT NULL AND u.dept > 0
      AND (u.dept = sel.id OR (IFNULL(sel.path,'') <> '' AND IFNULL(d.path,'') LIKE CONCAT(sel.path, '%')))
  )`
}

const projectWeeklyProjectSelectWithSource = `
SELECT DISTINCT p.id AS id,
       COALESCE(p.code, '') AS code,
       COALESCE(p.name, '') AS name,
       COALESCE(p.PM, '') AS pm,
       COALESCE(u.realname, p.PM, '') AS pm_name,
       COALESCE(u.dept, 0) AS pm_dept_id,
       COALESCE(d.name, '') AS pm_dept_name,
       COALESCE(DATE_FORMAT(p.begin, '%Y-%m-%d'), '') AS begin,
       COALESCE(p.status, '') AS status,
       COALESCE(p.model, '') AS model,
       CASE
         WHEN (p.PM = me.account
               OR EXISTS (SELECT 1 FROM zt_team AS tm WHERE tm.root = p.id AND tm.type = 'project' AND tm.account = me.account)
               OR EXISTS (SELECT 1 FROM zt_project AS ep JOIN zt_team AS tm ON tm.root = ep.id AND tm.type = 'execution' AND tm.account = me.account WHERE ep.project = p.id AND ep.deleted = '0')
               OR p.openedBy = me.account)
              AND p.follow LIKE CONCAT('%,', me.id, ',%') THEN 'both'
         WHEN (p.PM = me.account
               OR EXISTS (SELECT 1 FROM zt_team AS tm WHERE tm.root = p.id AND tm.type = 'project' AND tm.account = me.account)
               OR EXISTS (SELECT 1 FROM zt_project AS ep JOIN zt_team AS tm ON tm.root = ep.id AND tm.type = 'execution' AND tm.account = me.account WHERE ep.project = p.id AND ep.deleted = '0')
               OR p.openedBy = me.account) THEN 'participated'
         ELSE 'watched'
       END AS source,
       CASE
         WHEN (p.PM = me.account
               OR EXISTS (SELECT 1 FROM zt_team AS tm WHERE tm.root = p.id AND tm.type = 'project' AND tm.account = me.account)
               OR EXISTS (SELECT 1 FROM zt_project AS ep JOIN zt_team AS tm ON tm.root = ep.id AND tm.type = 'execution' AND tm.account = me.account WHERE ep.project = p.id AND ep.deleted = '0')
               OR p.openedBy = me.account) THEN 0
         ELSE 1
       END AS sort_group
FROM zt_project AS p
INNER JOIN zt_user AS me ON me.account = ? AND me.deleted = '0'
LEFT JOIN zt_user AS u ON u.account = p.PM AND u.deleted = '0'
LEFT JOIN zt_dept AS d ON d.id = u.dept
`

// FindMineProjectWeeklyProjects 查询参与（PM或团队成员）或关注的项目。
// 参与定义：zt_project.PM = me.account OR zt_team(root=p.id AND type='project' AND account=me.account)
// 关注定义：p.follow LIKE '%,{me.id},%'
// 排序：参与优先（sort_group=0），仅关注在后（sort_group=1），同组内 p.id DESC
func (r *Repo) FindMineProjectWeeklyProjects(ctx context.Context, account string, scope string, teamIDs []uint, limit int) ([]projectWeeklyProjectRow, error) {
	if limit <= 0 {
		limit = 200
	}
	args := []any{account}
	teamSQL := projectWeeklyTeamFilterSQL(teamIDs, &args)
	args = append(args, limit)

	whereParticipated := `(
    p.PM = me.account
    OR EXISTS (SELECT 1 FROM zt_team AS tm WHERE tm.root = p.id AND tm.type = 'project' AND tm.account = me.account)
    OR EXISTS (SELECT 1 FROM zt_project AS ep JOIN zt_team AS tm ON tm.root = ep.id AND tm.type = 'execution' AND tm.account = me.account WHERE ep.project = p.id AND ep.deleted = '0')
    OR p.openedBy = me.account
  )`

	whereScope := `(` + whereParticipated + ` OR p.follow LIKE CONCAT('%,', me.id, ',%'))`
	if scope == "participated" {
		whereScope = whereParticipated
	} else if scope == "watched" {
		whereScope = `p.follow LIKE CONCAT('%,', me.id, ',%')`
	}

	var rows []projectWeeklyProjectRow
	err := r.db.WithContext(ctx).Raw(projectWeeklyProjectSelectWithSource+`
WHERE p.deleted = '0'
  AND p.type = 'project'
  AND `+whereScope+`
  AND EXISTS (SELECT 1 FROM zt_projectweekly AS pw WHERE pw.project = p.id)`+teamSQL+`
ORDER BY sort_group ASC, p.id DESC
LIMIT ?`, args...).Scan(&rows).Error
	return rows, err
}

// FindWatchedProjectWeeklyProjects 关注且存在 zt_projectweekly 的项目。
func (r *Repo) FindWatchedProjectWeeklyProjects(ctx context.Context, account string, teamIDs []uint, limit int) ([]projectWeeklyProjectRow, error) {
	return r.FindMineProjectWeeklyProjects(ctx, account, "watched", teamIDs, limit)
}

// FindAllProjectWeeklyProjects 全量存在 zt_projectweekly 的项目。
func (r *Repo) FindAllProjectWeeklyProjects(ctx context.Context, teamIDs []uint, limit int) ([]projectWeeklyProjectRow, error) {
	if limit <= 0 {
		limit = 200
	}
	args := []any{}
	teamSQL := projectWeeklyTeamFilterSQL(teamIDs, &args)
	args = append(args, limit)
	var rows []projectWeeklyProjectRow
	err := r.db.WithContext(ctx).Raw(projectWeeklyProjectSelect+`
WHERE p.deleted = '0'
  AND p.type = 'project'
  AND EXISTS (SELECT 1 FROM zt_projectweekly AS pw WHERE pw.project = p.id)`+teamSQL+`
ORDER BY p.id DESC
LIMIT ?`, args...).Scan(&rows).Error
	return rows, err
}

// FindLatestWeeklyReports 批量取各项目最新一期 zt_weeklyreport。
func (r *Repo) FindLatestWeeklyReports(ctx context.Context, projectIDs []uint) ([]projectWeeklyReportRow, error) {
	if len(projectIDs) == 0 {
		return nil, nil
	}
	var rows []projectWeeklyReportRow
	err := r.db.WithContext(ctx).Raw(`
SELECT wr.id, wr.project, DATE_FORMAT(wr.weekStart, '%Y-%m-%d') AS weekStart,
       wr.staff, wr.workload, wr.overallSituation, wr.overallSituationDesc
FROM zt_weeklyreport AS wr
INNER JOIN (
  SELECT project, MAX(weekStart) AS mx
  FROM zt_weeklyreport
  WHERE project IN ?
  GROUP BY project
) AS t ON t.project = wr.project AND t.mx = wr.weekStart
WHERE wr.project IN ?
ORDER BY wr.project, wr.id DESC`, projectIDs, projectIDs).Scan(&rows).Error
	return rows, err
}

// FindWeeklyReportsByWeekStart 取指定自然周各项目周报行。
func (r *Repo) FindWeeklyReportsByWeekStart(ctx context.Context, projectIDs []uint, weekStart string) ([]projectWeeklyReportRow, error) {
	if len(projectIDs) == 0 || weekStart == "" {
		return nil, nil
	}
	var rows []projectWeeklyReportRow
	err := r.db.WithContext(ctx).Raw(`
SELECT id, project, DATE_FORMAT(weekStart, '%Y-%m-%d') AS weekStart,
       staff, workload, overallSituation, overallSituationDesc
FROM zt_weeklyreport
WHERE project IN ? AND weekStart = ?`, projectIDs, weekStart).Scan(&rows).Error
	return rows, err
}

// FindWeeklyEditedReportIDs 返回存在 action=edited 的周报 ID。
func (r *Repo) FindWeeklyEditedReportIDs(ctx context.Context, reportIDs []uint) (map[uint]bool, error) {
	out := map[uint]bool{}
	if len(reportIDs) == 0 {
		return out, nil
	}
	type row struct {
		ObjectID uint `gorm:"column:objectID"`
	}
	var rows []row
	err := r.db.WithContext(ctx).Raw(`
SELECT DISTINCT objectID
FROM zt_action
WHERE objectType = 'weekly' AND action = 'edited' AND objectID IN ?`, reportIDs).Scan(&rows).Error
	if err != nil {
		return out, err
	}
	for _, row := range rows {
		out[row.ObjectID] = true
	}
	return out, nil
}

// FindWeeklyReportHistory 单项目历史周报（周期倒序）。
func (r *Repo) FindWeeklyReportHistory(ctx context.Context, projectID uint, limit int) ([]projectWeeklyReportRow, error) {
	if limit <= 0 {
		limit = 50
	}
	var rows []projectWeeklyReportRow
	err := r.db.WithContext(ctx).Raw(`
SELECT id, project, DATE_FORMAT(weekStart, '%Y-%m-%d') AS weekStart,
       staff, workload, overallSituation, overallSituationDesc
FROM zt_weeklyreport
WHERE project = ?
ORDER BY weekStart DESC, id DESC
LIMIT ?`, projectID, limit).Scan(&rows).Error
	return rows, err
}

// FindProjectWeeklyByID 单个参与或关注项目壳（校验 参与 或 follow）。
func (r *Repo) FindProjectWeeklyByID(ctx context.Context, account string, projectID uint) (*projectWeeklyProjectRow, error) {
	var row projectWeeklyProjectRow
	err := r.db.WithContext(ctx).Raw(projectWeeklyProjectSelectWithSource+`
WHERE p.deleted = '0'
  AND p.type = 'project'
  AND p.id = ?
  AND (
    p.PM = me.account
    OR EXISTS (SELECT 1 FROM zt_team AS tm WHERE tm.root = p.id AND tm.type = 'project' AND tm.account = me.account)
    OR EXISTS (SELECT 1 FROM zt_project AS ep JOIN zt_team AS tm ON tm.root = ep.id AND tm.type = 'execution' AND tm.account = me.account WHERE ep.project = p.id AND ep.deleted = '0')
    OR p.openedBy = me.account
    OR p.follow LIKE CONCAT('%,', me.id, ',%')
  )
  AND EXISTS (SELECT 1 FROM zt_projectweekly AS pw WHERE pw.project = p.id)
LIMIT 1`, account, projectID).Scan(&row).Error
	if err != nil {
		return nil, err
	}
	if row.ID == 0 {
		return nil, nil
	}
	return &row, nil
}

// FindProjectWeeklyByIDAny 单个有周报管理对象的项目（不校验关注）。
func (r *Repo) FindProjectWeeklyByIDAny(ctx context.Context, projectID uint) (*projectWeeklyProjectRow, error) {
	var row projectWeeklyProjectRow
	err := r.db.WithContext(ctx).Raw(projectWeeklyProjectSelect+`
WHERE p.deleted = '0'
  AND p.type = 'project'
  AND p.id = ?
  AND EXISTS (SELECT 1 FROM zt_projectweekly AS pw WHERE pw.project = p.id)
LIMIT 1`, projectID).Scan(&row).Error
	if err != nil {
		return nil, err
	}
	if row.ID == 0 {
		return nil, nil
	}
	return &row, nil
}

// FindProjectWeeklyTeamLeaves 有周报管理对象的项目 → PM 所在部门（叶子）去重。
func (r *Repo) FindProjectWeeklyTeamLeaves(ctx context.Context) ([]projectWeeklyTeamRow, error) {
	var rows []projectWeeklyTeamRow
	err := r.db.WithContext(ctx).Raw(`
SELECT DISTINCT d.id AS id, COALESCE(d.name, '') AS name,
       COALESCE(d.parent, 0) AS parent, COALESCE(d.grade, 0) AS grade,
       COALESCE(d.path, '') AS path, COALESCE(d.` + "`order`" + `, 0) AS ` + "`order`" + `
FROM zt_project AS p
INNER JOIN zt_user AS u ON u.account = p.PM AND u.deleted = '0' AND u.dept > 0
INNER JOIN zt_dept AS d ON d.id = u.dept
WHERE p.deleted = '0'
  AND p.type = 'project'
  AND EXISTS (SELECT 1 FROM zt_projectweekly AS pw WHERE pw.project = p.id)
ORDER BY d.path ASC, d.` + "`order`" + ` ASC, d.id ASC`).Scan(&rows).Error
	return rows, err
}

// FindDeptsByIDs 按 id 批量取部门（含 parent/grade/path，供建树）。
func (r *Repo) FindDeptsByIDs(ctx context.Context, ids []uint) ([]projectWeeklyTeamRow, error) {
	ids = uniquePositiveUints(ids)
	if len(ids) == 0 {
		return nil, nil
	}
	var rows []projectWeeklyTeamRow
	err := r.db.WithContext(ctx).Raw(`
SELECT id, COALESCE(name, '') AS name, COALESCE(parent, 0) AS parent,
       COALESCE(grade, 0) AS grade, COALESCE(path, '') AS path,
       COALESCE(`+"`order`"+`, 0) AS `+"`order`"+`
FROM zt_dept
WHERE id IN ?
ORDER BY path ASC, `+"`order`"+` ASC, id ASC`, ids).Scan(&rows).Error
	return rows, err
}

// FindUserDeptID 当前账号所属部门（zt_user.dept）。
func (r *Repo) FindUserDeptID(ctx context.Context, account string) (uint, error) {
	account = strings.TrimSpace(account)
	if account == "" {
		return 0, nil
	}
	var deptID uint
	err := r.db.WithContext(ctx).Raw(`
SELECT dept FROM zt_user WHERE account = ? AND deleted = '0' LIMIT 1`, account).Scan(&deptID).Error
	return deptID, err
}

func (r *Repo) scanProjectCounts(ctx context.Context, query string, args ...any) (map[uint]int, error) {
	out := map[uint]int{}
	var rows []projectWeeklyCountRow
	if err := r.db.WithContext(ctx).Raw(query, args...).Scan(&rows).Error; err != nil {
		return out, err
	}
	for _, row := range rows {
		out[row.ProjectID] = row.Cnt
	}
	return out, nil
}

func sumWorkloadHours(raw string) float64 {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "[]" || raw == "{}" {
		return 0
	}
	var asMap map[string]any
	if err := json.Unmarshal([]byte(raw), &asMap); err == nil {
		var total float64
		for _, v := range asMap {
			total += anyToFloat(v)
		}
		return total
	}
	return 0
}

func anyToFloat(v any) float64 {
	switch t := v.(type) {
	case float64:
		return t
	case string:
		f, _ := strconv.ParseFloat(strings.TrimSpace(t), 64)
		return f
	case json.Number:
		f, _ := t.Float64()
		return f
	default:
		return 0
	}
}

func parseDateYMD(s string) time.Time {
	s = strings.TrimSpace(s)
	if s == "" || s == "0000-00-00" {
		return time.Time{}
	}
	t, err := time.ParseInLocation("2006-01-02", truncYMD(s), time.Local)
	if err != nil {
		return time.Time{}
	}
	return t
}

func truncYMD(s string) string {
	if len(s) >= 10 {
		return s[:10]
	}
	return s
}
