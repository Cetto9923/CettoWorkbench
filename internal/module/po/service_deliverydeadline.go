// =============================================================================
// 文件: internal/module/po/service_deliverydeadline.go
// 模块: PO 工作台
// 类型: service
// 职责: 首页超期标记使用同一份禅道评审截止配置。
// =============================================================================
package po

import (
	"context"
	"time"
)

func (s *Service) enrichHomeDeadlines(ctx context.Context, items []WorkItemDetail, demands map[int]DemandRow, stories map[int]StoryRow) error {
	if len(items) == 0 {
		return nil
	}
	cfg, err := s.repo.loadDeliveryDeadline(ctx)
	if err != nil {
		return err
	}
	now := time.Now()
	for i := range items {
		id := parseDemandNumericID(items[i].ID)
		launch := ""
		if items[i].Kind == "story" {
			row := stories[id]
			if row.Delivered {
				items[i].Overdue = false
				items[i].OverdueDays = 0
				continue
			}
			launch = row.EstimateLaunch
		} else {
			row := demands[id]
			if row.Status != "acceptanced" {
				continue
			}
			launch = row.EstimateLaunch
		}
		date, overdue, days := cfg.overdue(launch, now)
		if items[i].Kind == "story" || overdue && !items[i].Overdue {
			items[i].Deadline, items[i].Overdue, items[i].OverdueDays = date, overdue, days
		}
	}
	return nil
}
