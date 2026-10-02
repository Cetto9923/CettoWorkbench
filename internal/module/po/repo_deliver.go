// =============================================================================
// 文件: internal/module/po/repo_deliver.go
// 模块: PO 工作台
// 类型: repository
// 职责: 发起交付表单读写、上线窗口准备与审计历史持久化（对齐禅道 demand::deliver）。
// =============================================================================

package po

import (
	"context"
	"fmt"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"workbench/internal/model"
)

// DeliverDetailRow 查询交付表单所需的详情与前置信息。
type DeliverDetailRow struct {
	ID               uint   `gorm:"column:id"`
	Name             string `gorm:"column:name"`
	Status           string `gorm:"column:status"`
	Deleted          string `gorm:"column:deleted"`
	AssignedTo       string `gorm:"column:assignedTo"`
	DistributedBy    string `gorm:"column:distributedBy"`
	CreatedBy        string `gorm:"column:createdBy"`
	SubmitedBy       string `gorm:"column:submitedBy"`
	SubmitBy         string `gorm:"column:submitBy"`
	QD               string `gorm:"column:QD"`
	RD               string `gorm:"column:RD"`
	BRA              string `gorm:"column:BRA"`
	Accepter         string `gorm:"column:accepter"`
	VeriFier         string `gorm:"column:veriFier"`
	DeliverDate      string `gorm:"column:deliverDate"`
	IsCarReview      string `gorm:"column:isCarReview"`
	IsGrayVerifyPlan string `gorm:"column:isGrayVerifyPlan"`
	VerifyDate       string `gorm:"column:verifyDate"`
	VerifyPlan       string `gorm:"column:verifyPlan"`
	VerifyFinish     string `gorm:"column:verifyFinish"`
	Product          string `gorm:"column:product"`
	MainSystem       string `gorm:"column:mainSystem"`
}

// FindDeliverDemand 查询待交付需求详情。
func (r *Repo) FindDeliverDemand(ctx context.Context, id uint) (*DeliverDetailRow, error) {
	db, err := r.homeActionWriter()
	if err != nil {
		return nil, err
	}
	var row DeliverDetailRow
	const query = `
SELECT
  id, name, status, deleted, assignedTo, distributedBy, createdBy,
  submitedBy, submitBy, QD, RD, BRA, accepter, veriFier,
  DATE_FORMAT(deliverDate, '%Y-%m-%d') AS deliverDate,
  isCarReview, isGrayVerifyPlan, verifyDate, verifyPlan,
  DATE_FORMAT(verifyFinish, '%Y-%m-%d') AS verifyFinish,
  product, mainSystem
FROM zt_demand
WHERE id = ? AND deleted = '0'
LIMIT 1`
	if err := db.WithContext(ctx).Raw(query, id).Scan(&row).Error; err != nil {
		return nil, err
	}
	if row.ID == 0 {
		return nil, errHomeActionNotFound
	}
	return &row, nil
}

// CheckDeliverBlockers 检查严重缺陷数及阻塞情况。
func (r *Repo) CheckDeliverBlockers(ctx context.Context, demandID uint) (severeBugs int, openBugs int, err error) {
	db, err := r.homeActionWriter()
	if err != nil {
		return 0, 0, err
	}
	// 查询关联研发需求下的未关闭缺陷
	const query = `
SELECT
  COUNT(CASE WHEN b.severity IN ('1', '2', '10') THEN 1 END) AS severe_count,
  COUNT(1) AS open_count
FROM zt_bug b
INNER JOIN zt_story s ON s.id = b.story AND s.deleted = '0'
WHERE s.fromDemand = ?
  AND b.deleted = '0'
  AND b.status IN ('active', 'resolved')`

	type bugCountRow struct {
		SevereCount int `gorm:"column:severe_count"`
		OpenCount   int `gorm:"column:open_count"`
	}
	var countRow bugCountRow
	if err := db.WithContext(ctx).Raw(query, demandID).Scan(&countRow).Error; err != nil {
		return 0, 0, err
	}
	return countRow.SevereCount, countRow.OpenCount, nil
}

// FindDemandLinkedWindow 查询需求已关联的上线窗口（zt_demandwindow story=0）。
func (r *Repo) FindDemandLinkedWindow(ctx context.Context, demandID uint) (windowID uint, windowName, releaseDate string, err error) {
	db, err := r.homeActionWriter()
	if err != nil {
		return 0, "", "", err
	}
	const query = `
SELECT dw.versionWindow AS window_id, vw.name AS window_name, DATE_FORMAT(vw.releaseDate, '%Y-%m-%d') AS release_date
FROM zt_demandwindow dw
INNER JOIN zt_versionwindow vw ON vw.id = dw.versionWindow AND vw.deletedAt IS NULL
WHERE dw.demand = ? AND dw.story = 0 AND dw.deletedAt IS NULL
ORDER BY dw.updatedDate DESC, dw.id DESC
LIMIT 1`

	type winRow struct {
		WindowID    uint   `gorm:"column:window_id"`
		WindowName  string `gorm:"column:window_name"`
		ReleaseDate string `gorm:"column:release_date"`
	}
	var res winRow
	if err := db.WithContext(ctx).Raw(query, demandID).Scan(&res).Error; err != nil {
		return 0, "", "", err
	}
	return res.WindowID, res.WindowName, res.ReleaseDate, nil
}

// ListUpcomingDeliverWindows 查询未过期的上线窗口列表。
func (r *Repo) ListUpcomingDeliverWindows(ctx context.Context) ([]DeliverWindowOption, error) {
	db, err := r.homeActionWriter()
	if err != nil {
		return nil, err
	}
	const query = `
SELECT id, name, DATE_FORMAT(releaseDate, '%Y-%m-%d') AS releaseDate
FROM zt_versionwindow
WHERE deletedAt IS NULL AND releaseDate >= CURDATE()
ORDER BY releaseDate ASC`

	var rows []DeliverWindowOption
	if err := db.WithContext(ctx).Raw(query).Scan(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

// ListInsideUsersForDeliver 查询可作为生产验证责任人的内部用户列表。
func (r *Repo) ListInsideUsersForDeliver(ctx context.Context) ([]ClarifyOption, error) {
	db, err := r.homeActionWriter()
	if err != nil {
		return nil, err
	}
	const query = `
SELECT account AS value, CONCAT(realname, ' (', account, ')') AS label, pinyin
FROM zt_user
WHERE deleted = '0' AND type = 'inside'
ORDER BY account ASC`

	var rows []ClarifyOption
	if err := db.WithContext(ctx).Raw(query).Scan(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

func (r *Repo) RequireDeliverWindow(ctx context.Context, windowID uint) error {
	db, err := r.homeActionWriter()
	if err != nil {
		return err
	}
	var window model.VersionWindow
	if err := db.WithContext(ctx).First(&window, windowID).Error; err != nil {
		return fmt.Errorf("上线窗口不存在：%w", err)
	}
	return nil
}

func (r *Repo) SaveDeliverWindow(ctx context.Context, id, windowID uint, account string) error {
	db, err := r.homeActionWriter()
	if err != nil {
		return err
	}
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var window model.VersionWindow
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&window, windowID).Error; err != nil {
			return err
		}
		if err := tx.Unscoped().Where("demand = ? AND story = 0", id).Delete(&model.DemandWindow{}).Error; err != nil {
			return err
		}
		return tx.Create(&model.DemandWindow{DemandID: id, WindowID: uint64(windowID), CreatedBy: account, UpdatedBy: account}).Error
	})
}
