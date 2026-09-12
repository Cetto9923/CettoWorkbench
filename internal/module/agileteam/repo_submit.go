// =============================================================================
// 文件: internal/module/agileteam/repo_submit.go
// 模块: 敏捷小组治理
// 类型: repository
// 职责: 原子创建成员调整单、明细与提交历史。
// =============================================================================

package agileteam

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	mysql "github.com/go-sql-driver/mysql"
	"gorm.io/gorm"
)

var errAdjustNoConflict = errors.New("调整单号冲突，请重试")

const maxAdjustNoAttempts = 3

func nextAdjustNoTx(tx *gorm.DB) (string, error) {
	prefix := "TA" + time.Now().Format("20060102")
	var last string
	if err := tx.Model(&Adjustment{}).
		Select("adjustNo").
		Where("adjustNo LIKE ?", prefix+"%").
		Order("adjustNo DESC").
		Limit(1).
		Scan(&last).Error; err != nil {
		return "", err
	}
	seq := 1
	if last != "" {
		suffix := strings.TrimPrefix(last, prefix)
		if n, err := strconv.Atoi(suffix); err == nil {
			seq = n + 1
		}
	}
	return fmt.Sprintf("%s%03d", prefix, seq), nil
}

func isAdjustNoDuplicate(err error) bool {
	if err == nil {
		return false
	}
	var mysqlErr *mysql.MySQLError
	if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
		msg := strings.ToLower(mysqlErr.Message)
		return strings.Contains(msg, "adjustno") || strings.Contains(msg, "uk_adjustno")
	}
	lower := strings.ToLower(err.Error())
	return strings.Contains(lower, "1062") && (strings.Contains(lower, "adjustno") || strings.Contains(lower, "uk_adjustno"))
}

// SubmitPendingAdjustmentAtomically 在同一事务内锁小组、覆盖旧 pending、生成单号并写入调整单。
// 同一 teamgroup 的提交因此串行；同组允许多次提交，但只有最后一张保持 pending。
// 跨 teamgroup 仍可能撞号；uk_adjustNo 冲突时有限重试，最终返回受控 conflict。
func (r *Repo) SubmitPendingAdjustmentAtomically(ctx context.Context, adj *Adjustment, items []AdjustmentItem, history *History) (int64, error) {
	if adj == nil || adj.TeamgroupID == 0 {
		return 0, fmt.Errorf("invalid adjustment")
	}
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var current teamgroupLockRow
		if err := tx.Raw(`
SELECT id, parent, grade, COALESCE(path, '') AS path
FROM zt_teamgroup
WHERE id = ? AND deleted = '0'
LIMIT 1 FOR UPDATE`, adj.TeamgroupID).Scan(&current).Error; err != nil {
			return err
		}
		if current.ID == 0 {
			return errTeamgroupMissing
		}

		if err := tx.Model(&Adjustment{}).
			Where("teamgroupId = ? AND status = ?", adj.TeamgroupID, StatusPending).
			Updates(map[string]any{
				"status":      StatusSuperseded,
				"updatedBy":   adj.SubmittedBy,
				"updatedDate": adj.CreatedDate,
			}).Error; err != nil {
			return err
		}

		var created bool
		for attempt := 0; attempt < maxAdjustNoAttempts; attempt++ {
			no, err := nextAdjustNoTx(tx)
			if err != nil {
				return err
			}
			adj.ID = 0
			adj.AdjustNo = no
			if err := tx.Create(adj).Error; err != nil {
				if isAdjustNoDuplicate(err) && attempt+1 < maxAdjustNoAttempts {
					continue
				}
				if isAdjustNoDuplicate(err) {
					return errAdjustNoConflict
				}
				return err
			}
			created = true
			break
		}
		if !created {
			return errAdjustNoConflict
		}
		for i := range items {
			items[i].AdjustmentID = adj.ID
			if err := tx.Create(&items[i]).Error; err != nil {
				return err
			}
		}
		if history != nil {
			history.AdjustmentID = &adj.ID
			if history.TeamgroupID == 0 {
				history.TeamgroupID = adj.TeamgroupID
			}
			if history.CreatedDate.IsZero() {
				history.CreatedDate = time.Now()
			}
			if history.Summary == "" {
				history.Summary = fmt.Sprintf("成员调整 %s 已提交：新增%d / 移除%d / 角色调整%d",
					adj.AdjustNo, countAction(items, ActionAdd), countAction(items, ActionRemove), countAction(items, ActionRoleChange))
			}
			if err := tx.Create(history).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return 0, err
	}
	return adj.ID, nil
}

// CreateAdjustmentWithHistory 保留给旧调用点；实现已收口到带锁的 pending 提交。
func (r *Repo) CreateAdjustmentWithHistory(ctx context.Context, adj *Adjustment, items []AdjustmentItem, history *History) (int64, error) {
	return r.SubmitPendingAdjustmentAtomically(ctx, adj, items, history)
}
