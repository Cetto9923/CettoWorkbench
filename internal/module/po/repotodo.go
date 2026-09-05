// =============================================================================
// 文件: internal/module/po/repotodo.go
// 模块: PO 工作台
// 类型: action
// 职责: 我的待办聚合查询（V10.1 02 节 7 维 AND）。本期打通业务需求 + 任务 + Bug；
//       研发需求/测试单/审批作为扩展点预留。
// 依赖: 无
// =============================================================================

package po

import (
	"context"
	"strings"
	"time"

	"gorm.io/gorm"
)

// FindTodoItems 查询我的待办列表（V10.1 02 节 7 维 AND 公式）。
// 数据真源:
//   - 业务需求 zt_demand (个人责任 scope: assignedTo/distributedBy/QD/RD/accepter/澄清 PM)
//   - 任务 zt_task (assignedTo=account)
//   - Bug zt_bug (assignedTo=account)
//
// 排序: 优先级 P1>P2>P3, deadline ASC NULLS LAST (zentao 用 '0000-00-00' 表示无 deadline,
//
//	用 '9999-12-31' 占位确保排最后), id DESC。
func (r *Repo) FindTodoItems(ctx context.Context, account string, req TodoListReq) ([]TodoItem, int64, error) {
	if r == nil || r.db == nil || strings.TrimSpace(account) == "" {
		return nil, 0, nil
	}
	res, err := r.QueryTodoUnified(ctx, account, req)
	if err != nil {
		return nil, 0, err
	}
	return res.Items, res.Total, nil
}

func demandTodoRelation(assignedTo, account string) (string, string) {
	if strings.TrimSpace(assignedTo) == strings.TrimSpace(account) {
		return "我负责", "待我处理"
	}
	return "我配合", "待我跟进"
}

func todoActionLabel(status string) string {
	switch status {
	case "draft", "wait", "refuse":
		return "受理"
	case "active":
		return "澄清"
	case "clarified":
		return "排期"
	case "developing", "testing":
		return "跟进"
	case "waitacceptance":
		return "验收"
	case "acceptanced":
		return "发起交付"
	case "waitdeliver":
		return "跟进发布"
	default:
		return "查看"
	}
}

// formatTodoDeadline 将禅道的零日期归为空，避免前端展示不存在的截止日。
func formatTodoDeadline(deadline *time.Time) string {
	if deadline == nil || deadline.Year() <= 1 {
		return ""
	}
	return deadline.Format("2006-01-02")
}

// applyDemandStageFilter 把 V10.1 价值流 10 阶段映射到 zt_demand.status SQL。
func applyDemandStageFilter(q *gorm.DB, stage string) *gorm.DB {
	switch stage {
	case "accept":
		return q.Where("status IN ?", []string{"draft", "wait", "refuse"})
	case "clarify":
		return q.Where("status = ? AND NOT EXISTS (SELECT 1 FROM zt_demandclarify dc WHERE dc.demand = zt_demand.id)", "active")
	case "schedule":
		return q.Where(`status = 'clarified' AND (
			developFinish IS NULL OR developFinish = '0000-00-00'
			OR testFinish IS NULL OR testFinish = '0000-00-00'
			OR verifyFinish IS NULL OR verifyFinish = '0000-00-00'
			OR estimateLaunch IS NULL OR estimateLaunch = '0000-00-00'
			OR QD = '' OR mainDevelopers = ''
		)`)
	case "developing":
		return q.Where("status = ?", "developing")
	case "testing":
		return q.Where("status = ?", "testing")
	case "waitacceptance":
		return q.Where(`(
			(status = 'testing')
			OR (status = 'waitacceptance')
		)`)
	case "acceptanced":
		return q.Where("status = ?", "acceptanced")
	case "released":
		return q.Where("status = ? AND overall = '0' AND parent != '-1'", "released")
	}
	return q
}

// applyTodoActionFilter 把 V10.1 办理场景映射到 zt_demand SQL。
func applyTodoActionFilter(q *gorm.DB, action TodoAction) *gorm.DB {
	switch action {
	case TodoActionReview:
		return q.Where("status IN ?", []string{"draft", "wait", "active", "refuse"})
	case TodoActionSchedule:
		return q.Where(`status = 'clarified' AND (
			developFinish IS NULL OR developFinish = '0000-00-00'
			OR testFinish IS NULL OR testFinish = '0000-00-00'
			OR verifyFinish IS NULL OR verifyFinish = '0000-00-00'
			OR estimateLaunch IS NULL OR estimateLaunch = '0000-00-00'
			OR QD = '' OR mainDevelopers = ''
		)`)
	case TodoActionVerify:
		return q.Where("status IN ?", []string{"testing", "waitacceptance"})
	case TodoActionDeliver:
		return q.Where("status = ?", "acceptanced")
	}
	return q
}

// loadAccountDisplayMap 加载 account → 展示名映射（zhentao 兼容）。
func (r *Repo) loadAccountDisplayMap(ctx context.Context) (map[string]string, error) {
	type row struct {
		Account  string `gorm:"column:account"`
		Realname string `gorm:"column:realname"`
	}
	var rows []row
	if err := r.db.WithContext(ctx).Table("zt_user").
		Where("deleted = ?", "0").
		Select("account, realname").
		Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make(map[string]string, len(rows))
	for _, row := range rows {
		if strings.TrimSpace(row.Account) == "" {
			continue
		}
		if strings.TrimSpace(row.Realname) == "" {
			out[row.Account] = row.Account
		} else {
			out[row.Account] = FormatAccountName(row.Account, row.Realname)
		}
	}
	return out, nil
}
