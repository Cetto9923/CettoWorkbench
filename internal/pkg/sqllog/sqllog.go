// =============================================================================
// 文件: internal/pkg/sqllog/sqllog.go
// 模块: 基础设施
// 类型: infra
// 职责: 以 JSON 行格式记录 SQL 查询日志（sql.log），支持按 request_id 聚合。
// 依赖: internal/config
// =============================================================================

package sqllog

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	gomysql "github.com/go-sql-driver/mysql"
	"gorm.io/gorm"

	"workbench/internal/config"
)

const slowThreshold = 200 * time.Millisecond

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

// LogQuery 记录单条 SQL 查询。
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

	entry := queryEntry{
		Time:      formatTime(time.Now()),
		RequestID: requestID,
		Seq:       seq,
		SQL:       SanitizeSQL(sql),
		Elapsed:   formatDuration(elapsed),
		Rows:      rows,
		File:      callerLocation(),
	}
	if err != nil {
		entry.Error = SafeErrorCategory(err)
	}

	defaultWriter.write(entry)
}

// LogRequestSummary 记录请求级 SQL 汇总。
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
	mu      sync.Mutex
	enabled bool
	file    *os.File
}

func (w *Writer) open(cfg *config.Config) error {
	dir := strings.TrimSpace(cfg.Log.Dir)
	if dir == "" {
		w.enabled = false
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
	w.file = file
	w.enabled = true
	return nil
}

func (w *Writer) sync() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.file == nil {
		return nil
	}
	return w.file.Sync()
}

func (w *Writer) write(v any) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if !w.enabled || w.file == nil {
		return
	}

	data, err := json.Marshal(v)
	if err != nil {
		return
	}
	_, _ = w.file.Write(data)
	_, _ = w.file.Write([]byte("\n"))
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

// SanitizeSQL preserves parameterized templates after GORM removes bind values.
// Inline literals/comments or ambiguous escaping use only a stable fingerprint;
// do not attempt to reconstruct a safe template from interpolated SQL.
func SanitizeSQL(raw string) string {
	if strings.ContainsAny(raw, "\"'\\#0123456789") || strings.Contains(raw, "--") || strings.Contains(raw, "/*") {
		sum := sha256.Sum256([]byte(raw))
		return fmt.Sprintf("SQL fingerprint sha256:%x", sum)
	}
	return strings.Join(strings.Fields(raw), " ")
}

// SafeErrorCategory 返回数据库错误的安全分类摘要，坚决不包含任何原始驱动参数或输入内容。
func SafeErrorCategory(err error) string {
	if err == nil {
		return ""
	}

	var mysqlErr *gomysql.MySQLError
	if errors.As(err, &mysqlErr) {
		return fmt.Sprintf("mysql error %d: %s", mysqlErr.Number, mysqlErrCodeName(mysqlErr.Number))
	}

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return "record not found"
	}
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return "duplicate key"
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return "deadline exceeded"
	}
	if errors.Is(err, context.Canceled) {
		return "context canceled"
	}

	errStr := strings.ToLower(err.Error())
	switch {
	case strings.Contains(errStr, "duplicate") || strings.Contains(errStr, "1062"):
		return "mysql error 1062: duplicate entry"
	case strings.Contains(errStr, "deadlock") || strings.Contains(errStr, "1213"):
		return "mysql error 1213: deadlock"
	case strings.Contains(errStr, "timeout") || strings.Contains(errStr, "1205"):
		return "mysql error 1205: lock wait timeout"
	case strings.Contains(errStr, "foreign key") || strings.Contains(errStr, "1451") || strings.Contains(errStr, "1452"):
		return "foreign key constraint violation"
	case strings.Contains(errStr, "record not found"):
		return "record not found"
	case strings.Contains(errStr, "syntax"):
		return "sql syntax error"
	case strings.Contains(errStr, "connection refused") || strings.Contains(errStr, "connection reset"):
		return "database connection error"
	default:
		return "database query error"
	}
}

func mysqlErrCodeName(code uint16) string {
	switch code {
	case 1062:
		return "duplicate entry"
	case 1048:
		return "column cannot be null"
	case 1213:
		return "deadlock"
	case 1205:
		return "lock wait timeout"
	case 1451, 1452:
		return "foreign key constraint violation"
	case 1054:
		return "unknown column"
	case 1146:
		return "table does not exist"
	case 1064:
		return "syntax error"
	default:
		return "query failure"
	}
}
