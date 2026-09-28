// =============================================================================
// 文件: internal/pkg/sqllog/reader.go
// 模块: 基础设施
// 类型: infra
// 职责: 从 sql-YYYY-MM-DD.log 读取请求汇总行与单条 SQL 查询行。
// 依赖: 无
// =============================================================================

package sqllog

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// RequestSummary 请求级 SQL 汇总（与按日日志中汇总行字段对应）。
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

// QueryEntry 单条 SQL 查询记录（与按日 sql-YYYY-MM-DD.log 中查询行对应）。
type QueryEntry struct {
	Time      string  `json:"time"`
	RequestID string  `json:"request_id"`
	Method    string  `json:"method"`
	Route     string  `json:"route"`
	Seq       int     `json:"seq"`
	SQL       string  `json:"sql"`
	Elapsed   string  `json:"elapsed"`
	ElapsedMS float64 `json:"elapsed_ms"`
	Rows      int64   `json:"rows"`
	File      string  `json:"file"`
	Error     string  `json:"error,omitempty"`
}

// DailyLogPaths 返回目录下落在 [startDate, endDate] 内的 sql-YYYY-MM-DD.log 路径（按日期升序）。
// startDate / endDate 为空时不限制对应边界。
func DailyLogPaths(dir, startDate, endDate string) ([]string, error) {
	dir = strings.TrimSpace(dir)
	if dir == "" {
		return nil, nil
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	start := strings.TrimSpace(startDate)
	end := strings.TrimSpace(endDate)
	paths := make([]string, 0)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasPrefix(name, dailyLogPrefix) || !strings.HasSuffix(name, dailyLogSuffix) {
			continue
		}
		day := strings.TrimSuffix(strings.TrimPrefix(name, dailyLogPrefix), dailyLogSuffix)
		if len(day) != 10 {
			continue
		}
		if start != "" && day < start {
			continue
		}
		if end != "" && day > end {
			continue
		}
		paths = append(paths, filepath.Join(dir, name))
	}
	sort.Strings(paths)
	return paths, nil
}

// ReadRequestSummariesInRange 读取日期范围内全部按日文件中的请求汇总行。
func ReadRequestSummariesInRange(dir, startDate, endDate string) ([]RequestSummary, error) {
	paths, err := DailyLogPaths(dir, startDate, endDate)
	if err != nil {
		return nil, err
	}

	summaries := make([]RequestSummary, 0)
	for _, path := range paths {
		daySummaries, err := ReadRequestSummaries(path)
		if err != nil {
			return nil, err
		}
		summaries = append(summaries, daySummaries...)
	}
	return summaries, nil
}

// ReadRequestSummaries 读取日志文件中全部请求汇总行；文件不存在时返回空切片。
// 仅接受带非空 level 的行，避免把同文件中的 SQL 明细误解析为汇总。
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
		line := scanner.Bytes()
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}

		var entry RequestSummary
		if err := json.Unmarshal(line, &entry); err != nil {
			continue
		}
		if strings.TrimSpace(entry.Level) == "" {
			continue
		}
		if strings.TrimSpace(entry.Route) == "" {
			continue
		}

		entry.ElapsedMS = parseElapsedMS(entry.Elapsed)
		summaries = append(summaries, entry)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return summaries, nil
}

// ScanQueryEntries 从 offset 字节处开始扫描单条 SQL 查询行，每解析出一行调用 fn，
// 返回可安全续读的下一个偏移量。写入侧每条日志分两次 Write（先数据、后换行），
// 读取时可能落在两次之间，因此只有以 '\n' 结尾的完整行才计入返回值。
// 文件不存在时原样返回 offset 且不报错。
func ScanQueryEntries(path string, offset int64, fn func(entry QueryEntry)) (int64, error) {
	if offset < 0 {
		offset = 0
	}

	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return offset, nil
		}
		return offset, err
	}

	defer file.Close()
	if offset > 0 {
		//说明有缓存基点
		if _, err := file.Seek(offset, io.SeekStart); err != nil {
			//返回新的偏移量
			return offset, err
		}
	}

	// 缓冲上限对齐改造前的 scanner.Buffer(buf, 1024*1024)：64KB~1MB 的单行仍能整行解析，
	// 超过 1MB 的行退化为跳过（改造前会让整个请求失败）。
	reader := bufio.NewReaderSize(file, 1024*1024)
	pos := offset
	safe := offset

	for {
		line, err := reader.ReadSlice('\n')
		pos += int64(len(line))
		if err == nil {
			safe = pos
			if entry, ok := parseQueryLine(line); ok {
				fn(entry)
			}
			continue
		}
		if errors.Is(err, bufio.ErrBufferFull) {
			// 超长行的中间块：不推进 safe，继续读到该行结尾
			continue
		}
		if err == io.EOF {
			return safe, nil
		}
		return safe, err
	}
}

// parseQueryLine 解析一行日志；空白行、请求汇总行、损坏行都返回 ok=false。
// ReadSlice 返回的切片在下一次读取后失效，本函数只在本次调用内使用它。
func parseQueryLine(line []byte) (QueryEntry, bool) {
	trimmed := bytes.TrimSpace(line)
	if len(trimmed) == 0 {
		return QueryEntry{}, false
	}

	var entry QueryEntry
	if err := json.Unmarshal(trimmed, &entry); err != nil {
		return QueryEntry{}, false
	}
	if strings.TrimSpace(entry.SQL) == "" {
		return QueryEntry{}, false
	}

	entry.ElapsedMS = parseElapsedMS(entry.Elapsed)
	return entry, true
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
