// =============================================================================
// 文件: internal/module/po/repofocus.go
// 模块: PO 工作台
// 类型: action
// 职责: 价值流焦点叠加（超期/我负责/待我处理）与独立研需 scope。
// 依赖: 无
// =============================================================================

package po

import (
	"context"
	"strings"
	"time"

	"gorm.io/gorm"
)

// 待我处理分阶段模式（静态写在 mysqlStageFilters；仅 filter.myAction 时生效）。
const (
	myActionAll               = "all"                 // 澄清/排期：不加额外条件
	myActionCreatedOrReviewer = "created_or_reviewer" // 受理：创建人或评审人
	myActionBRA               = "bra"                 // 提测/发起交付/发布：BRA
	myActionQDOrAssigned      = "qd_or_assigned"      // 联调测试：QD 或指派人
	myActionAcceptance        = "acceptance"          // 验收：BRA / 澄清 PM / waitacceptance+RD
	myActionOriginator        = "originator"          // 评价反馈：提出人
)

// applyDemandOverdue 业需超期：acceptanced 看 estimateLaunch；其余非终态看 deadline。
func applyDemandOverdue(q *gorm.DB) *gorm.DB {
	today := time.Now().Format("2006-01-02")
	return q.Where(`(
		(status = ? AND `+dateSetExpr("estimateLaunch")+` AND estimateLaunch < ?)
		OR (status NOT IN ? AND `+dateSetExpr("deadline")+` AND deadline < ?)
	)`, "acceptanced", today, []string{"acceptanced", "closed", "refuse", "released"}, today)
}

// applyDemandMyManaged 业需「我负责」：指派给我或 BRA 是我。
func applyDemandMyManaged(q *gorm.DB, account string) *gorm.DB {
	return q.Where("(assignedTo = ? OR BRA = ?)", account, account)
}

// applyDemandMyAction 业需「待我处理」：按阶段 myActionMode 叠加。
func applyDemandMyAction(q *gorm.DB, account, mode string) *gorm.DB {
	switch mode {
	case myActionAll, "":
		return q
	case myActionCreatedOrReviewer:
		return q.Where("(createdBy = ? OR FIND_IN_SET(?, reviewer) > 0)", account, account)
	case myActionBRA:
		return q.Where("BRA = ?", account)
	case myActionQDOrAssigned:
		return q.Where("(QD = ? OR assignedTo = ?)", account, account)
	case myActionAcceptance:
		return q.Where(`(
			BRA = ?
			OR id IN (SELECT demand FROM zt_demandclarify WHERE PM = ?)
			OR (status = ? AND RD = ?)
		)`, account, account, "waitacceptance", account)
	case myActionOriginator:
		return q.Where("originator = ?", account)
	default:
		return q
	}
}

// applyStoryOverdue 独立研需超期：阶段非 released/closed 且 estimateLaunch < 今天。
func applyStoryOverdue(q *gorm.DB) *gorm.DB {
	today := time.Now().Format("2006-01-02")
	return q.Where("stage NOT IN ?", []string{"released", "closed"}).
		Where(dateSetExpr("estimateLaunch")+" AND estimateLaunch < ?", today)
}

// storyAssignedOrProductReqM 指派人或所属产品的需求负责人（zt_product.ReqM）为当前用户。
const storyAssignedOrProductReqM = `(zt_story.assignedTo = ? OR EXISTS (
	SELECT 1 FROM zt_product p
	WHERE p.id = zt_story.product AND p.deleted = '0' AND p.ReqM = ?
))`

// storyProductReqM 所属产品的需求负责人（zt_product.ReqM）为当前用户。
const storyProductReqM = `EXISTS (
	SELECT 1 FROM zt_product p
	WHERE p.id = zt_story.product AND p.deleted = '0' AND p.ReqM = ?
)`

// scheduleStoryScope 排期阶段独立研发需求：非需求池、非父需求、排除已关闭、指派人或所属产品 ReqM 为当前用户、关键日期未填。
func (r *Repo) scheduleStoryScope(ctx context.Context, account string, filter mysqlStageFilter) *gorm.DB {
	q := r.db.WithContext(ctx).Table("zt_story").
		Where("deleted = ?", "0").
		Where("status != ?", "closed").
		Where("IFNULL(sourceType, '') != ?", "demandpool").
		Where("type = ?", "story").
		Where("isParent = ?", "0").
		Where("product != ?", "0").
		Where(storyAssignedOrProductReqM, account, account).
		Where("(" + strings.Join([]string{
			dateUnsetExpr("developFinish"),
			dateUnsetExpr("testFinish"),
			dateUnsetExpr("verifyFinish"),
		}, " OR ") + ")")
	if filter.overdue {
		q = applyStoryOverdue(q)
	}
	return q
}

// deliverStoryScope 交付阶段独立研发需求：非需求池、非父需求、排除已关闭；
// 默认指派人或 ReqM；待我处理时仅 product.ReqM；今天 >= deliverDate。
func (r *Repo) deliverStoryScope(ctx context.Context, account string, filter mysqlStageFilter) *gorm.DB {
	today := time.Now().Format("2006-01-02")
	q := r.db.WithContext(ctx).Table("zt_story").
		Where("deleted = ?", "0").
		Where("status != ?", "closed").
		Where("IFNULL(sourceType, '') != ?", "demandpool").
		Where("type = ?", "story").
		Where("isParent = ?", "0")
	if filter.myAction {
		q = q.Where(storyProductReqM, account)
	} else {
		q = q.Where(storyAssignedOrProductReqM, account, account)
	}
	q = q.Where(dateSetExpr("deliverDate")+" AND deliverDate <= ?", today)
	if filter.overdue {
		q = applyStoryOverdue(q)
	}
	return q
}

// CountScheduleStories 统计排期阶段独立研发需求数量。
func (r *Repo) CountScheduleStories(ctx context.Context, account string, filter mysqlStageFilter) (int64, error) {
	if r == nil || r.db == nil || strings.TrimSpace(account) == "" {
		return 0, nil
	}
	var total int64
	err := r.scheduleStoryScope(ctx, account, filter).Count(&total).Error
	return total, err
}

// FindScheduleStoryIDs 查询排期阶段独立研发需求 ID。
func (r *Repo) FindScheduleStoryIDs(ctx context.Context, account string, filter mysqlStageFilter) ([]int, error) {
	if r == nil || r.db == nil || strings.TrimSpace(account) == "" {
		return nil, nil
	}
	var ids []int
	err := r.scheduleStoryScope(ctx, account, filter).
		Order("id DESC").
		Pluck("id", &ids).Error
	if err != nil {
		return nil, err
	}
	return ids, nil
}

// FindScheduleStories 查询排期阶段独立研发需求列表。
func (r *Repo) FindScheduleStories(ctx context.Context, account string, filter mysqlStageFilter) ([]StoryRow, error) {
	if r == nil || r.db == nil || strings.TrimSpace(account) == "" {
		return nil, nil
	}
	var rows []StoryRow
	err := r.scheduleStoryScope(ctx, account, filter).
		Select("id", "title", "pri", "status", "stage").
		Order("id DESC").
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	return rows, nil
}

// CountDeliverStories 统计交付阶段独立研发需求数量。
func (r *Repo) CountDeliverStories(ctx context.Context, account string, filter mysqlStageFilter) (int64, error) {
	if r == nil || r.db == nil || strings.TrimSpace(account) == "" {
		return 0, nil
	}
	var total int64
	err := r.deliverStoryScope(ctx, account, filter).Count(&total).Error
	return total, err
}

// FindDeliverStoryIDs 查询交付阶段独立研发需求 ID。
func (r *Repo) FindDeliverStoryIDs(ctx context.Context, account string, filter mysqlStageFilter) ([]int, error) {
	if r == nil || r.db == nil || strings.TrimSpace(account) == "" {
		return nil, nil
	}
	var ids []int
	err := r.deliverStoryScope(ctx, account, filter).
		Order("id DESC").
		Pluck("id", &ids).Error
	if err != nil {
		return nil, err
	}
	return ids, nil
}

// FindDeliverStories 查询交付阶段独立研发需求列表。
func (r *Repo) FindDeliverStories(ctx context.Context, account string, filter mysqlStageFilter) ([]StoryRow, error) {
	if r == nil || r.db == nil || strings.TrimSpace(account) == "" {
		return nil, nil
	}
	var rows []StoryRow
	err := r.deliverStoryScope(ctx, account, filter).
		Select("id", "title", "pri", "status", "stage").
		Order("id DESC").
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	return rows, nil
}
