// =============================================================================
// 文件: internal/module/po/repo_detail_flow.go
// 模块: PO 工作台
// 类型: repo
// 职责: 需求流转与审批数据读取（变更记录、挂起日志、评审记录、主管部门审批）。
// 依赖: gorm.io/gorm
// =============================================================================

package po

import (
	"context"
	"strings"
	"time"
)

// HasTable 探针：检查指定表在当前库是否存在。
func (r *DemandDetailRepo) HasTable(ctx context.Context, tableName string) (bool, error) {
	if r == nil || r.db == nil {
		return false, nil
	}
	var count int64
	err := r.db.WithContext(ctx).Raw(
		"SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = ?",
		tableName,
	).Scan(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *DemandDetailRepo) loadUserRealnames(ctx context.Context, accounts []string) map[string]string {
	res := make(map[string]string)
	if len(accounts) == 0 {
		return res
	}
	clean := make([]string, 0, len(accounts))
	for _, acc := range accounts {
		acc = strings.TrimSpace(acc)
		if acc != "" {
			clean = append(clean, acc)
		}
	}
	if len(clean) == 0 {
		return res
	}
	type uRow struct {
		Account  string `gorm:"column:account"`
		Realname string `gorm:"column:realname"`
	}
	var rows []uRow
	_ = r.db.WithContext(ctx).Table("zt_user").
		Select("account, realname").
		Where("deleted = '0' AND account IN ?", clean).
		Scan(&rows).Error
	for _, rw := range rows {
		if rw.Realname != "" {
			res[rw.Account] = rw.Realname
		}
	}
	return res
}

func resolveRealname(userMap map[string]string, account string) string {
	if name, ok := userMap[account]; ok && name != "" {
		return name
	}
	return account
}

type rawChange struct {
	ID                   uint       `gorm:"column:id"`
	Demand               uint       `gorm:"column:demand"`
	ChangeBy             string     `gorm:"column:changeBy"`
	ChangeDate           *time.Time `gorm:"column:changeDate"`
	ChangeType           string     `gorm:"column:changeType"`
	ReasonType           string     `gorm:"column:reasonType"`
	ChangeReason         string     `gorm:"column:changeReason"`
	Desc                 string     `gorm:"column:desc"`
	OldDesc              string     `gorm:"column:oldDesc"`
	ChangeMainDept       string     `gorm:"column:changeMainDept"`
	Result               string     `gorm:"column:result"`
	CreatedDate          *time.Time `gorm:"column:createdDate"`
	EstimateLaunchChange *time.Time `gorm:"column:estimateLaunchChange"`
	DevelopFinishChange  *time.Time `gorm:"column:developFinishChange"`
	TestFinishChange     *time.Time `gorm:"column:testFinishChange"`
	VerifyFinishChange   *time.Time `gorm:"column:verifyFinishChange"`
}

// FindDemandChanges 查询需求变更记录列表。
func (r *DemandDetailRepo) FindDemandChanges(ctx context.Context, demandID uint) ([]DemandChangeItem, error) {
	exists, err := r.HasTable(ctx, "zt_demandchange")
	if err != nil || !exists {
		return nil, err
	}

	var raws []rawChange
	err = r.db.WithContext(ctx).Raw(`
SELECT id, demand, changeBy, changeDate,
       COALESCE(changeType, '') AS changeType,
       COALESCE(reasonType, '') AS reasonType,
       COALESCE(changeReason, '') AS changeReason,
       COALESCE(`+"`desc`"+`, '') AS `+"`desc`"+`,
       COALESCE(oldDesc, '') AS oldDesc,
       COALESCE(changeMainDept, '') AS changeMainDept,
       COALESCE(result, '') AS result,
       createdDate, estimateLaunchChange, developFinishChange,
       testFinishChange, verifyFinishChange
FROM zt_demandchange
WHERE demand = ?
ORDER BY id DESC`, demandID).Scan(&raws).Error
	if err != nil {
		return nil, err
	}
	if len(raws) == 0 {
		return nil, nil
	}

	accounts := make([]string, 0, len(raws))
	changeIDs := make([]uint, 0, len(raws))
	for _, rw := range raws {
		accounts = append(accounts, rw.ChangeBy)
		changeIDs = append(changeIDs, rw.ID)
	}
	userMap := r.loadUserRealnames(ctx, accounts)
	revMap := r.loadChangeReviewers(ctx, changeIDs)

	items := make([]DemandChangeItem, 0, len(raws))
	for _, row := range raws {
		item := mapChangeRowToItem(row)
		item.ChangeByName = resolveRealname(userMap, row.ChangeBy)
		if acc, ok := revMap[row.ID]; ok {
			item.ChangeReviewer = strings.Join(acc.Accounts, ",")
			item.ChangeReviewerName = strings.Join(acc.Names, "、")
		}
		items = append(items, item)
	}
	return items, nil
}

type changeReviewerAccumulator struct {
	Accounts []string
	Names    []string
}

func (r *DemandDetailRepo) loadChangeReviewers(ctx context.Context, changeIDs []uint) map[uint]*changeReviewerAccumulator {
	res := make(map[uint]*changeReviewerAccumulator)
	if len(changeIDs) == 0 {
		return res
	}
	exists, err := r.HasTable(ctx, "zt_demandchangereview")
	if err != nil || !exists {
		return res
	}

	type rawRev struct {
		ChangeID uint   `gorm:"column:changeID"`
		Reviewer string `gorm:"column:reviewer"`
	}
	var rows []rawRev
	_ = r.db.WithContext(ctx).Raw(`
SELECT changeID, reviewer
FROM zt_demandchangereview
WHERE changeID IN (?) AND deleted = '0'
ORDER BY id ASC`, changeIDs).Scan(&rows).Error

	accounts := make([]string, 0, len(rows))
	for _, row := range rows {
		accounts = append(accounts, row.Reviewer)
	}
	userMap := r.loadUserRealnames(ctx, accounts)

	for _, row := range rows {
		acc, ok := res[row.ChangeID]
		if !ok {
			acc = &changeReviewerAccumulator{}
			res[row.ChangeID] = acc
		}
		if row.Reviewer != "" {
			acc.Accounts = append(acc.Accounts, row.Reviewer)
			acc.Names = append(acc.Names, resolveRealname(userMap, row.Reviewer))
		}
	}
	return res
}

func mapChangeRowToItem(r rawChange) DemandChangeItem {
	return DemandChangeItem{
		ID:                   r.ID,
		DemandID:             r.Demand,
		ChangeBy:             r.ChangeBy,
		ChangeDate:           formatTime(r.ChangeDate, "2006-01-02 15:04"),
		ChangeType:           r.ChangeType,
		ChangeTypeLabel:      mapChangeTypeLabel(r.ChangeType),
		ReasonType:           r.ReasonType,
		ChangeReason:         r.ChangeReason,
		Desc:                 r.Desc,
		OldDesc:              r.OldDesc,
		ChangeMainDept:       r.ChangeMainDept,
		Result:               r.Result,
		ResultLabel:          mapReviewResultLabel(r.Result),
		CreatedDate:          formatTime(r.CreatedDate, "2006-01-02 15:04"),
		EstimateLaunchChange: formatTime(r.EstimateLaunchChange, "2006-01-02"),
		DevelopFinishChange:  formatTime(r.DevelopFinishChange, "2006-01-02"),
		TestFinishChange:     formatTime(r.TestFinishChange, "2006-01-02"),
		VerifyFinishChange:   formatTime(r.VerifyFinishChange, "2006-01-02"),
	}
}

// FindDemandHangLogs 查询需求挂起日志列表。
func (r *DemandDetailRepo) FindDemandHangLogs(ctx context.Context, demandID uint) ([]DemandHangLogItem, error) {
	exists, err := r.HasTable(ctx, "zt_hanglog")
	if err != nil || !exists {
		return nil, err
	}

	type rawHangLog struct {
		ID         uint       `gorm:"column:id"`
		Account    string     `gorm:"column:account"`
		Action     string     `gorm:"column:action"`
		HangUpType string     `gorm:"column:hangUpType"`
		Date       *time.Time `gorm:"column:date"`
		Extra      string     `gorm:"column:extra"`
		Comment    string     `gorm:"column:comment"`
	}

	var raws []rawHangLog
	err = r.db.WithContext(ctx).Raw(`
SELECT id, account, action,
       COALESCE(hangUpType, '') AS hangUpType, date,
       COALESCE(extra, '') AS extra,
       COALESCE(comment, '') AS comment
FROM zt_hanglog
WHERE objectType = 'demand' AND objectID = ?
ORDER BY id DESC`, demandID).Scan(&raws).Error
	if err != nil {
		return nil, err
	}
	if len(raws) == 0 {
		return nil, nil
	}

	accounts := make([]string, 0, len(raws))
	for _, rw := range raws {
		accounts = append(accounts, rw.Account)
	}
	userMap := r.loadUserRealnames(ctx, accounts)

	items := make([]DemandHangLogItem, 0, len(raws))
	for _, row := range raws {
		actLabel := "挂起"
		if row.Action == "recovery" {
			actLabel = "恢复"
		}
		items = append(items, DemandHangLogItem{
			ID:              row.ID,
			Account:         row.Account,
			AccountName:     resolveRealname(userMap, row.Account),
			Action:          row.Action,
			ActionLabel:     actLabel,
			HangUpType:      row.HangUpType,
			HangUpTypeLabel: mapHangUpTypeLabel(row.HangUpType),
			Date:            formatTime(row.Date, "2006-01-02 15:04"),
			Extra:           row.Extra,
			Comment:         row.Comment,
		})
	}
	return items, nil
}

// FindDemandReviewRecords 查询需求评审记录。
func (r *DemandDetailRepo) FindDemandReviewRecords(ctx context.Context, demandID uint) ([]DemandReviewRecordItem, error) {
	exists, err := r.HasTable(ctx, "zt_demandreviewrecord")
	if err != nil || !exists {
		return nil, err
	}

	type rawRecord struct {
		ID           uint       `gorm:"column:id"`
		ReviewType   string     `gorm:"column:reviewType"`
		ReviewDate   *time.Time `gorm:"column:reviewDate"`
		ReviewResult string     `gorm:"column:reviewResult"`
		ReviewStatus string     `gorm:"column:reviewStatus"`
		CreatedBy    string     `gorm:"column:createdBy"`
		CreatedDate  *time.Time `gorm:"column:createdDate"`
	}

	var raws []rawRecord
	err = r.db.WithContext(ctx).Raw(`
SELECT id, COALESCE(reviewType, '') AS reviewType, reviewDate,
       COALESCE(reviewResult, '') AS reviewResult,
       COALESCE(reviewStatus, '') AS reviewStatus,
       COALESCE(createdBy, '') AS createdBy,
       createdDate
FROM zt_demandreviewrecord
WHERE objectID = ? AND deleted = '0'
ORDER BY createdDate DESC, id DESC`, demandID).Scan(&raws).Error
	if err != nil {
		return nil, err
	}
	if len(raws) == 0 {
		return nil, nil
	}

	accounts := make([]string, 0, len(raws))
	for _, rw := range raws {
		accounts = append(accounts, rw.CreatedBy)
	}
	userMap := r.loadUserRealnames(ctx, accounts)

	items := make([]DemandReviewRecordItem, 0, len(raws))
	for _, row := range raws {
		items = append(items, DemandReviewRecordItem{
			ID:                row.ID,
			ReviewType:        row.ReviewType,
			ReviewTypeLabel:   mapReviewTypeLabel(row.ReviewType),
			ReviewDate:        formatTime(row.ReviewDate, "2006-01-02"),
			ReviewResult:      row.ReviewResult,
			ReviewStatus:      row.ReviewStatus,
			ReviewStatusLabel: mapReviewStatusLabel(row.ReviewStatus),
			CreatedBy:         row.CreatedBy,
			CreatedByName:     resolveRealname(userMap, row.CreatedBy),
			CreatedDate:       formatTime(row.CreatedDate, "2006-01-02"),
		})
	}
	return items, nil
}

type rawReview struct {
	ID           uint       `gorm:"column:id"`
	Result       string     `gorm:"column:result"`
	Reviewer     string     `gorm:"column:reviewer"`
	Comment      string     `gorm:"column:comment"`
	SubmitedBy   string     `gorm:"column:submitedBy"`
	SubmitedDate *time.Time `gorm:"column:submitedDate"`
	ReviewDate   *time.Time `gorm:"column:reviewDate"`
	ResultStatus string     `gorm:"column:resultStatus"`
}

// FindDemandManagerReviews 查询主管部门审批记录（含子表产品与部门）。
func (r *DemandDetailRepo) FindDemandManagerReviews(ctx context.Context, demandID uint) ([]DemandManagerReviewItem, error) {
	exists, err := r.HasTable(ctx, "zt_demandmanagerreview")
	if err != nil || !exists {
		return nil, err
	}

	var raws []rawReview
	err = r.db.WithContext(ctx).Raw(`
SELECT id, COALESCE(result, '') AS result,
       COALESCE(reviewer, '') AS reviewer,
       COALESCE(comment, '') AS comment,
       COALESCE(submitedBy, '') AS submitedBy,
       submitedDate, reviewDate,
       COALESCE(resultStatus, '') AS resultStatus
FROM zt_demandmanagerreview
WHERE demand = ?
ORDER BY id DESC`, demandID).Scan(&raws).Error
	if err != nil {
		return nil, err
	}
	if len(raws) == 0 {
		return nil, nil
	}

	accounts := make([]string, 0, len(raws)*2)
	reviewIDs := make([]uint, 0, len(raws))
	for _, rw := range raws {
		accounts = append(accounts, rw.Reviewer, rw.SubmitedBy)
		reviewIDs = append(reviewIDs, rw.ID)
	}
	userMap := r.loadUserRealnames(ctx, accounts)
	detailMap := r.loadManagerReviewDetails(ctx, reviewIDs)
	return assembleManagerReviewItems(raws, userMap, detailMap), nil
}

func assembleManagerReviewItems(raws []rawReview, userMap map[string]string, detailMap map[uint]*reviewDetailAccumulator) []DemandManagerReviewItem {
	items := make([]DemandManagerReviewItem, 0, len(raws))
	for _, row := range raws {
		item := DemandManagerReviewItem{
			ID:             row.ID,
			Result:         row.Result,
			ResultLabel:    mapReviewResultLabel(row.Result),
			Reviewer:       row.Reviewer,
			ReviewerName:   resolveRealname(userMap, row.Reviewer),
			Comment:        row.Comment,
			SubmitedBy:     row.SubmitedBy,
			SubmitedByName: resolveRealname(userMap, row.SubmitedBy),
			SubmitedDate:   formatTime(row.SubmitedDate, "2006-01-02 15:04"),
			ReviewDate:     formatTime(row.ReviewDate, "2006-01-02 15:04"),
			ResultStatus:   row.ResultStatus,
		}
		if d, ok := detailMap[row.ID]; ok {
			item.Products = d.Products
			item.Departments = d.Departments
			item.DepartmentReviewers = d.DepartmentReviewers
		}
		items = append(items, item)
	}
	return items
}

type reviewDetailAccumulator struct {
	Products            []string
	Departments         []string
	DepartmentReviewers []string
}

func (r *DemandDetailRepo) loadManagerReviewDetails(ctx context.Context, reviewIDs []uint) map[uint]*reviewDetailAccumulator {
	res := make(map[uint]*reviewDetailAccumulator)
	if len(reviewIDs) == 0 {
		return res
	}
	detailExists, err := r.HasTable(ctx, "zt_demandmanagerreviewdetail")
	if err != nil || !detailExists {
		return res
	}

	type rawDetail struct {
		RID                uint   `gorm:"column:rid"`
		ProductName        string `gorm:"column:product_name"`
		Department         string `gorm:"column:department"`
		DepartmentReviewer string `gorm:"column:departmentReviewer"`
	}

	var details []rawDetail
	_ = r.db.WithContext(ctx).Raw(`
SELECT d.rid, COALESCE(p.name, '') AS product_name,
       COALESCE(d.department, '') AS department,
       COALESCE(d.departmentReviewer, '') AS departmentReviewer
FROM zt_demandmanagerreviewdetail d
LEFT JOIN zt_product p ON d.product = p.id AND p.deleted = '0'
WHERE d.rid IN (?)
ORDER BY d.id ASC`, reviewIDs).Scan(&details).Error

	for _, d := range details {
		acc, ok := res[d.RID]
		if !ok {
			acc = &reviewDetailAccumulator{}
			res[d.RID] = acc
		}
		if d.ProductName != "" {
			acc.Products = append(acc.Products, d.ProductName)
		}
		if d.Department != "" {
			acc.Departments = append(acc.Departments, d.Department)
		}
		if d.DepartmentReviewer != "" {
			acc.DepartmentReviewers = append(acc.DepartmentReviewers, d.DepartmentReviewer)
		}
	}
	return res
}
