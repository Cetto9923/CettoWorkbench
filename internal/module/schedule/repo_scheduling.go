// =============================================================================
// 文件: internal/module/schedule/repo_scheduling.go
// 模块: 排期工作台
// 类型: action
// 职责: 排期一体化弹窗业需详情只读查询。
// 依赖: internal/module/schedule/form.go
// =============================================================================

package schedule

import (
	"context"
	"errors"
	"strings"
)

type demandSchedulingRow struct {
	ID               uint   `gorm:"column:id"`
	Name             string `gorm:"column:name"`
	Pri              string `gorm:"column:pri"`
	Status           string `gorm:"column:status"`
	Stage            string `gorm:"column:stage"`
	BRA              string `gorm:"column:BRA"`
	RD               string `gorm:"column:RD"`
	QD               string `gorm:"column:QD"`
	Accepter         string `gorm:"column:accepter"`
	MainSystem       string `gorm:"column:mainSystem"`
	SchedulePlanDate string `gorm:"column:schedulePlanDate"`
	DevelopFinish    string `gorm:"column:developFinish"`
	TestFinish       string `gorm:"column:testFinish"`
	AcceptancedDate  string `gorm:"column:acceptancedDate"`
}

// GetDemandSchedulingDetail 查询排期一体化弹窗所需的业需详情。
func (r *Repo) GetDemandSchedulingDetail(ctx context.Context, demandID uint) (*DemandSchedulingDetail, error) {
	if demandID == 0 {
		return nil, errors.New("业需 ID 无效")
	}

	const query = `
SELECT
  id,
  name,
  pri,
  status,
  stage,
  BRA,
  RD,
  QD,
  accepter,
  mainSystem,
  DATE_FORMAT(schedulePlanDate, '%Y-%m-%d') AS schedulePlanDate,
  DATE_FORMAT(developFinish, '%Y-%m-%d') AS developFinish,
  DATE_FORMAT(testFinish, '%Y-%m-%d') AS testFinish,
  DATE_FORMAT(acceptancedDate, '%Y-%m-%d %H:%i:%s') AS acceptancedDate
FROM zt_demand
WHERE id = ?
  AND deleted = '0'
LIMIT 1`

	var row demandSchedulingRow
	if err := r.db.WithContext(ctx).Raw(query, demandID).Scan(&row).Error; err != nil {
		return nil, err
	}
	if row.ID == 0 {
		return nil, errors.New("业需不存在")
	}

	mainSystemID := parseUintString(row.MainSystem)
	mainSystemName := ""
	if mainSystemID > 0 {
		productNames, err := r.FindProductsByIDs(ctx, []uint{mainSystemID})
		if err != nil {
			return nil, err
		}
		mainSystemName = productNames[mainSystemID]
	}

	windowID, windowName, err := r.findDemandWindowRef(ctx, demandID)
	if err != nil {
		return nil, err
	}

	accounts := collectNonEmptyAccounts(row.BRA, row.RD, row.QD, row.Accepter)
	realnameByAccount, err := r.FindUsersByAccounts(ctx, accounts)
	if err != nil {
		return nil, err
	}

	return &DemandSchedulingDetail{
		ID:               row.ID,
		Name:             strings.TrimSpace(row.Name),
		Pri:              parseDemandPri(row.Pri),
		BRA:              strings.TrimSpace(row.BRA),
		BRAName:          resolveRealname(row.BRA, realnameByAccount),
		RD:               strings.TrimSpace(row.RD),
		RDName:           resolveRealname(row.RD, realnameByAccount),
		QD:               strings.TrimSpace(row.QD),
		QDName:           resolveRealname(row.QD, realnameByAccount),
		Accepter:         strings.TrimSpace(row.Accepter),
		AccepterName:     resolveRealname(row.Accepter, realnameByAccount),
		MainSystemID:     mainSystemID,
		MainSystemName:   mainSystemName,
		SchedulePlanDate: formatZenTaoDate(row.SchedulePlanDate),
		DevelopFinish:    formatZenTaoDate(row.DevelopFinish),
		TestFinish:       formatZenTaoDate(row.TestFinish),
		AcceptancedDate:  formatZenTaoDate(row.AcceptancedDate),
		WindowID:         windowID,
		WindowName:       windowName,
	}, nil
}

// findDemandLevelWindow 读 zt_demandwindow 业需级（story=0）窗口关联，
// 优先于 plan 反推链路使用。查无行时返回 (0, "", nil) 不视为错误。
// 注意：Raw 不自动加软删除过滤，必须手加 dw.deletedAt IS NULL。
func (r *Repo) findDemandLevelWindow(ctx context.Context, demandID uint) (uint, string, error) {
	const query = `
SELECT dw.versionWindow AS windowID, vw.name AS windowName
FROM zt_demandwindow dw
INNER JOIN zt_versionwindow vw ON vw.id = dw.versionWindow AND vw.deletedAt IS NULL
WHERE dw.demand = ?
  AND dw.story = 0
  AND dw.deletedAt IS NULL
ORDER BY dw.updatedDate DESC, dw.id DESC
LIMIT 1`

	type demandLevelWindowRow struct {
		WindowID   uint   `gorm:"column:windowID"`
		WindowName string `gorm:"column:windowName"`
	}
	var row demandLevelWindowRow
	if err := r.db.WithContext(ctx).Raw(query, demandID).Scan(&row).Error; err != nil {
		return 0, "", err
	}
	if row.WindowID == 0 {
		return 0, "", nil
	}
	return row.WindowID, strings.TrimSpace(row.WindowName), nil
}

func (r *Repo) findDemandWindowRef(ctx context.Context, demandID uint) (uint, string, error) {
	// 优先读 zt_demandwindow 业需级（story=0）记录，避免无 story/plan 时回显为 0。
	if windowID, windowName, err := r.findDemandLevelWindow(ctx, demandID); err != nil {
		return 0, "", err
	} else if windowID > 0 {
		return windowID, windowName, nil
	}

	// Fallback：业需级无记录时走 story → plan 反推链路（保留原逻辑）。
	stories, err := r.FindStoriesByDemands(ctx, []uint{demandID})
	if err != nil {
		return 0, "", err
	}
	if len(stories) == 0 {
		return 0, "", nil
	}

	windowByStory, err := r.FindStoryWindowMappings(ctx, pluckStoryIDs(stories))
	if err != nil {
		return 0, "", err
	}

	for _, story := range stories {
		ref, ok := windowByStory[story.ID]
		if !ok || ref.WindowID == 0 {
			continue
		}
		return ref.WindowID, strings.TrimSpace(ref.WindowName), nil
	}
	return 0, "", nil
}

func collectNonEmptyAccounts(accounts ...string) []string {
	seen := make(map[string]struct{})
	out := make([]string, 0, len(accounts))
	for _, account := range accounts {
		account = strings.TrimSpace(account)
		if account == "" {
			continue
		}
		if _, exists := seen[account]; exists {
			continue
		}
		seen[account] = struct{}{}
		out = append(out, account)
	}
	return out
}

func formatZenTaoDate(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" || strings.HasPrefix(raw, "0000-00-00") {
		return ""
	}
	if len(raw) >= 10 {
		return raw[:10]
	}
	return raw
}

type schedulingWindowRow struct {
	ID          uint   `gorm:"column:id"`
	Name        string `gorm:"column:name"`
	ReleaseDate string `gorm:"column:releaseDate"`
}

type schedulingUserRow struct {
	Account  string `gorm:"column:account"`
	Realname string `gorm:"column:realname"`
}

// ListUpcomingSchedulingWindows 查询未过期的版本窗口列表。
func (r *Repo) ListUpcomingSchedulingWindows(ctx context.Context) ([]SchedulingWindowOption, error) {
	const query = `
SELECT
  id,
  name,
  DATE_FORMAT(releaseDate, '%Y-%m-%d') AS releaseDate
FROM zt_versionwindow
WHERE deletedAt IS NULL
  AND releaseDate >= CURDATE()
ORDER BY releaseDate ASC`

	var rows []schedulingWindowRow
	if err := r.db.WithContext(ctx).Raw(query).Scan(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]SchedulingWindowOption, 0, len(rows))
	for _, row := range rows {
		out = append(out, SchedulingWindowOption{
			ID:          row.ID,
			Name:        strings.TrimSpace(row.Name),
			ReleaseDate: formatZenTaoDate(row.ReleaseDate),
		})
	}
	return out, nil
}

// ListInsideUsersForScheduling 查询内部用户列表（负责人下拉）。
func (r *Repo) ListInsideUsersForScheduling(ctx context.Context) ([]SchedulingUserOption, error) {
	const query = `
SELECT account, realname
FROM zt_user
WHERE deleted = '0'
  AND type = 'inside'
ORDER BY account ASC`

	var rows []schedulingUserRow
	if err := r.db.WithContext(ctx).Raw(query).Scan(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]SchedulingUserOption, 0, len(rows))
	for _, row := range rows {
		account := strings.TrimSpace(row.Account)
		if account == "" {
			continue
		}
		realname := strings.TrimSpace(row.Realname)
		if realname == "" {
			realname = account
		}
		out = append(out, SchedulingUserOption{
			Account:  account,
			Realname: realname,
		})
	}
	return out, nil
}

// GetDemandInvolvedProducts 查询业需涉及的系统列表（来自 demandclarify）。
func (r *Repo) GetDemandInvolvedProducts(ctx context.Context, demandID uint) ([]ZtProductOption, error) {
	if demandID == 0 {
		return []ZtProductOption{}, nil
	}

	const query = `
SELECT DISTINCT p.id, p.name
FROM zt_demandclarify dc
JOIN zt_product p ON p.id = dc.product AND p.deleted = '0'
WHERE dc.demand = ?
ORDER BY p.id ASC`

	var rows []ZtProductOption
	if err := r.db.WithContext(ctx).Raw(query, demandID).Scan(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]ZtProductOption, 0, len(rows))
	for _, row := range rows {
		if row.ID == 0 {
			continue
		}
		out = append(out, ZtProductOption{
			ID:   row.ID,
			Name: strings.TrimSpace(row.Name),
		})
	}
	return out, nil
}

// GetDemandStories 按业需 ID 查询关联研发需求（排期弹窗用户故事条目）。
func (r *Repo) GetDemandStories(ctx context.Context, demandID uint) ([]ZtStory, error) {
	if demandID == 0 {
		return []ZtStory{}, nil
	}

	const query = `
SELECT
  id,
  title,
  product,
  estimate,
  assignedTo,
  CAST(IFNULL(NULLIF(isMainSystemAssociation, ''), '0') AS SIGNED) AS isMainSystemAssociation
FROM zt_story
WHERE fromDemand = ?
  AND deleted = '0'
ORDER BY isMainSystemAssociation DESC, id ASC`

	var rows []ZtStory
	if err := r.db.WithContext(ctx).Raw(query, demandID).Scan(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

// GetDemandUserStories 按业需 ID 查询用户故事条目（来自 zt_demanduserstory）。
func (r *Repo) GetDemandUserStories(ctx context.Context, demandID uint) ([]ZtDemandUserStory, error) {
	if demandID == 0 {
		return []ZtDemandUserStory{}, nil
	}

	const query = `
SELECT
  id,
  demand,
  role,
  gv,
  product,
  point,
  revpoint,
  source_type AS sourceType
FROM zt_demanduserstory
WHERE demand = ?
ORDER BY id ASC`

	var rows []ZtDemandUserStory
	if err := r.db.WithContext(ctx).Raw(query, demandID).Scan(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

// GetStoryTasks 查询某个研发需求下的未关闭任务列表。
func (r *Repo) GetStoryTasks(ctx context.Context, storyID uint) ([]ZtTaskItem, error) {
	if storyID == 0 {
		return []ZtTaskItem{}, nil
	}

	const query = `
SELECT
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
WHERE story = ?
  AND deleted = '0'
  AND status != 'closed'
ORDER BY id ASC`

	var rows []ZtTaskItem
	if err := r.db.WithContext(ctx).Raw(query, storyID).Scan(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

// GetProductProjects 查询产品关联的进行中和未开始项目。
func (r *Repo) GetProductProjects(ctx context.Context, productID uint) ([]ZtProjectOption, error) {
	if productID == 0 {
		return []ZtProjectOption{}, nil
	}

	const query = `
SELECT p.id, p.name, p.status, p.model
FROM zt_projectproduct pp
JOIN zt_project p ON p.id = pp.project AND p.deleted = '0' AND p.type = 'project'
WHERE pp.product = ?
  AND p.status IN ('doing', 'wait')
ORDER BY p.id DESC`

	var rows []ZtProjectOption
	if err := r.db.WithContext(ctx).Raw(query, productID).Scan(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

// GetProjectExecutions 查询项目下进行中和未开始的执行。
func (r *Repo) GetProjectExecutions(ctx context.Context, projectID uint) ([]ZtExecutionOption, error) {
	if projectID == 0 {
		return []ZtExecutionOption{}, nil
	}

	const query = `
SELECT id, name, type, status
FROM zt_project
WHERE parent = ?
  AND type IN ('sprint', 'stage', 'kanban')
  AND deleted = '0'
  AND status IN ('doing', 'wait')
ORDER BY id DESC`

	var rows []ZtExecutionOption
	if err := r.db.WithContext(ctx).Raw(query, projectID).Scan(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

// FindProjectsByIDs 批量查项目名称。
func (r *Repo) FindProjectsByIDs(ctx context.Context, ids []uint) (map[uint]string, error) {
	if len(ids) == 0 {
		return map[uint]string{}, nil
	}

	const query = `
SELECT id, name
FROM zt_project
WHERE id IN ?
  AND deleted = '0'`

	type projectNameRow struct {
		ID   uint   `gorm:"column:id"`
		Name string `gorm:"column:name"`
	}
	var rows []projectNameRow
	if err := r.db.WithContext(ctx).Raw(query, ids).Scan(&rows).Error; err != nil {
		return nil, err
	}
	out := make(map[uint]string, len(rows))
	for _, row := range rows {
		if row.ID == 0 {
			continue
		}
		out[row.ID] = strings.TrimSpace(row.Name)
	}
	return out, nil
}
