// =============================================================================
// 文件: internal/module/debug/repoapilog.go
// 模块: SQL 性能分析
// 类型: readonly
// 职责: 从 api-YYYY-MM-DD.log 读取禅道 API 请求日志。
// 依赖: 无
// =============================================================================

package debug

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	dailyAPILogPrefix = "api-"
	dailyAPILogSuffix = ".log"
)

// FindAPIEntries 读取指定日期的 API 日志，按时间倒序分页。
// 最多保留最近 maxQueryLimit 条供分页；total 为可分页条数。
func (r *Repo) FindAPIEntries(ctx context.Context, req RepoFindAPIEntriesReq) ([]APILogItem, int64, int, int, error) {
	_ = ctx

	date := strings.TrimSpace(req.Date)
	path := filepath.Join(r.logDir, dailyAPILogPrefix+date+dailyAPILogSuffix)
	page := normalizePage(req.Page)
	pageSize := normalizeQueryLimit(req.PageSize)

	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []APILogItem{}, 0, page, pageSize, nil
		}
		return nil, 0, page, pageSize, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 8*1024*1024)

	// 只保留最近 maxQueryLimit 条（文件按时间追加）
	ring := make([]APILogItem, 0, maxQueryLimit)
	for scanner.Scan() {
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}
		item, ok := parseAPILogLine(line)
		if !ok {
			continue
		}
		if len(ring) < maxQueryLimit {
			ring = append(ring, item)
			continue
		}
		copy(ring[0:], ring[1:])
		ring[len(ring)-1] = item
	}
	if err := scanner.Err(); err != nil {
		return nil, 0, page, pageSize, err
	}
	if len(ring) == 0 {
		return []APILogItem{}, 0, page, pageSize, nil
	}

	// 倒序：最新在前
	all := make([]APILogItem, len(ring))
	for i := range ring {
		all[i] = ring[len(ring)-1-i]
	}
	total := int64(len(all))

	start := (page - 1) * pageSize
	if start >= len(all) {
		return []APILogItem{}, total, page, pageSize, nil
	}
	end := start + pageSize
	if end > len(all) {
		end = len(all)
	}
	return all[start:end], total, page, pageSize, nil
}

func normalizePage(page int) int {
	if page <= 0 {
		return 1
	}
	return page
}

func parseAPILogLine(line []byte) (APILogItem, bool) {
	var raw struct {
		Time     string          `json:"time"`
		Elapsed  string          `json:"elapsed"`
		Method   string          `json:"method"`
		Path     string          `json:"path"`
		URL      string          `json:"url"`
		Status   int             `json:"status"`
		Success  bool            `json:"success"`
		Request  json.RawMessage `json:"request"`
		Response json.RawMessage `json:"response"`
		Error    string          `json:"error"`
	}
	if err := json.Unmarshal(line, &raw); err != nil {
		return APILogItem{}, false
	}
	if strings.TrimSpace(raw.Method) == "" && strings.TrimSpace(raw.Path) == "" {
		return APILogItem{}, false
	}
	return APILogItem{
		Time:      raw.Time,
		Elapsed:   raw.Elapsed,
		ElapsedMS: parseAPIElapsedMS(raw.Elapsed),
		Method:    raw.Method,
		Path:      raw.Path,
		URL:       raw.URL,
		Status:    raw.Status,
		Success:   raw.Success,
		Request:   raw.Request,
		Response:  raw.Response,
		Error:     raw.Error,
	}, true
}

func parseAPIElapsedMS(text string) float64 {
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
