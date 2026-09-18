// =============================================================================
// 文件: internal/pkg/sqllog/sqllog.go
// 模块: 基础设施
// 类型: infra
// 职责: 以 JSON 行格式记录 SQL 查询日志（sql.log + 按日 sql-YYYY-MM-DD.log），dev 环境同时彩色输出到控制台。
// 依赖: internal/config
// =============================================================================

package sqllog

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"workbench/internal/config"
)

const (
	slowThreshold  = 200 * time.Millisecond
	dailyLogPrefix = "sql-"
	dailyLogSuffix = ".log"
)

var defaultWriter = &Writer{}

// Init 初始化 SQL 日志写入器；log.dir 为空时不写文件。
func Init(cfg *config.Config) error {
	return defaultWriter.open(cfg)
}

// Sync 刷盘 SQL 日志。
func Sync() error {
	return defaultWriter.sync()
}

// NewRequestID 生成请求追踪 ID。
func NewRequestID() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}

// LogQuery 记录单条 SQL 查询（写入 sql.log 与按日 sql-YYYY-MM-DD.log）。
func LogQuery(ctx context.Context, sql string, elapsed time.Duration, rows int64, err error) {
	state := RequestStateFromContext(ctx)
	slow := elapsed > slowThreshold

	seq := 0
	requestID := ""
	if state != nil {
		seq = state.nextSeq()
		requestID = state.RequestID
		state.markQuery(err, slow)
	}

	file := callerLocation()
	entry := queryEntry{
		Time:      formatTime(time.Now()),
		RequestID: requestID,
		Seq:       seq,
		SQL:       sql,
		Elapsed:   formatDuration(elapsed),
		Rows:      rows,
		File:      file,
	}
	if err != nil {
		entry.Error = err.Error()
	}

	defaultWriter.write(entry)
	defaultWriter.writeDaily(entry)
	defaultWriter.printConsoleSQL(sql, elapsed, rows, file, err)
}

// LogRequestSummary 记录请求级 SQL 汇总（仅写入 sql.log，供性能分析页读取）。
func LogRequestSummary(state *RequestState, route string, elapsed time.Duration) {
	if state == nil {
		return
	}

	sqlCount, hasError, hasSlow := state.snapshot()
	level := "INFO"
	switch {
	case hasError:
		level = "ERROR"
	case hasSlow:
		level = "WARN"
	}

	entry := requestSummaryEntry{
		Time:      formatTime(time.Now()),
		Level:     level,
		RequestID: state.RequestID,
		Method:    state.Method,
		Route:     route,
		Elapsed:   formatDuration(elapsed),
		SQLCount:  sqlCount,
	}

	defaultWriter.write(entry)
}

type Writer struct {
	mu             sync.Mutex
	enabled        bool
	consoleEnabled bool
	file           *os.File // sql.log（性能分析）
	dir            string
	day            string
	dailyFile      *os.File // sql-YYYY-MM-DD.log（按日查询明细）
	now            func() time.Time
}

func (w *Writer) open(cfg *config.Config) error {
	w.mu.Lock()
	w.consoleEnabled = cfg != nil && !strings.EqualFold(strings.TrimSpace(cfg.App.Env), "prod")
	if w.now == nil {
		w.now = time.Now
	}
	w.mu.Unlock()

	dir := ""
	if cfg != nil {
		dir = strings.TrimSpace(cfg.Log.Dir)
	}
	if dir == "" {
		w.mu.Lock()
		w.enabled = false
		w.dir = ""
		w.day = ""
		if w.file != nil {
			_ = w.file.Close()
			w.file = nil
		}
		if w.dailyFile != nil {
			_ = w.dailyFile.Close()
			w.dailyFile = nil
		}
		w.mu.Unlock()
		return nil
	}

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create log dir %q: %w", dir, err)
	}

	logPath := filepath.Join(dir, "sql.log")
	file, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("open sql log file %q: %w", logPath, err)
	}

	w.mu.Lock()
	defer w.mu.Unlock()
	if w.file != nil {
		_ = w.file.Close()
	}
	if w.dailyFile != nil {
		_ = w.dailyFile.Close()
		w.dailyFile = nil
	}
	w.file = file
	w.dir = dir
	w.day = ""
	w.enabled = true
	return nil
}

func (w *Writer) sync() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	var firstErr error
	if w.file != nil {
		if err := w.file.Sync(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	if w.dailyFile != nil {
		if err := w.dailyFile.Sync(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

func (w *Writer) write(v any) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if !w.enabled || w.dir == "" {
		return
	}
	if err := w.ensureMainFileLocked(); err != nil {
		return
	}

	data, err := json.Marshal(v)
	if err != nil {
		return
	}
	_, _ = w.file.Write(data)
	_, _ = w.file.Write([]byte("\n"))
}

// writeDaily 将单条 SQL 查询写入按日文件 sql-YYYY-MM-DD.log（与 sql.log 分离）。
func (w *Writer) writeDaily(v any) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if !w.enabled || w.dir == "" {
		return
	}
	if err := w.ensureDailyFileLocked(); err != nil {
		return
	}
	data, err := json.Marshal(v)
	if err != nil {
		return
	}
	_, _ = w.dailyFile.Write(data)
	_, _ = w.dailyFile.Write([]byte("\n"))
}

// openFileAlive 判断已打开的文件句柄是否仍指向 path 上的同一 inode。
// 外部删除/替换日志文件后，句柄仍可写但路径上已无文件，此时需重新 OpenFile。
func openFileAlive(f *os.File, path string) bool {
	if f == nil {
		return false
	}
	fi, err := f.Stat()
	if err != nil {
		return false
	}
	pi, err := os.Stat(path)
	if err != nil {
		return false
	}
	return os.SameFile(fi, pi)
}

func (w *Writer) ensureMainFileLocked() error {
	path := filepath.Join(w.dir, "sql.log")
	if openFileAlive(w.file, path) {
		return nil
	}
	if w.file != nil {
		_ = w.file.Close()
		w.file = nil
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	w.file = f
	return nil
}

func (w *Writer) ensureDailyFileLocked() error {
	nowFn := w.now
	if nowFn == nil {
		nowFn = time.Now
	}
	day := nowFn().Format("2006-01-02")
	path := filepath.Join(w.dir, dailyLogPrefix+day+dailyLogSuffix)
	if openFileAlive(w.dailyFile, path) && w.day == day {
		return nil
	}
	if w.dailyFile != nil {
		_ = w.dailyFile.Close()
		w.dailyFile = nil
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	w.dailyFile = f
	w.day = day
	return nil
}

type requestSummaryEntry struct {
	Time      string `json:"time"`
	Level     string `json:"level"`
	RequestID string `json:"request_id"`
	Method    string `json:"method"`
	Route     string `json:"route"`
	Elapsed   string `json:"elapsed"`
	SQLCount  int    `json:"sql_count"`
}

type queryEntry struct {
	Time      string `json:"time"`
	RequestID string `json:"request_id,omitempty"`
	Seq       int    `json:"seq"`
	SQL       string `json:"sql"`
	Elapsed   string `json:"elapsed"`
	Rows      int64  `json:"rows"`
	File      string `json:"file"`
	Error     string `json:"error,omitempty"`
}

func formatTime(t time.Time) string {
	return t.Format("2006-01-02 15:04:05.000")
}

func formatDuration(d time.Duration) string {
	if d < time.Millisecond {
		return fmt.Sprintf("%.2fµs", float64(d.Microseconds()))
	}
	if d < time.Second {
		return fmt.Sprintf("%.2fms", float64(d.Microseconds())/1000)
	}
	return fmt.Sprintf("%.2fs", d.Seconds())
}

func callerLocation() string {
	pcs := make([]uintptr, 24)
	n := runtime.Callers(3, pcs)
	frames := runtime.CallersFrames(pcs[:n])
	for {
		frame, more := frames.Next()
		if isBusinessFrame(frame.File) {
			return relModulePath(frame.File) + fmt.Sprintf(":%d", frame.Line)
		}
		if !more {
			break
		}
	}
	return ""
}

func isBusinessFrame(file string) bool {
	if file == "" {
		return false
	}
	if strings.Contains(file, "gorm.io/") {
		return false
	}
	if strings.Contains(file, "/internal/pkg/database/") {
		return false
	}
	if strings.Contains(file, "/internal/pkg/sqllog/") {
		return false
	}
	if strings.Contains(file, "/runtime/") {
		return false
	}
	return strings.Contains(file, "/internal/")
}

func relModulePath(path string) string {
	const marker = "goframework/"
	if idx := strings.LastIndex(path, marker); idx >= 0 {
		return path[idx+len(marker):]
	}
	return filepath.ToSlash(path)
}
