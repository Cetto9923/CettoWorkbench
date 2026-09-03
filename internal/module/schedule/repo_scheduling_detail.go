// =============================================================================
// 文件: internal/module/schedule/repo_scheduling_detail.go
// 模块: 排期工作台
// 类型: action
// 职责: 排期一体化弹窗业需详情只读查询(GetDemandSchedulingDetail)
//       及详情链路上的窗口关联查询(findDemandLevelWindow / findDemandWindowRef)。
// 依赖: (无项目内部包)
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
