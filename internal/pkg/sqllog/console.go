// =============================================================================
// 文件: internal/pkg/sqllog/console.go
// 模块: 基础设施
// 类型: infra
// 职责: dev 环境向控制台输出带 ANSI 颜色的 SQL。
// 依赖: 无
// =============================================================================

package sqllog

import (
	"fmt"
	"os"
	"strings"
	"time"
	"unicode"
)

const (
	ansiReset  = "\033[0m"
	ansiCyan   = "\033[36m"
	ansiYellow = "\033[33m"
	ansiRed    = "\033[31m"
	ansiGreen  = "\033[32m"
	ansiDim    = "\033[2m"
	ansiBold   = "\033[1m"
)

// printConsoleSQL 仅在 consoleEnabled 时向 stderr 打印彩色 SQL。
func (w *Writer) printConsoleSQL(sql string, elapsed time.Duration, rows int64, file string, err error) {
	if w == nil {
		return
	}
	w.mu.Lock()
	enabled := w.consoleEnabled
	w.mu.Unlock()
	if !enabled {
		return
	}

	elapsedText := formatDuration(elapsed)
	elapsedColor := ansiGreen
	if elapsed > slowThreshold {
		elapsedColor = ansiYellow
	}

	var b strings.Builder
	b.WriteString(ansiDim)
	b.WriteString("[SQL] ")
	b.WriteString(ansiReset)
	b.WriteString(elapsedColor)
	b.WriteString(elapsedText)
	b.WriteString(ansiReset)
	b.WriteString(ansiDim)
	b.WriteString(fmt.Sprintf(" rows=%d", rows))
	if file != "" {
		b.WriteString(" ")
		b.WriteString(file)
	}
	b.WriteString(ansiReset)
	b.WriteByte('\n')
	b.WriteString(colorizeSQL(sql))
	if err != nil {
		b.WriteByte('\n')
		b.WriteString(ansiRed)
		b.WriteString(ansiBold)
		b.WriteString("error: ")
		b.WriteString(err.Error())
		b.WriteString(ansiReset)
	}
	b.WriteByte('\n')
	_, _ = fmt.Fprint(os.Stderr, b.String())
}

// colorizeSQL 给 SQL 关键字上色（字符串字面量内不着色，字面量用黄色）。
func colorizeSQL(sql string) string {
	if sql == "" {
		return sql
	}
	var b strings.Builder
	b.Grow(len(sql) + 64)
	i := 0
	for i < len(sql) {
		if sql[i] == '\'' {
			end := scanSQLString(sql, i)
			b.WriteString(ansiYellow)
			b.WriteString(sql[i:end])
			b.WriteString(ansiReset)
			i = end
			continue
		}
		if isIdentStart(rune(sql[i])) {
			end := i + 1
			for end < len(sql) && isIdentPart(rune(sql[end])) {
				end++
			}
			word := sql[i:end]
			if isSQLKeyword(word) {
				b.WriteString(ansiCyan)
				b.WriteString(ansiBold)
				b.WriteString(strings.ToUpper(word))
				b.WriteString(ansiReset)
			} else {
				b.WriteString(word)
			}
			i = end
			continue
		}
		b.WriteByte(sql[i])
		i++
	}
	return b.String()
}

func scanSQLString(sql string, start int) int {
	i := start + 1
	for i < len(sql) {
		if sql[i] == '\'' {
			if i+1 < len(sql) && sql[i+1] == '\'' {
				i += 2
				continue
			}
			return i + 1
		}
		i++
	}
	return len(sql)
}

func isIdentStart(r rune) bool {
	return unicode.IsLetter(r) || r == '_'
}

func isIdentPart(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_'
}

func isSQLKeyword(word string) bool {
	switch strings.ToUpper(word) {
	case "SELECT", "DISTINCT", "FROM", "INNER", "LEFT", "RIGHT", "JOIN", "ON",
		"WHERE", "AND", "OR", "NOT", "IN", "IS", "NULL", "AS", "ORDER", "GROUP",
		"BY", "HAVING", "LIMIT", "OFFSET", "INSERT", "INTO", "VALUES", "UPDATE",
		"SET", "DELETE", "UNION", "ALL", "EXISTS", "BETWEEN", "LIKE", "ASC", "DESC",
		"COUNT", "SUM", "AVG", "MAX", "MIN", "CASE", "WHEN", "THEN", "ELSE", "END":
		return true
	default:
		return false
	}
}
