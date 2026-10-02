// =============================================================================
// 文件: internal/pkg/database/schema.go
// 模块: 基础设施
// 类型: repo
// 职责: 启动时只读验证工作台字段，禁止运行期 DDL。
// 依赖: gorm.io/gorm
// =============================================================================

package database

import (
	"fmt"
	"gorm.io/gorm"
	"strings"
)

func CheckSchema(db *gorm.DB) error {
	required := map[string]string{
		"zt_operation_logs":               "id tenantId userId account method path query body ip userAgent statusCode createdAt",
		"zt_wb_dept_manager_override":     "id dept account remark createdBy updatedBy createdDate updatedDate deleted",
		"zt_wb_agileteam_adjustment":      "id teamgroupId adjustNo status reason submittedBy confirmedBy confirmedDate rejectedBy rejectedDate rejectReason createdBy updatedBy createdDate updatedDate deletedAt",
		"zt_wb_agileteam_adjustment_item": "id adjustmentId account actionType role prevRole availableHours prevHours createdBy updatedBy createdDate updatedDate deletedAt",
		"zt_wb_agileteam_history":         "id teamgroupId eventType adjustmentId summary actor createdDate",
	}
	tables := make([]string, 0, len(required))
	for table := range required {
		tables = append(tables, table)
	}
	var rows []struct {
		Table  string `gorm:"column:table_name"`
		Column string `gorm:"column:column_name"`
	}
	if err := db.Raw(`SELECT table_name, column_name FROM information_schema.columns WHERE table_schema = DATABASE() AND table_name IN ?`, tables).Scan(&rows).Error; err != nil {
		return fmt.Errorf("检查工作台表结构失败：%w", err)
	}
	found := make(map[string]map[string]bool, len(tables))
	for _, row := range rows {
		if found[row.Table] == nil {
			found[row.Table] = map[string]bool{}
		}
		found[row.Table][row.Column] = true
	}
	for table, columns := range required {
		for _, column := range strings.Fields(columns) {
			if !found[table][column] {
				return fmt.Errorf("工作台表 %s 缺少字段 %s，请先执行已审核的安装或升级脚本", table, column)
			}
		}
	}
	return nil
}
