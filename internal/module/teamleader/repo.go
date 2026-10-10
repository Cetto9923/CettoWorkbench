// =============================================================================
// 文件: internal/module/teamleader/repo.go
// 模块: 团队长工作台
// 类型: repo
// 职责: 禅道 zt_teamgroup / zt_team / zt_user 的只读查询与参数化检索。
// 依赖: gorm.io/gorm
// =============================================================================

package teamleader

import (
	"context"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"
)

// Repo 团队长模块数据访问层。
type Repo struct {
	db *gorm.DB
}

// NewRepo 创建 Repo 实例。
func NewRepo(db *gorm.DB) *Repo {
	return &Repo{db: db}
}

// TeamgroupRaw 团队组原始数据行。
type TeamgroupRaw struct {
	ID      uint   `gorm:"column:id"`
	Type    string `gorm:"column:type"`
	Name    string `gorm:"column:name"`
	Manager string `gorm:"column:manager"`
	PO      string `gorm:"column:PO"`
	Parent  uint   `gorm:"column:parent"`
	Grade   int    `gorm:"column:grade"`
	Status  string `gorm:"column:status"`
}

// TeamMemberRaw 团队成员原始数据行。
type TeamMemberRaw struct {
	GroupID uint   `gorm:"column:group_id"`
	Account string `gorm:"column:account"`
	Role    string `gorm:"column:role"`
}

// FindParentTeams 查询所有有效的三级团队（type='parent'）。
func (r *Repo) FindParentTeams(ctx context.Context) ([]TeamgroupRaw, error) {
	var rows []TeamgroupRaw
	err := r.db.WithContext(ctx).
		Table("zt_teamgroup").
		Select("id, type, name, manager, PO, parent, grade, status").
		Where("type = ? AND deleted = ?", "parent", "0").
		Order("id ASC").
		Find(&rows).Error
	return rows, err
}

// FindParentTeamByID 查询指定 ID 的三级团队。
func (r *Repo) FindParentTeamByID(ctx context.Context, id uint) (*TeamgroupRaw, error) {
	var row TeamgroupRaw
	err := r.db.WithContext(ctx).
		Table("zt_teamgroup").
		Select("id, type, name, manager, PO, parent, grade, status").
		Where("id = ? AND type = ? AND deleted = ?", id, "parent", "0").
		Take(&row).Error
	if err != nil {
		return nil, err
	}
	return &row, nil
}

// FindSubGroupsByParentID 查询指定父团队下属的所有子小组。
func (r *Repo) FindSubGroupsByParentID(ctx context.Context, parentID uint) ([]TeamgroupRaw, error) {
	if parentID == 0 {
		return []TeamgroupRaw{}, nil
	}
	var rows []TeamgroupRaw
	err := r.db.WithContext(ctx).
		Table("zt_teamgroup").
		Select("id, type, name, manager, PO, parent, grade, status").
		Where("parent = ? AND type = ? AND deleted = ?", parentID, "child", "0").
		Order("id ASC").
		Find(&rows).Error
	return rows, err
}

// FindMembersByGroupIDs 批量参数化查询指定小组列表的所有成员。
// 当 groupIDs 为空时直接短路返回空切片，避免无效查询。
func (r *Repo) FindMembersByGroupIDs(ctx context.Context, groupIDs []uint) ([]TeamMemberRaw, error) {
	if len(groupIDs) == 0 {
		return []TeamMemberRaw{}, nil
	}
	var rows []TeamMemberRaw
	err := r.db.WithContext(ctx).
		Table("zt_team").
		Select("root AS group_id, account, COALESCE(role, '') AS role").
		Where("type = ? AND root IN ?", "teamgroup", groupIDs).
		Order("root ASC, account ASC").
		Find(&rows).Error
	return rows, err
}

// FindUserParentTeamIDsByAccount 查找用户关联的所有三级团队 ID 集合：
// 1. 直接在 parent 团队担任 manager 或 PO；
// 2. 在 child 小组担任 manager 或 PO，向上反推 parent；
// 3. 在 child 小组作为正式成员（zt_team），向上反推 parent。
func (r *Repo) FindUserParentTeamIDsByAccount(ctx context.Context, account string) ([]uint, error) {
	account = strings.TrimSpace(account)
	if account == "" {
		return []uint{}, nil
	}
	pattern := "(^|[[:space:],;])" + account + "([[:space:],;]|$)"
	type idRow struct {
		ID uint `gorm:"column:id"`
	}
	var rows []idRow

	// 联合查询三种归属路径，严格使用参数化占位符
	query := `
SELECT DISTINCT p.id
FROM zt_teamgroup p
WHERE p.type = 'parent' AND p.deleted = '0'
  AND (p.manager REGEXP ? OR p.PO = ?)
UNION
SELECT DISTINCT p.id
FROM zt_teamgroup c
JOIN zt_teamgroup p ON p.id = c.parent AND p.type = 'parent' AND p.deleted = '0'
WHERE c.type = 'child' AND c.deleted = '0'
  AND (c.manager REGEXP ? OR c.PO = ?)
UNION
SELECT DISTINCT p.id
FROM zt_team t
JOIN zt_teamgroup c ON c.id = t.root AND c.type = 'child' AND c.deleted = '0'
JOIN zt_teamgroup p ON p.id = c.parent AND p.type = 'parent' AND p.deleted = '0'
WHERE t.type = 'teamgroup' AND t.account = ?
ORDER BY id ASC`

	err := r.db.WithContext(ctx).Raw(query, pattern, account, pattern, account, account).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]uint, 0, len(rows))
	for _, it := range rows {
		out = append(out, it.ID)
	}
	return out, nil
}

// ResolveRealnames 批量解析工号列表对应的真实姓名。
func (r *Repo) ResolveRealnames(ctx context.Context, accounts []string) (map[string]string, error) {
	out := make(map[string]string)
	if len(accounts) == 0 {
		return out, nil
	}
	// 账号去重清洗
	uniq := make([]string, 0, len(accounts))
	seen := make(map[string]bool)
	for _, a := range accounts {
		a = strings.TrimSpace(a)
		if a != "" && !seen[a] {
			seen[a] = true
			uniq = append(uniq, a)
		}
	}
	if len(uniq) == 0 {
		return out, nil
	}
	type userRow struct {
		Account  string `gorm:"column:account"`
		Realname string `gorm:"column:realname"`
	}
	var rows []userRow
	err := r.db.WithContext(ctx).
		Table("zt_user").
		Select("account, realname").
		Where("account IN ? AND deleted = ?", uniq, "0").
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	for _, u := range rows {
		name := strings.TrimSpace(u.Realname)
		if name == "" {
			name = u.Account
		}
		out[u.Account] = name
	}
	return out, nil
}

// TaskDataRow 任务查询数据库原始行。
type TaskDataRow struct {
	ID              int64      `gorm:"column:id"`
	Name            string     `gorm:"column:name"`
	Type            string     `gorm:"column:type"`
	Status          string     `gorm:"column:status"`
	StoryID         int64      `gorm:"column:story"`
	AssignedTo      string     `gorm:"column:assignedTo"`
	FinishedBy      string     `gorm:"column:finishedBy"`
	Deadline        *time.Time `gorm:"column:deadline"`
	FinishedDate    *time.Time `gorm:"column:finishedDate"`
	StoryTitle      string     `gorm:"column:storyTitle"`
	DemandTeamGroup string     `gorm:"column:demandTeamGroup"`
}

// CountGroupConfirmedTasks 真实统计小组研发任务指标（区分状态、近30天完成及未完成逾期）。
func (r *Repo) CountGroupConfirmedTasks(ctx context.Context, groupID uint, account string, thirtyDaysAgo, today time.Time) (GroupTaskSummary, error) {
	var summary GroupTaskSummary
	if r == nil || r.db == nil || groupID == 0 {
		return summary, nil
	}
	gidStr := strconv.FormatUint(uint64(groupID), 10)
	account = strings.TrimSpace(account)

	type countResult struct {
		WaitCount    int `gorm:"column:wait_count"`
		DoingCount   int `gorm:"column:doing_count"`
		DoneCount    int `gorm:"column:done_count"`
		OverdueCount int `gorm:"column:overdue_count"`
	}
	var res countResult

	query := r.db.WithContext(ctx).
		Table("zt_demand AS d").
		Select(`
			COUNT(CASE WHEN t.status = 'wait' THEN 1 END) AS wait_count,
			COUNT(CASE WHEN t.status = 'doing' THEN 1 END) AS doing_count,
			COUNT(CASE WHEN t.status = 'done' AND t.finishedDate IS NOT NULL AND t.finishedDate != '0000-00-00 00:00:00' AND t.finishedDate >= ? THEN 1 END) AS done_count,
			COUNT(CASE WHEN t.status IN ('wait', 'doing') AND t.deadline IS NOT NULL AND t.deadline != '0000-00-00' AND t.deadline < ? THEN 1 END) AS overdue_count
		`, thirtyDaysAgo, today).
		Joins("INNER JOIN zt_story AS s ON s.fromDemand = d.id AND s.deleted = '0'").
		Joins("INNER JOIN zt_task AS t ON t.story = s.id AND t.deleted = '0'").
		Where("d.deleted = '0' AND d.teamGroup = ?", gidStr)

	if account != "" && account != "all" {
		query = query.Where("(t.assignedTo = ? OR (t.status = 'done' AND t.finishedBy = ?))", account, account)
	}

	if err := query.Scan(&res).Error; err != nil {
		return summary, err
	}

	summary.ConfirmedWait = res.WaitCount
	summary.ConfirmedDoing = res.DoingCount
	summary.ConfirmedDone = res.DoneCount
	summary.ConfirmedOverdue = res.OverdueCount
	summary.ConfirmedTotal = res.WaitCount + res.DoingCount + res.DoneCount
	return summary, nil
}

// FindGroupConfirmedTasksPaged 分页查询明确归属于指定小组的需求研发任务。
func (r *Repo) FindGroupConfirmedTasksPaged(ctx context.Context, groupID uint, account string, thirtyDaysAgo time.Time, limit, offset int) ([]TaskDataRow, error) {
	if r == nil || r.db == nil || groupID == 0 {
		return []TaskDataRow{}, nil
	}
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	gidStr := strconv.FormatUint(uint64(groupID), 10)
	account = strings.TrimSpace(account)

	var rows []TaskDataRow
	query := r.db.WithContext(ctx).
		Table("zt_demand AS d").
		Select(`t.id, t.name, t.type, t.status, t.story, t.assignedTo, t.finishedBy, t.deadline, t.finishedDate,
			IFNULL(s.title, '') AS storyTitle,
			d.teamGroup AS demandTeamGroup`).
		Joins("INNER JOIN zt_story AS s ON s.fromDemand = d.id AND s.deleted = '0'").
		Joins("INNER JOIN zt_task AS t ON t.story = s.id AND t.deleted = '0'").
		Where("d.deleted = '0' AND d.teamGroup = ?", gidStr).
		Where("(t.status IN ('wait', 'doing') OR (t.status = 'done' AND t.finishedDate IS NOT NULL AND t.finishedDate != '0000-00-00 00:00:00' AND t.finishedDate >= ?))", thirtyDaysAgo)

	if account != "" && account != "all" {
		query = query.Where("(t.assignedTo = ? OR (t.status = 'done' AND t.finishedBy = ?))", account, account)
	}

	err := query.Order("t.id DESC").Limit(limit).Offset(offset).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	return rows, nil
}

// CountPendingTasksByAccounts 统计待核查候选任务总数（严格校验当前查看者对所属项目的访问授权边界）。
func (r *Repo) CountPendingTasksByAccounts(ctx context.Context, accounts []string, viewerAccount string, isSuperAdmin bool) (int, error) {
	if r == nil || r.db == nil || len(accounts) == 0 {
		return 0, nil
	}
	var count int64
	query := r.db.WithContext(ctx).
		Table("zt_task AS t").
		Joins("INNER JOIN zt_project AS p ON p.id = t.project AND p.deleted = '0'").
		Joins("LEFT JOIN zt_story AS s ON s.id = t.story AND s.deleted = '0'").
		Joins("LEFT JOIN zt_demand AS d ON d.id = s.fromDemand AND d.deleted = '0'").
		Where("t.deleted = '0' AND t.status IN ('wait', 'doing')").
		Where("t.assignedTo IN ?", accounts).
		Where(`(
			t.story = 0
			OR s.id IS NULL
			OR d.id IS NULL
			OR d.teamGroup = ''
			OR d.teamGroup IS NULL
		)`)
	query = applyViewerProjectACL(query, viewerAccount, isSuperAdmin)
	err := query.Count(&count).Error
	if err != nil {
		return 0, err
	}
	return int(count), nil
}

// FindPendingTasksByAccountsPaged 分页查询小组成员名下待核查归属的候选任务（严格校验当前查看者对所属项目的访问授权边界）。
func (r *Repo) FindPendingTasksByAccountsPaged(ctx context.Context, accounts []string, viewerAccount string, isSuperAdmin bool, limit, offset int) ([]TaskDataRow, error) {
	if r == nil || r.db == nil || len(accounts) == 0 {
		return []TaskDataRow{}, nil
	}
	if limit <= 0 || limit > 200 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}
	var rows []TaskDataRow
	query := r.db.WithContext(ctx).
		Table("zt_task AS t").
		Select(`t.id, t.name, t.type, t.status, t.story, t.assignedTo, t.finishedBy, t.deadline, t.finishedDate,
			IFNULL(s.title, '') AS storyTitle,
			IFNULL(d.teamGroup, '') AS demandTeamGroup`).
		Joins("INNER JOIN zt_project AS p ON p.id = t.project AND p.deleted = '0'").
		Joins("LEFT JOIN zt_story AS s ON s.id = t.story AND s.deleted = '0'").
		Joins("LEFT JOIN zt_demand AS d ON d.id = s.fromDemand AND d.deleted = '0'").
		Where("t.deleted = '0' AND t.status IN ('wait', 'doing')").
		Where("t.assignedTo IN ?", accounts).
		Where(`(
			t.story = 0
			OR s.id IS NULL
			OR d.id IS NULL
			OR d.teamGroup = ''
			OR d.teamGroup IS NULL
		)`)
	query = applyViewerProjectACL(query, viewerAccount, isSuperAdmin)
	err := query.Order("t.id DESC").Limit(limit).Offset(offset).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	return rows, nil
}

// applyViewerProjectACL 校验当前查看者对任务关联项目的可见权限，严格以登录用户为主体。
func applyViewerProjectACL(query *gorm.DB, viewerAccount string, isSuperAdmin bool) *gorm.DB {
	if isSuperAdmin || query == nil {
		return query
	}
	acc := strings.TrimSpace(viewerAccount)
	if acc == "" {
		return query.Where("1 = 0")
	}
	pattern := "%," + acc + ",%"
	return query.Where(`(
		p.acl = 'open'
		OR p.PM = ?
		OR p.openedBy = ?
		OR CONCAT(',', p.whitelist, ',') LIKE ?
		OR EXISTS (SELECT 1 FROM zt_team AS tm WHERE tm.root = p.id AND tm.type = 'project' AND tm.account = ?)
		OR EXISTS (SELECT 1 FROM zt_project AS ep JOIN zt_team AS tm ON tm.root = ep.id AND tm.type = 'execution' AND tm.account = ? WHERE ep.project = p.id AND ep.deleted = '0')
		OR (p.acl = 'program' AND EXISTS (SELECT 1 FROM zt_project AS pr WHERE pr.id = p.parent AND pr.deleted = '0' AND (
			pr.acl = 'open' OR pr.PM = ? OR CONCAT(',', pr.whitelist, ',') LIKE ? OR EXISTS (SELECT 1 FROM zt_team AS ptm WHERE ptm.root = pr.id AND ptm.type = 'project' AND ptm.account = ?)
		)))
	)`, acc, acc, pattern, acc, acc, acc, pattern, acc)
}

// FindSubGroupNamesMap 批量查询小组ID对应的名称映射。
func (r *Repo) FindSubGroupNamesMap(ctx context.Context, groupIDs []uint) (map[uint]string, error) {
	out := make(map[uint]string)
	if len(groupIDs) == 0 {
		return out, nil
	}
	var rows []struct {
		ID   uint   `gorm:"column:id"`
		Name string `gorm:"column:name"`
	}
	err := r.db.WithContext(ctx).
		Table("zt_teamgroup").
		Select("id, name").
		Where("id IN ? AND deleted = '0'", groupIDs).
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	for _, it := range rows {
		out[it.ID] = it.Name
	}
	return out, nil
}

