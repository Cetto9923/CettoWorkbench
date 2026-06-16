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

func (r *Repo) findDemandWindowRef(ctx context.Context, demandID uint) (uint, string, error) {
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
