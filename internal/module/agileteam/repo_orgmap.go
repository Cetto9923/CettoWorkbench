package agileteam

import (
	"context"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"

	"workbench/internal/pkg/errorx"
)

// SetOrgTeamMapping 锁定敏捷小组行后维护组织挂靠；deptID 为 0 表示解除挂靠。
func (r *Repo) SetOrgTeamMapping(ctx context.Context, teamgroupID, deptID uint, actor string) error {
	actor = strings.TrimSpace(actor)
	if teamgroupID == 0 || actor == "" {
		return fmt.Errorf("敏捷小组和维护人不能为空")
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var groupID uint
		if err := tx.Raw(`SELECT id FROM zt_teamgroup WHERE id = ? AND deleted = '0' FOR UPDATE`, teamgroupID).Scan(&groupID).Error; err != nil {
			return err
		}
		if groupID == 0 {
			return errorx.New("not_found", "敏捷小组不存在")
		}
		if deptID > 0 {
			var leafID uint
			if err := tx.Raw(`
SELECT d.id FROM zt_dept d
WHERE d.id = ?
  AND NOT EXISTS (SELECT 1 FROM zt_dept child WHERE child.parent = d.id)
LIMIT 1`, deptID).Scan(&leafID).Error; err != nil {
				return err
			}
			if leafID == 0 {
				return fmt.Errorf("组织团队不存在或不是最底层部门")
			}
		}
		if err := tx.Exec(`
UPDATE zt_wb_agileteam_orgmap
SET status = 'inactive', updatedBy = ?, updatedDate = NOW()
WHERE teamgroupId = ? AND status = 'active'`, actor, teamgroupID).Error; err != nil {
			return err
		}
		if deptID > 0 {
			if err := tx.Exec(`
INSERT INTO zt_wb_agileteam_orgmap (teamgroupId, deptId, status, createdBy, updatedBy)
VALUES (?, ?, 'active', ?, ?)`, teamgroupID, deptID, actor, actor).Error; err != nil {
				return err
			}
		}
		summary := "已解除敏捷小组与组织团队挂靠"
		if deptID > 0 {
			summary = fmt.Sprintf("敏捷小组已挂靠组织部门 #%d", deptID)
		}
		return tx.Exec(`INSERT INTO zt_wb_agileteam_history (teamgroupId, eventType, summary, actor, createdDate) VALUES (?, ?, ?, ?, ?)`, teamgroupID, EventOrgTeamMapping, summary, actor, time.Now()).Error
	})
}
