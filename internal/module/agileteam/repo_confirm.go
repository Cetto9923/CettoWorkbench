// =============================================================================
// 文件: internal/module/agileteam/repo_confirm.go
// 模块: 敏捷小组治理
// 类型: repository
// 职责: 将调整单状态变更、正式成员写入和历史记录放在事务边界内。
// =============================================================================

package agileteam

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
)

var errAdjustmentStateChanged = errors.New("agile adjustment state changed")

// ApplyConfirmedAdjustment 原子应用一张待确认调整单。
// 先用 status=pending 的条件更新抢占确认权；若 RowsAffected=0，则说明已被其它请求处理，
// 整个事务回滚，避免并发确认导致部分成员写入或重复历史。
func (r *Repo) ApplyConfirmedAdjustment(ctx context.Context, adj *Adjustment, items []AdjustmentItem, account string, now time.Time) error {
	if adj == nil || adj.ID <= 0 {
		return fmt.Errorf("invalid adjustment")
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		res := tx.Model(&Adjustment{}).
			Where("id = ? AND status = ?", adj.ID, StatusPending).
			Updates(map[string]any{
				"status":        StatusConfirmed,
				"confirmedBy":   account,
				"confirmedDate": now,
				"updatedBy":     account,
				"updatedDate":   now,
			})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected != 1 {
			return errAdjustmentStateChanged
		}

		for _, item := range items {
			switch item.ActionType {
			case ActionAdd, ActionRoleChange:
				if err := upsertTeamMemberTx(tx, adj.TeamgroupID, item.Account, item.Role, item.AvailableHours, now); err != nil {
					return err
				}
			case ActionRemove:
				if err := tx.Exec(
					"DELETE FROM zt_team WHERE type = 'teamgroup' AND root = ? AND account = ?",
					adj.TeamgroupID, item.Account,
				).Error; err != nil {
					return err
				}
			default:
				return fmt.Errorf("unsupported adjustment action: %s", item.ActionType)
			}
		}

		history := &History{
			TeamgroupID:  adj.TeamgroupID,
			EventType:    EventConfirm,
			AdjustmentID: &adj.ID,
			Actor:        account,
			Summary:      "成员调整 " + adj.AdjustNo + " 已确认生效",
			CreatedDate:  now,
		}
		return tx.Create(history).Error
	})
}

// TransitionPendingAdjustment 原子执行 pending -> rejected 等不涉及 zt_team 的状态流转，
// 并与历史记录同事务提交。所有状态流转都用 status=pending 条件更新，避免确认/驳回并发覆盖。
func (r *Repo) TransitionPendingAdjustment(ctx context.Context, adj *Adjustment, fields map[string]any, history *History) error {
	if adj == nil || adj.ID <= 0 {
		return fmt.Errorf("invalid adjustment")
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		res := tx.Model(&Adjustment{}).
			Where("id = ? AND status = ?", adj.ID, StatusPending).
			Updates(fields)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected != 1 {
			return errAdjustmentStateChanged
		}
		if history == nil {
			return nil
		}
		if history.CreatedDate.IsZero() {
			history.CreatedDate = time.Now()
		}
		return tx.Create(history).Error
	})
}

func upsertTeamMemberTx(tx *gorm.DB, teamgroupID uint, account, role string, hours float64, now time.Time) error {
	var count int64
	if err := tx.Table("zt_team").
		Where("type = 'teamgroup' AND root = ? AND account = ?", teamgroupID, account).
		Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return tx.Exec(`
UPDATE zt_team SET role = ?, hours = ?
WHERE type = 'teamgroup' AND root = ? AND account = ?`, role, hours, teamgroupID, account).Error
	}
	joinDate := now.Format("2006-01-02")
	return tx.Exec(`
INSERT INTO zt_team (root, type, teamgroup, account, role, position, limited, `+"`join`"+`, days, hours, estimate, consumed, `+"`left`"+`, `+"`order`"+`)
VALUES (?, 'teamgroup', 0, ?, ?, '', 'no', ?, 0, ?, 0, 0, 0, 0)`,
		teamgroupID, account, role, joinDate, hours).Error
}
