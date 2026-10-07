// =============================================================================
// 文件: internal/module/po/repo_deliverydeadline.go
// 模块: PO 工作台
// 类型: repo
// 职责: 只读禅道常规窗口，共用评审截止日与超期条件。
// =============================================================================
package po

import (
	"context"
	"strings"
	"time"

	"workbench/internal/pkg/datefmt"
)

// unsetDeliveryDate 是禅道表示「未设日期」的哨兵值，按未设处理。
const unsetDeliveryDate = "2099-12-31"

// defaultDeliveryClock 是读不到禅道 deadlineTime 时的兜底截止时刻。
const defaultDeliveryClock = "16:30"

type deliveryDeadlineConfig struct {
	Windows string
	Date    string
	Clock   string
}

func (r *Repo) loadDeliveryDeadline(ctx context.Context) (deliveryDeadlineConfig, error) {
	cfg := deliveryDeadlineConfig{Clock: defaultDeliveryClock}
	var rows []struct {
		Section string
		Key     string
		Value   string
	}
	err := r.db.WithContext(ctx).Table("zt_config").Select("section, `key`, value").Where("owner = ? AND module = ? AND ((section = ? AND `key` = ?) OR (section = ? AND `key` IN ?))", "system", "demand", "regularChange", "WindowList", "delivery", []string{"deadlineDate", "deadlineTime"}).Find(&rows).Error
	for _, row := range rows {
		switch row.Key {
		case "WindowList":
			cfg.Windows = row.Value
		case "deadlineDate":
			cfg.Date = row.Value
		case "deadlineTime":
			if row.Value != "" {
				cfg.Clock = row.Value
			}
		}
	}
	return cfg, err
}

func (cfg deliveryDeadlineConfig) deadline(launch string) string {
	launch = datefmt.Raw(launch)
	if !validDeliveryDate(launch) {
		return ""
	}
	if cfg.Date != "" {
		return cfg.Date
	}
	for _, entry := range strings.Split(cfg.Windows, "|") {
		pair := strings.Split(entry, ",")
		if len(pair) == 2 && launch >= pair[0] && launch <= pair[1] {
			return pair[0]
		}
	}
	return ""
}

func validDeliveryDate(value string) bool {
	_, err := time.Parse(datefmt.Layout, value)
	return err == nil && value != unsetDeliveryDate
}

func (cfg deliveryDeadlineConfig) overdue(launch string, now time.Time) (string, bool, int) {
	date := cfg.deadline(launch)
	if !validDeliveryDate(date) {
		return "", false, 0
	}
	deadline, err := time.ParseInLocation(datefmt.Layout+" 15:04", date+" "+cfg.Clock, now.Location())
	if err != nil || !now.After(deadline) {
		return date, false, 0
	}
	return date, true, int(now.Sub(deadline).Hours()/24) + 1
}

func (cfg deliveryDeadlineConfig) overdueSQL(column string, now time.Time) (string, []interface{}) {
	if cfg.Date != "" && !validDeliveryDate(cfg.Date) {
		return "1 = 0", nil
	}
	sql := "CASE"
	args := []interface{}{}
	if validDeliveryDate(cfg.Date) {
		sql += " WHEN " + dateSetExpr(column) + " AND CAST(" + column + " AS CHAR) <> '" + unsetDeliveryDate + "' THEN ?"
		args = append(args, cfg.Date+" "+cfg.Clock)
	} else {
		for _, entry := range strings.Split(cfg.Windows, "|") {
			pair := strings.Split(entry, ",")
			if len(pair) != 2 || !validDeliveryDate(pair[0]) || !validDeliveryDate(pair[1]) {
				continue
			}
			sql += " WHEN " + column + " BETWEEN ? AND ? THEN ?"
			args = append(args, pair[0], pair[1], pair[0]+" "+cfg.Clock)
		}
	}
	if len(args) == 0 {
		return "1 = 0", nil
	}
	return sql + " ELSE NULL END < ?", append(args, now.Format("2006-01-02 15:04:05"))
}

func (cfg deliveryDeadlineConfig) homeOverdueOrder(now time.Time) (string, []interface{}) {
	// 需求与需求的 estimateLaunch 用同一套截止规则，故只生成一次条件与参数。
	overdueSQL, overdueArgs := cfg.overdueSQL("estimateLaunch", now)
	sql := "CASE WHEN (kind = 'demand' AND EXISTS (SELECT 1 FROM zt_demand WHERE id = focus_objects.id AND ((" + dateSetExpr("deadline") + " AND status NOT IN " + terminalStatusSQL + " AND deadline < ?) OR (status = 'acceptanced' AND (" + overdueSQL + "))))) OR (kind = 'story' AND EXISTS (SELECT 1 FROM zt_story WHERE id = focus_objects.id AND NOT " + storyDeliveredSQL + " AND (" + overdueSQL + "))) THEN 0 ELSE 1 END"
	args := append([]interface{}{now.Format(datefmt.Layout)}, overdueArgs...)
	return sql, append(args, overdueArgs...)
}
