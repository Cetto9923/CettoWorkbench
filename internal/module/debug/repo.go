// =============================================================================
// 文件: internal/module/debug/repo.go
// 模块: SQL 性能分析
// 类型: readonly
// 职责: 从 sql.log / sql-YYYY-MM-DD.log 读取请求汇总与单条 SQL 明细。
// 依赖: internal/pkg/sqllog
// =============================================================================

package debug

import (
	"context"
	"path/filepath"
	"slices"
	"strings"

	"workbench/internal/pkg/sqllog"
)

const (
	dailySQLLogPrefix = "sql-"
	dailySQLLogSuffix = ".log"
	defaultQueryLimit = 200
	maxQueryLimit     = 2000
)

// Repo SQL 性能分析数据访问层。
type Repo struct {
	logDir  string
	logPath string
}

var ignoredRequests = []string{
	"/favicon.ico",
	"/debug/sqlperf",
	"/debug/sqlperf/requests",
	"/debug/sqllog",
	"/debug/sqllog/queries",
}

// NewRepo 创建 Repo。
func NewRepo(logDir string) *Repo {
	dir := strings.TrimSpace(logDir)
	return &Repo{
		logDir:  dir,
		logPath: filepath.Join(dir, "sql.log"),
	}
}

// FindAll 读取全部请求汇总行。
func (r *Repo) FindAll(ctx context.Context, req RepoFindAllReq) ([]RequestItem, int64, error) {
	_ = ctx

	summaries, err := sqllog.ReadRequestSummaries(r.logPath)
	if err != nil {
		return nil, 0, err
	}

	items := make([]RequestItem, 0, len(summaries))
	for _, summary := range summaries {
		if isIgnoredRequest(summary.Method, summary.Route) {
			continue
		}
		if !matchDateRange(summary.Time, req.StartDate, req.EndDate) {
			continue
		}
		items = append(items, toRequestItem(summary))
	}

	return items, int64(len(items)), nil
}

// FindQueries 读取指定日期的 SQL 明细，按耗时降序，并截断到 limit。
func (r *Repo) FindQueries(ctx context.Context, req RepoFindQueriesReq) ([]QueryItem, int64, error) {
	_ = ctx

	date := strings.TrimSpace(req.Date)
	path := filepath.Join(r.logDir, dailySQLLogPrefix+date+dailySQLLogSuffix)
	entries, err := sqllog.ReadQueryEntries(path)
	if err != nil {
		return nil, 0, err
	}

	items := make([]QueryItem, 0, len(entries))
	for _, entry := range entries {
		items = append(items, toQueryItem(entry))
	}

	slices.SortFunc(items, func(a, b QueryItem) int {
		switch {
		case a.ElapsedMS > b.ElapsedMS:
			return -1
		case a.ElapsedMS < b.ElapsedMS:
			return 1
		default:
			return strings.Compare(b.Time, a.Time)
		}
	})

	total := int64(len(items))
	limit := normalizeQueryLimit(req.Limit)
	if len(items) > limit {
		items = items[:limit]
	}
	return items, total, nil
}

func toRequestItem(summary sqllog.RequestSummary) RequestItem {
	return RequestItem{
		Time:      summary.Time,
		Level:     summary.Level,
		RequestID: summary.RequestID,
		Method:    summary.Method,
		Route:     summary.Route,
		Elapsed:   summary.Elapsed,
		ElapsedMS: summary.ElapsedMS,
		SQLCount:  summary.SQLCount,
	}
}

func toQueryItem(entry sqllog.QueryEntry) QueryItem {
	return QueryItem{
		Time:      entry.Time,
		RequestID: entry.RequestID,
		Seq:       entry.Seq,
		SQL:       entry.SQL,
		Elapsed:   entry.Elapsed,
		ElapsedMS: entry.ElapsedMS,
		Rows:      entry.Rows,
		File:      entry.File,
		Error:     entry.Error,
	}
}

func normalizeQueryLimit(limit int) int {
	if limit <= 0 {
		return defaultQueryLimit
	}
	if limit > maxQueryLimit {
		return maxQueryLimit
	}
	return limit
}

func isIgnoredRequest(method, route string) bool {
	return strings.EqualFold(strings.TrimSpace(method), "GET") && slices.Contains(ignoredRequests, route)
}

func matchDateRange(timeText, startDate, endDate string) bool {
	if len(timeText) < 10 {
		return false
	}
	day := timeText[:10]
	start := strings.TrimSpace(startDate)
	end := strings.TrimSpace(endDate)
	if start != "" && day < start {
		return false
	}
	if end != "" && day > end {
		return false
	}
	return true
}
