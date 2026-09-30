// =============================================================================
// 文件: internal/module/po/repo_detail_mgmt.go
// 模块: PO 工作台
// 类型: repo
// 职责: 需求管理信息（7 项重要检查项与 8 项实际时间，含发布联查）数据读取。
// 依赖: gorm.io/gorm
// =============================================================================

package po

import (
	"context"
	"strings"
	"time"
)

// DemandManagementRow 需求管理项及时间原始行。
type DemandManagementRow struct {
	MultiLegalPersonLogo string     `gorm:"column:multiLegalPersonLogo"`
	IsRelatedAccounts    string     `gorm:"column:isRelatedAccounts"`
	IsImportantOrder     string     `gorm:"column:isImportantOrder"`
	IsNeedReview         string     `gorm:"column:isNeedReview"`
	OnetimeAcceptance    string     `gorm:"column:onetimeAcceptance"`
	IsCarReview          string     `gorm:"column:isCarReview"`
	VerifyDate           string     `gorm:"column:verifyDate"`
	ClarifyDate          *time.Time `gorm:"column:clarifyDate"`
	ActualDevStartDate   *time.Time `gorm:"column:actualDevStartDate"`
	ActualTestStartDate  *time.Time `gorm:"column:actualTestStartDate"`
	SubmitAcceptanceDate *time.Time `gorm:"column:submitAcceptanceDate"`
	AcceptancedDate      *time.Time `gorm:"column:acceptancedDate"`
	ReviewedDate         *time.Time `gorm:"column:reviewedDate"`
	DeliverDate          *time.Time `gorm:"column:deliverDate"`
}

type rawRelease struct {
	Date          *time.Time `gorm:"column:date"`
	ReleaseStatus string     `gorm:"column:releaseStatus"`
}

// FindDemandManagementRow 查询管理信息原始字段。
func (r *DemandDetailRepo) FindDemandManagementRow(ctx context.Context, demandID uint) (*DemandManagementRow, error) {
	if r == nil || r.db == nil {
		return nil, nil
	}
	var row DemandManagementRow
	err := r.db.WithContext(ctx).Raw(`
SELECT COALESCE(multiLegalPersonLogo, '') AS multiLegalPersonLogo,
       COALESCE(isRelatedAccounts, '') AS isRelatedAccounts,
       COALESCE(isImportantOrder, '') AS isImportantOrder,
       COALESCE(isNeedReview, '') AS isNeedReview,
       COALESCE(onetimeAcceptance, '') AS onetimeAcceptance,
       COALESCE(isCarReview, '') AS isCarReview,
       COALESCE(verifyDate, '') AS verifyDate,
       clarifyDate, actualDevStartDate, actualTestStartDate,
       submitAcceptanceDate, acceptancedDate, reviewedDate, deliverDate
FROM zt_demand
WHERE id = ? AND deleted = '0'
LIMIT 1`, demandID).Scan(&row).Error
	if err != nil {
		return nil, err
	}
	return &row, nil
}

// FindRealReleaseDate 联查研发需求与发布计算实际发布日期。
func (r *DemandDetailRepo) FindRealReleaseDate(ctx context.Context, demandID uint) (string, error) {
	if r == nil || r.db == nil {
		return "", nil
	}
	var pubIDs []string
	err := r.db.WithContext(ctx).Raw(`
SELECT associatedPublicationID
FROM zt_story
WHERE fromDemand = ? AND deleted = '0'`, demandID).Scan(&pubIDs).Error
	if err != nil || len(pubIDs) == 0 {
		return "", err
	}

	ids, ok := extractPublicationIDs(pubIDs)
	if !ok {
		return "", nil
	}

	var releases []rawRelease
	err = r.db.WithContext(ctx).Raw(`
SELECT date, releaseStatus
FROM zt_release
WHERE id IN (?) AND deleted = '0'`, ids).Scan(&releases).Error
	if err != nil || len(releases) == 0 {
		return "", err
	}

	latest := findLatestReleaseDate(releases)
	if latest == nil {
		return "", nil
	}
	return latest.Format("2006-01-02"), nil
}

func extractPublicationIDs(rawPubIDs []string) ([]string, bool) {
	ids := make([]string, 0, len(rawPubIDs))
	for _, p := range rawPubIDs {
		p = strings.TrimSpace(p)
		if p == "" || p == "0" {
			return nil, false // 有未关联发布的研发需求，按禅道规则视为未全部发布
		}
		ids = append(ids, p)
	}
	return ids, true
}

func findLatestReleaseDate(releases []rawRelease) *time.Time {
	var latest *time.Time
	for _, rel := range releases {
		if rel.ReleaseStatus != "online" && rel.ReleaseStatus != "toLaunched" {
			return nil
		}
		if rel.Date != nil && (latest == nil || rel.Date.After(*latest)) {
			latest = rel.Date
		}
	}
	return latest
}
