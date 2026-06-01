// =============================================================================
// 文件: internal/pkg/sqllog/reader.go
// 模块: 基础设施
// 类型: infra
// 职责: 从 sql.log 读取请求级 SQL 汇总行，供性能分析页使用。
// 依赖: 无
// =============================================================================

package sqllog

import (
	"bufio"
	"encoding/json"
	"os"
	"strconv"
	"strings"
)

// RequestSummary 请求级 SQL 汇总（与 sql.log 汇总行字段对应）。
type RequestSummary struct {
	Time      string  `json:"time"`
	Level     string  `json:"level"`
	RequestID string  `json:"request_id"`
	Method    string  `json:"method"`
	Route     string  `json:"route"`
	Elapsed   string  `json:"elapsed"`
	ElapsedMS float64 `json:"elapsed_ms"`
	SQLCount  int     `json:"sql_count"`
}

// ReadRequestSummaries 读取日志文件中全部请求汇总行；文件不存在时返回空切片。
func ReadRequestSummaries(path string) ([]RequestSummary, error) {
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []RequestSummary{}, nil
		}
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	summaries := make([]RequestSummary, 0)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var entry requestSummaryEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue
		}
		if strings.TrimSpace(entry.Route) == "" {
			continue
		}

		summaries = append(summaries, RequestSummary{
			Time:      entry.Time,
			Level:     entry.Level,
			RequestID: entry.RequestID,
			Method:    entry.Method,
			Route:     entry.Route,
			Elapsed:   entry.Elapsed,
			ElapsedMS: parseElapsedMS(entry.Elapsed),
			SQLCount:  entry.SQLCount,
		})
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return summaries, nil
}

func parseElapsedMS(text string) float64 {
	text = strings.TrimSpace(text)
	if text == "" {
		return 0
	}

	switch {
	case strings.HasSuffix(text, "µs"):
		value, err := strconv.ParseFloat(strings.TrimSuffix(text, "µs"), 64)
		if err != nil {
			return 0
		}
		return value / 1000
	case strings.HasSuffix(text, "ms"):
		value, err := strconv.ParseFloat(strings.TrimSuffix(text, "ms"), 64)
		if err != nil {
			return 0
		}
		return value
	case strings.HasSuffix(text, "s"):
		value, err := strconv.ParseFloat(strings.TrimSuffix(text, "s"), 64)
		if err != nil {
			return 0
		}
		return value * 1000
	default:
		return 0
	}
}
