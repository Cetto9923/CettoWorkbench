// =============================================================================
// 文件: internal/module/po/repodone_authz.go
// 模块: PO 工作台
// 类型: repo
// 职责: 我的已办动作级授权所需的最小投影（FindDoneActionActor /
//       CheckDoneActionObjectVisibility），与 repodone_enrich.go 的批量
//       组装职责分离，便于 F01 测试聚焦授权路径。授权决策由 Service
//       负责，Repo 只返回候选行。
// =============================================================================

package po

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"
)

// FindDoneActionActor 返回 zt_action 的 actor 字段；不存在返回 gorm.ErrRecordNotFound。
//
// 用途：F01 DoneDetail 授权预查询，Service 据此区分「动作 actor 自身」与
// 「需要进一步对象级授权检查」两种路径。
func (r *Repo) FindDoneActionActor(ctx context.Context, actionID int64) (string, error) {
	if r == nil || r.db == nil {
		return "", fmt.Errorf("done repo is not configured")
	}
	if actionID <= 0 {
		return "", fmt.Errorf("invalid action id")
	}
	var row struct {
		Actor string `gorm:"column:actor"`
	}
	err := r.db.WithContext(ctx).Table("zt_action").
		Select("actor").Where("id = ?", actionID).Scan(&row).Error
	if err != nil {
		return "", err
	}
	if row.Actor == "" {
		return "", gorm.ErrRecordNotFound
	}
	return row.Actor, nil
}

// CheckDoneActionObjectVisibility 判断 actor.Account 是否与动作对象存在
// 至少一条 PO/责任/提出/测试/验收/评审/创建/闭环关系（不依赖动作 actor）。
//
// 仅做候选行投影，不做最终放行决策：Service 仍需根据 IsSuperAdmin、动作
// actor 等再判一次。当前仅支持需求类对象（demand / story / task / bug），
// 其他类型对象（todo / risk / issue / feedback / release / build）按现有
// 「仅动作 actor 自身可见」处理，超出本批次范围。
func (r *Repo) CheckDoneActionObjectVisibility(ctx context.Context, actionID int64, account string) (bool, error) {
	if r == nil || r.db == nil {
		return false, fmt.Errorf("done repo is not configured")
	}
	if actionID <= 0 || account == "" {
		return false, nil
	}

	var action struct {
		ObjectType string `gorm:"column:objectType"`
		ObjectID   int64  `gorm:"column:objectID"`
	}
	if err := r.db.WithContext(ctx).Table("zt_action").
		Select("objectType, objectID").Where("id = ?", actionID).Scan(&action).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, err
	}
	if action.ObjectID == 0 {
		return false, nil
	}

	switch action.ObjectType {
	case "demand":
		var hits int64
		err := r.db.WithContext(ctx).Raw(`
SELECT COUNT(*) FROM zt_demand d
WHERE d.id = ? AND d.deleted = '0'
  AND (
    d.originator = ? OR d.assignedTo = ? OR d.QD = ? OR d.BRA = ? OR
    d.accepter = ? OR d.reviewer = ? OR d.createdBy = ? OR
    d.closedBy = ? OR d.editedBy = ? OR d.feedbackedBy = ?
  )`, action.ObjectID, account, account, account, account, account, account,
			account, account, account, account).Scan(&hits).Error
		if err != nil {
			return false, err
		}
		return hits > 0, nil
	case "story":
		var hits int64
		err := r.db.WithContext(ctx).Raw(`
SELECT COUNT(*) FROM zt_story s
WHERE s.id = ? AND s.deleted = '0'
  AND (s.openedBy = ? OR s.assignedTo = ? OR s.reviewedBy = ?
       OR s.closedBy = ? OR s.lastEditedBy = ? OR s.fromDemand IN (
           SELECT d.id FROM zt_demand d WHERE d.deleted='0' AND (
             d.originator = ? OR d.assignedTo = ? OR d.QD = ? OR d.BRA = ? OR
             d.accepter = ? OR d.reviewer = ? OR d.createdBy = ? OR
             d.closedBy = ? OR d.editedBy = ? OR d.feedbackedBy = ?
           )
       ))`, action.ObjectID, account, account, account, account, account,
			account, account, account, account, account, account,
			account, account, account).Scan(&hits).Error
		if err != nil {
			return false, err
		}
		return hits > 0, nil
	case "task":
		var hits int64
		err := r.db.WithContext(ctx).Raw(`
SELECT COUNT(*) FROM zt_task t
WHERE t.id = ? AND t.deleted = '0'
  AND (t.openedBy = ? OR t.assignedTo = ? OR t.closedBy = ?
       OR t.finishedBy = ? OR t.canceledBy = ? OR t.lastEditedBy = ?)`,
			action.ObjectID, account, account, account, account, account, account).
			Scan(&hits).Error
		if err != nil {
			return false, err
		}
		return hits > 0, nil
	case "bug":
		var hits int64
		err := r.db.WithContext(ctx).Raw(`
SELECT COUNT(*) FROM zt_bug b
WHERE b.id = ? AND b.deleted = '0'
  AND (b.openedBy = ? OR b.assignedTo = ? OR b.resolvedBy = ?
       OR b.closedBy = ? OR b.lastEditedBy = ?)`,
			action.ObjectID, account, account, account, account, account).
			Scan(&hits).Error
		if err != nil {
			return false, err
		}
		return hits > 0, nil
	default:
		// todo/risk/issue/feedback/release/build 等非典型对象类型，按
		// 「仅动作 actor 自身可见」处理，避免本次以外的对象类型被无依据放行。
		return false, nil
	}
}
