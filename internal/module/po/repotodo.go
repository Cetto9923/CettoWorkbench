// =============================================================================
// 文件: internal/module/po/repotodo.go
// 模块: PO 工作台
// 类型: action
// 职责: 待办共用辅助（CountOpenTodos、关系/截止日/优先级标签、账号展示名映射）。
//       列表真源已迁至 QueryTodoUnified / todo_query_*.go。
// 依赖: 无
// =============================================================================

package po

import (
	"context"
	"strings"
	"time"
)

// CountOpenTodos 仅返回当前账号待办总数（不取 items）。
// 与 FindTodoItems 在空 req 下的总数等价，但避免 N 行 SQL 回传。
func (r *Repo) CountOpenTodos(ctx context.Context, account string) (int64, error) {
	if r == nil || r.db == nil || strings.TrimSpace(account) == "" {
		return 0, nil
	}
	res, err := r.QueryTodoUnified(ctx, account, TodoListReq{Page: 1, PageSize: 0})
	if err != nil {
		return 0, err
	}
	return res.Total, nil
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

// issueRiskPriLabel 把 zt_issue/zt_risk.pri（char(30)，混存数字串与 low/middle/high/urgent）映射为 P1..P4。
// zentao 优先级语义: 1/urgent 最高 → P1，4/low 最低 → P4；无法识别返回空。
func issueRiskPriLabel(pri string) string {
	switch strings.ToLower(strings.TrimSpace(pri)) {
	case "1", "urgent", "immediate":
		return "P1"
	case "2", "high":
		return "P2"
	case "3", "middle", "medium":
		return "P3"
	case "4", "low":
		return "P4"
	}
	return ""
}
