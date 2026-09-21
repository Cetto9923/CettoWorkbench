// =============================================================================
// 文件: internal/module/agileteam/repo_adjustment.go
// 模块: 敏捷小组治理
// 类型: action
// 职责: zt_wb_agileteam_adjustment* 与历史写入。
// 依赖: 无
// =============================================================================

package agileteam

import (
	"context"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
)

// CreateAdjustment 创建调整单 + 明细（事务）。
func (r *Repo) CreateAdjustment(ctx context.Context, adj *Adjustment, items []AdjustmentItem) (int64, error) {
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(adj).Error; err != nil {
			return err
		}
		for i := range items {
			items[i].AdjustmentID = adj.ID
			if err := tx.Create(&items[i]).Error; err != nil {
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

// FindAdjustmentByID 查调整单。
func (r *Repo) FindAdjustmentByID(ctx context.Context, id int64) (*Adjustment, error) {
	var row Adjustment
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

// ListItems 查调整明细。
func (r *Repo) ListItems(ctx context.Context, adjustmentID int64) ([]AdjustmentItem, error) {
	var rows []AdjustmentItem
	err := r.db.WithContext(ctx).
		Where("adjustmentId = ?", adjustmentID).
		Order("id ASC").
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	if rows == nil {
		rows = []AdjustmentItem{}
	}
	return rows, nil
}

// FindPendingByTeamgroup 查小组当前待确认调整单。
func (r *Repo) FindPendingByTeamgroup(ctx context.Context, teamgroupID uint) (*Adjustment, error) {
	var row Adjustment
	err := r.db.WithContext(ctx).
		Where("teamgroupId = ? AND status = ?", teamgroupID, StatusPending).
		Order("id DESC").
		First(&row).Error
	if err != nil {
		return nil, err
	}
	return &row, nil
}

// ListPendingByTeamgroupIDs 批量查待确认调整。
func (r *Repo) ListPendingByTeamgroupIDs(ctx context.Context, ids []uint) (map[uint]Adjustment, error) {
	out := map[uint]Adjustment{}
	if len(ids) == 0 {
		return out, nil
	}
	var rows []Adjustment
	err := r.db.WithContext(ctx).
		Where("teamgroupId IN ? AND status = ?", ids, StatusPending).
		Order("id ASC").
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	for _, it := range rows {
		// 同一小组只保留最新一条 pending（按 id 升序后覆盖）
		out[it.TeamgroupID] = it
	}
	return out, nil
}

// CountPendingItems 统计 pending 调整明细的 add/remove 数。
func (r *Repo) CountPendingItems(ctx context.Context, adjustmentIDs []int64) (map[int64]struct{ Add, Remove, Change int }, error) {
	out := map[int64]struct{ Add, Remove, Change int }{}
	if len(adjustmentIDs) == 0 {
		return out, nil
	}
	type row struct {
		AdjustmentID int64  `gorm:"column:adjustmentId"`
		ActionType   string `gorm:"column:actionType"`
		Cnt          int    `gorm:"column:cnt"`
	}
	var rows []row
	err := r.db.WithContext(ctx).Raw(`
SELECT adjustmentId, actionType, COUNT(*) AS cnt
FROM zt_wb_agileteam_adjustment_item
WHERE adjustmentId IN ? AND deletedAt IS NULL
GROUP BY adjustmentId, actionType`, adjustmentIDs).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	for _, it := range rows {
		cur := out[it.AdjustmentID]
		switch it.ActionType {
		case ActionAdd:
			cur.Add = it.Cnt
		case ActionRemove:
			cur.Remove = it.Cnt
		case ActionRoleChange:
			cur.Change = it.Cnt
		}
		out[it.AdjustmentID] = cur
	}
	return out, nil
}

// UpdateAdjustmentStatus 更新调整单状态。
func (r *Repo) UpdateAdjustmentStatus(ctx context.Context, id int64, fields map[string]any) error {
	return r.db.WithContext(ctx).Model(&Adjustment{}).Where("id = ?", id).Updates(fields).Error
}

// AppendHistory 追加历史。
func (r *Repo) AppendHistory(ctx context.Context, h *History) error {
	if h.CreatedDate.IsZero() {
		h.CreatedDate = time.Now()
	}
	return r.db.WithContext(ctx).Create(h).Error
}

// ListHistory 查时间线。
func (r *Repo) ListHistory(ctx context.Context, teamgroupID uint, limit int) ([]History, error) {
	if limit <= 0 {
		limit = 50
	}
	var rows []History
	err := r.db.WithContext(ctx).
		Where("teamgroupId = ?", teamgroupID).
		Order("id DESC").
		Limit(limit).
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	if rows == nil {
		rows = []History{}
	}
	return rows, nil
}

// LastAdjustTimes 各小组最近调整时间。
func (r *Repo) LastAdjustTimes(ctx context.Context, ids []uint) (map[uint]time.Time, error) {
	out := map[uint]time.Time{}
	if len(ids) == 0 {
		return out, nil
	}
	type row struct {
		TeamgroupID uint      `gorm:"column:teamgroupId"`
		MaxDate     time.Time `gorm:"column:maxDate"`
	}
	var rows []row
	err := r.db.WithContext(ctx).Raw(`
SELECT teamgroupId, MAX(createdDate) AS maxDate
FROM zt_wb_agileteam_adjustment
WHERE teamgroupId IN ? AND deletedAt IS NULL
GROUP BY teamgroupId`, ids).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	for _, it := range rows {
		out[it.TeamgroupID] = it.MaxDate
	}
	return out, nil
}

// NextAdjustNo 生成调整单号 TA+年月日+序号。
func (r *Repo) NextAdjustNo(ctx context.Context) (string, error) {
	prefix := "TA" + time.Now().Format("20060102")
	var count int64
	err := r.db.WithContext(ctx).
		Model(&Adjustment{}).
		Where("adjustNo LIKE ?", prefix+"%").
		Count(&count).Error
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s%03d", prefix, count+1), nil
}

// ListPendingAddGroupIDsForAccount 返回账号作为 pending 新增成员所在的小组 ID。
func (r *Repo) ListPendingAddGroupIDsForAccount(ctx context.Context, account string) ([]uint, error) {
	account = strings.TrimSpace(account)
	if account == "" {
		return nil, nil
	}
	var ids []uint
	err := r.db.WithContext(ctx).Raw(`
SELECT DISTINCT a.teamgroupId
FROM zt_wb_agileteam_adjustment_item i
INNER JOIN zt_wb_agileteam_adjustment a ON a.id = i.adjustmentId AND a.deletedAt IS NULL
WHERE a.status = ? AND i.actionType = ? AND i.account = ? AND i.deletedAt IS NULL`,
		StatusPending, ActionAdd, account).Scan(&ids).Error
	if err != nil {
		return nil, err
	}
	if ids == nil {
		ids = []uint{}
	}
	return ids, nil
}

// ListPendingAddMembers 返回各小组 pending 新增成员（看板合并用）。
func (r *Repo) ListPendingAddMembers(ctx context.Context, teamgroupIDs []uint) (map[uint][]AdjustmentItem, error) {
	out := map[uint][]AdjustmentItem{}
	if len(teamgroupIDs) == 0 {
		return out, nil
	}
	type row struct {
		TeamgroupID    uint    `gorm:"column:teamgroupId"`
		Account        string  `gorm:"column:account"`
		Role           string  `gorm:"column:role"`
		AvailableHours float64 `gorm:"column:availableHours"`
		SubmittedBy    string  `gorm:"column:submittedBy"`
	}
	var rows []row
	err := r.db.WithContext(ctx).Raw(`
SELECT a.teamgroupId, i.account, i.role, i.availableHours, a.submittedBy
FROM zt_wb_agileteam_adjustment_item i
INNER JOIN zt_wb_agileteam_adjustment a ON a.id = i.adjustmentId AND a.deletedAt IS NULL
WHERE a.status = ? AND a.teamgroupId IN ? AND i.actionType = ? AND i.deletedAt IS NULL`,
		StatusPending, teamgroupIDs, ActionAdd).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	for _, it := range rows {
		out[it.TeamgroupID] = append(out[it.TeamgroupID], AdjustmentItem{
			Account: it.Account, Role: it.Role, AvailableHours: it.AvailableHours,
			CreatedBy: it.SubmittedBy, ActionType: ActionAdd,
		})
	}
	return out, nil
}

// ListPendingRemoveAccounts 返回各小组 pending 移除账号。
func (r *Repo) ListPendingRemoveAccounts(ctx context.Context, teamgroupIDs []uint) (map[uint]map[string]bool, error) {
	out := map[uint]map[string]bool{}
	if len(teamgroupIDs) == 0 {
		return out, nil
	}
	type row struct {
		TeamgroupID uint   `gorm:"column:teamgroupId"`
		Account     string `gorm:"column:account"`
	}
	var rows []row
	err := r.db.WithContext(ctx).Raw(`
SELECT a.teamgroupId, i.account
FROM zt_wb_agileteam_adjustment_item i
INNER JOIN zt_wb_agileteam_adjustment a ON a.id = i.adjustmentId AND a.deletedAt IS NULL
WHERE a.status = ? AND a.teamgroupId IN ? AND i.actionType = ? AND i.deletedAt IS NULL`,
		StatusPending, teamgroupIDs, ActionRemove).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	for _, it := range rows {
		if out[it.TeamgroupID] == nil {
			out[it.TeamgroupID] = map[string]bool{}
		}
		out[it.TeamgroupID][it.Account] = true
	}
	return out, nil
}
