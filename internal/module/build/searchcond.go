// =============================================================================
// 文件: internal/module/build/searchcond.go
// 模块: 版本管理
// 类型: action
// 职责: 关联需求 bySearch 条件白名单与参数化 WHERE 片段（对齐禅道 search::setWhere）。
// 依赖: 无
// =============================================================================

package build

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

// SearchCond 单组用户搜索条件。
type SearchCond struct {
	Field    string
	Operator string
	Value    string
}

var allowedOperators = map[string]struct{}{
	"=": {}, "!=": {}, ">": {}, ">=": {}, "<": {}, "<=": {},
	"include": {}, "notinclude": {}, "belong": {}, "between": {},
}

// 对齐禅道 product/search 字段；linkStory 去掉 product。
var allowedSearchFields = map[string]string{
	"title": "s.title", "id": "s.id", "keywords": "s.keywords",
	"status": "s.status", "pri": "s.pri", "module": "s.module",
	"stage": "s.stage", "branch": "s.branch", "grade": "s.grade",
	"plan": "s.plan", "estimate": "s.estimate",
	"source": "s.source", "sourceNote": "s.sourceNote", "fromBug": "s.fromBug",
	"category": "s.category",
	"openedBy": "s.openedBy", "reviewedBy": "s.reviewedBy", "assignedTo": "s.assignedTo",
	"closedBy": "s.closedBy", "lastEditedBy": "s.lastEditedBy",
	"mailto": "s.mailto", "closedReason": "s.closedReason", "version": "s.version",
	"openedDate": "s.openedDate", "reviewedDate": "s.reviewedDate",
	"assignedDate": "s.assignedDate", "closedDate": "s.closedDate",
	"lastEditedDate": "s.lastEditedDate", "activatedDate": "s.activatedDate",
	"estimateLaunch": "s.estimateLaunch", "deliverDate": "s.deliverDate",
	"isMainSystemAssociation": "s.isMainSystemAssociation",
	"isCarReview": "s.isCarReview",
	"result": "result", // 特殊：EXISTS 子查询
}

var dateOnlyRe = regexp.MustCompile(`^\d{4}-\d{1,2}-\d{1,2}$`)
var fieldNameRe = regexp.MustCompile(`^[a-zA-Z0-9]+$`)

// NormalizeOperator 非法运算符回退为 =。
func NormalizeOperator(op string) string {
	op = strings.TrimSpace(op)
	if _, ok := allowedOperators[op]; ok {
		return op
	}
	return "="
}

// NormalizeAndOr 仅允许 and/or。
func NormalizeAndOr(v string) string {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "or":
		return "or"
	default:
		return "and"
	}
}

// IsAllowedSearchField 字段白名单（含字母数字校验）。
func IsAllowedSearchField(field string) bool {
	field = strings.TrimSpace(field)
	if field == "" || !fieldNameRe.MatchString(field) {
		return false
	}
	_, ok := allowedSearchFields[field]
	return ok
}

// ResolveDynamicDate 解析禅道日期动态变量。
func ResolveDynamicDate(value string, now time.Time) (begin, end time.Time, ok bool) {
	value = strings.TrimSpace(value)
	dayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	switch value {
	case "$today":
		return dayStart, dayStart.Add(24*time.Hour - time.Second), true
	case "$yesterday":
		y := dayStart.AddDate(0, 0, -1)
		return y, y.Add(24*time.Hour - time.Second), true
	case "$thisWeek":
		weekday := int(now.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		begin = dayStart.AddDate(0, 0, 1-weekday)
		end = begin.AddDate(0, 0, 7).Add(-time.Second)
		return begin, end, true
	case "$lastWeek":
		weekday := int(now.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		thisBegin := dayStart.AddDate(0, 0, 1-weekday)
		begin = thisBegin.AddDate(0, 0, -7)
		end = thisBegin.Add(-time.Second)
		return begin, end, true
	case "$thisMonth":
		begin = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		end = begin.AddDate(0, 1, 0).Add(-time.Second)
		return begin, end, true
	case "$lastMonth":
		thisBegin := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		begin = thisBegin.AddDate(0, -1, 0)
		end = thisBegin.Add(-time.Second)
		return begin, end, true
	default:
		return time.Time{}, time.Time{}, false
	}
}

// BuildCondClause 生成参数化条件片段。moduleChildIDs 仅 module 的 belong/include/notinclude 使用。
func BuildCondClause(cond SearchCond, control string, moduleChildIDs ...uint) (sql string, args []any, ok bool) {
	field := strings.TrimSpace(cond.Field)
	value := strings.TrimSpace(cond.Value)
	if value == "ZERO" {
		value = "0"
	}
	if !IsAllowedSearchField(field) || value == "" {
		return "", nil, false
	}
	op := NormalizeOperator(cond.Operator)
	col, _ := allowedSearchFields[field]
	control = strings.TrimSpace(control)

	if field == "result" {
		return buildResultClause(op, value)
	}

	if begin, end, dynOK := ResolveDynamicDate(value, time.Now()); dynOK {
		return col + " >= ? AND " + col + " <= ?", []any{begin, end}, true
	}

	if isDateControl(control, field) && dateOnlyRe.MatchString(value) {
		return buildDateLiteralClause(col, op, value)
	}

	if field == "module" && (op == "belong" || op == "include" || op == "notinclude") {
		ids := moduleChildIDs
		if len(ids) == 0 {
			return "", nil, false
		}
		holders := make([]string, len(ids))
		args = make([]any, len(ids))
		for i, id := range ids {
			holders[i] = "?"
			args[i] = id
		}
		inSQL := col + " IN (" + strings.Join(holders, ",") + ")"
		if op == "notinclude" {
			return col + " NOT IN (" + strings.Join(holders, ",") + ")", args, true
		}
		return inSQL, args, true
	}

	if field == "plan" && (op == "include" || op == "=") {
		return "CONCAT(',', " + col + ", ',') LIKE ?", []any{"%," + value + ",%"}, true
	}
	if field == "plan" && (op == "notinclude" || op == "!=") {
		return "CONCAT(',', " + col + ", ',') NOT LIKE ?", []any{"%," + value + ",%"}, true
	}

	// mailto / reviewedBy 等逗号分隔字段：按完整 token 匹配
	if (field == "mailto" || field == "reviewedBy") && (op == "include" || op == "=") {
		return "CONCAT(',', " + col + ", ',') LIKE ?", []any{"%," + value + ",%"}, true
	}
	if (field == "mailto" || field == "reviewedBy") && (op == "notinclude" || op == "!=") {
		return "CONCAT(',', " + col + ", ',') NOT LIKE ?", []any{"%," + value + ",%"}, true
	}

	if (op == "include" || op == "notinclude") && control == "select" {
		like := "CONCAT(',', " + col + ", ',') LIKE ?"
		arg := "%," + value + ",%"
		if op == "notinclude" {
			return "CONCAT(',', " + col + ", ',') NOT LIKE ?", []any{arg}, true
		}
		return like, []any{arg}, true
	}

	switch op {
	case "include":
		return col + " LIKE ?", []any{"%" + value + "%"}, true
	case "notinclude":
		return col + " NOT LIKE ?", []any{"%" + value + "%"}, true
	case "belong":
		return col + " = ?", []any{value}, true
	case "between":
		// 单值 between 对齐禅道：回退 =
		return col + " = ?", []any{value}, true
	case "=", "!=", ">", ">=", "<", "<=":
		return col + " " + op + " ?", []any{value}, true
	default:
		return col + " = ?", []any{value}, true
	}
}

func isDateControl(control, field string) bool {
	if control == "date" {
		return true
	}
	switch field {
	case "openedDate", "reviewedDate", "assignedDate", "closedDate", "lastEditedDate", "activatedDate", "estimateLaunch", "deliverDate":
		return true
	default:
		return false
	}
}

func buildDateLiteralClause(col, op, value string) (string, []any, bool) {
	begin := value + " 00:00:00"
	end := value + " 23:59:59"
	switch op {
	case "=":
		return col + " >= ? AND " + col + " <= ?", []any{begin, end}, true
	case "!=":
		return "(" + col + " < ? OR " + col + " > ?)", []any{begin, end}, true
	case "<=":
		return col + " <= ?", []any{end}, true
	case ">":
		return col + " > ?", []any{end}, true
	case ">=":
		return col + " >= ?", []any{begin}, true
	case "<":
		return col + " < ?", []any{begin}, true
	default:
		return col + " >= ? AND " + col + " <= ?", []any{begin, end}, true
	}
}

func buildResultClause(op, value string) (string, []any, bool) {
	sub := "EXISTS (SELECT 1 FROM zt_storyreview sr WHERE sr.story = s.id AND sr.version = s.version AND sr.result %s ?)"
	switch op {
	case "!=":
		return fmt.Sprintf(sub, "!="), []any{value}, true
	case "include":
		return "EXISTS (SELECT 1 FROM zt_storyreview sr WHERE sr.story = s.id AND sr.version = s.version AND sr.result LIKE ?)", []any{"%" + value + "%"}, true
	default:
		return fmt.Sprintf(sub, "="), []any{value}, true
	}
}

// CombineCondSQL 用 and/or 连接两组条件；任一组空则返回另一组。
func CombineCondSQL(sql1 string, args1 []any, andOr string, sql2 string, args2 []any) (string, []any) {
	andOr = NormalizeAndOr(andOr)
	switch {
	case sql1 == "" && sql2 == "":
		return "", nil
	case sql1 == "":
		return sql2, args2
	case sql2 == "":
		return sql1, args1
	default:
		joiner := " AND "
		if andOr == "or" {
			joiner = " OR "
		}
		outArgs := append(append([]any{}, args1...), args2...)
		return "(" + sql1 + ")" + joiner + "(" + sql2 + ")", outArgs
	}
}
