// =============================================================================
// 文件: internal/module/sqlperf/repo.go
// 模块: SQL 性能分析
// 类型: readonly
// 职责: 从 sql.log 读取请求级 SQL 汇总数据。
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

// Repo SQL 性能分析数据访问层。
type Repo struct {
	logPath string
}

var ignoredRequests = []string{
	"/favicon.ico",
	"/debug/sqlperf",
	"/debug/sqlperf/requests",
}

// NewRepo 创建 Repo。
func NewRepo(logDir string) *Repo {
	dir := strings.TrimSpace(logDir)
	return &Repo{logPath: filepath.Join(dir, "sql.log")}
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
