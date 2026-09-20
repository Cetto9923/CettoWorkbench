// =============================================================================
// 文件: internal/module/schedule/repo_scheduling_status.go
// 模块: 排期工作台
// 类型: action
// 职责: 排期保存时业需状态推进（clarified → developing）。
// 依赖: （无）
// =============================================================================

package schedule

import (
	"context"
	"errors"
	"strings"
	"time"
)

type demandStatusRow struct {
	Status             string `gorm:"column:status"`
	ActualDevStartDate string `gorm:"column:actualDevStartDate"`
}

// PromoteDemandToDevelopingIfClarified 保存研发需求后：业需状态为 clarified 时更新为 developing，
// 并按禅道 updateDemandStatus 规则回填 actualDevStartDate（为空时写今天）。
// 返回是否发生了状态变更。
func (r *Repo) PromoteDemandToDevelopingIfClarified(ctx context.Context, demandID uint, account string) (bool, error) {
	if demandID == 0 {
		return false, errors.New("业需 ID 无效")
	}

	const query = `
SELECT
  status,
  DATE_FORMAT(actualDevStartDate, '%Y-%m-%d') AS actualDevStartDate
FROM zt_demand
WHERE id = ? AND deleted = '0'
LIMIT 1`

	var row demandStatusRow
	if err := r.db.WithContext(ctx).Raw(query, demandID).Scan(&row).Error; err != nil {
		return false, err
	}
	if strings.TrimSpace(row.Status) != "clarified" {
		return false, nil
	}

	updates := map[string]interface{}{
		"status": "developing",
	}
	if formatZenTaoDate(row.ActualDevStartDate) == "" {
		updates["actualDevStartDate"] = time.Now().Format("2006-01-02")
	}
	if err := r.UpdateDemandScheduling(ctx, demandID, updates); err != nil {
		return false, err
	}
	// 对齐禅道 tostory：action=updateDeveloping, extra=story
	if err := r.CreateAction(ctx, "demand", demandID, "updateDeveloping", account, 0, 0, 0, "story"); err != nil {
		return false, err
	}
	return true, nil
}
