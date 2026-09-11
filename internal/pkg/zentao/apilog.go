// =============================================================================
// 文件: internal/pkg/zentao/apilog.go
// 模块: 基础设施
// 类型: infra
// 职责: 禅道 REST 请求日志，按日写入 api-YYYY-MM-DD.log。
// 依赖: internal/config
// =============================================================================

package zentao

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"workbench/internal/config"
)

const (
	apiLogFilePrefix = "api-"
	apiLogFileSuffix = ".log"
)

var defaultAPILog = &apiLogWriter{}

// InitAPILog 初始化禅道 API 请求日志；log.dir 为空时不写文件。
func InitAPILog(cfg *config.Config) error {
	return defaultAPILog.open(cfg)
}

// SyncAPILog 刷盘 API 日志。
func SyncAPILog() error {
	return defaultAPILog.sync()
}

type apiLogEntry struct {
	Time     string `json:"time"`               // 请求开始时间
	Elapsed  string `json:"elapsed"`            // 请求时长
	Method   string `json:"method"`             // HTTP 方法
	Path     string `json:"path"`               // 相对路径，如 /projects/1/builds
	URL      string `json:"url"`                // 完整 URL
	Status   int    `json:"status"`             // HTTP 状态码（未发出则为 0）
	Success  bool   `json:"success"`            // 是否 2xx
	Request  any    `json:"request,omitempty"`  // 请求参数
	Response any    `json:"response,omitempty"` // 返回参数
	Error    string `json:"error,omitempty"`    // 错误信息
}

type apiLogWriter struct {
	mu      sync.Mutex
	enabled bool
	dir     string
	day     string
	file    *os.File
	now     func() time.Time
}

func (w *apiLogWriter) open(cfg *config.Config) error {
	if cfg == nil {
		w.enabled = false
		return nil
	}
	dir := strings.TrimSpace(cfg.Log.Dir)
	if dir == "" {
		w.mu.Lock()
		w.enabled = false
		if w.file != nil {
			_ = w.file.Close()
			w.file = nil
		}
		w.mu.Unlock()
		return nil
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create api log dir %q: %w", dir, err)
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.file != nil {
		_ = w.file.Close()
		w.file = nil
	}
	w.dir = dir
	w.day = ""
	w.enabled = true
	if w.now == nil {
		w.now = time.Now
	}
	return nil
}

func (w *apiLogWriter) sync() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.file == nil {
		return nil
	}
	return w.file.Sync()
}

func (w *apiLogWriter) write(entry apiLogEntry) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if !w.enabled || w.dir == "" {
		return
	}
	if err := w.ensureFileLocked(); err != nil {
		return
	}
	data, err := json.Marshal(entry)
	if err != nil {
		return
	}
	_, _ = w.file.Write(data)
	_, _ = w.file.Write([]byte("\n"))
}

func (w *apiLogWriter) ensureFileLocked() error {
	nowFn := w.now
	if nowFn == nil {
		nowFn = time.Now
	}
	day := nowFn().Format("2006-01-02")
	if w.file != nil && w.day == day {
		return nil
	}
	if w.file != nil {
		_ = w.file.Close()
		w.file = nil
	}
	name := apiLogFilePrefix + day + apiLogFileSuffix
	path := filepath.Join(w.dir, name)
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	w.file = f
	w.day = day
	return nil
}

func logAPICall(entry apiLogEntry) {
	defaultAPILog.write(entry)
}

func formatAPILogTime(t time.Time) string {
	return t.Format("2006-01-02 15:04:05.000")
}

func formatAPILogDuration(d time.Duration) string {
	if d < time.Millisecond {
		return fmt.Sprintf("%.2fµs", float64(d.Microseconds()))
	}
	if d < time.Second {
		return fmt.Sprintf("%.2fms", float64(d.Microseconds())/1000)
	}
	return fmt.Sprintf("%.2fs", d.Seconds())
}

// encodeRequestBody 将请求体转为可落盘结构；password 字段脱敏。
func encodeRequestBody(body any) any {
	if body == nil {
		return nil
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return fmt.Sprintf("%v", body)
	}
	var v any
	if err := json.Unmarshal(raw, &v); err == nil {
		return redactSecrets(v)
	}
	return string(raw)
}

func redactSecrets(v any) any {
	switch t := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(t))
		for k, val := range t {
			if strings.EqualFold(k, "password") {
				out[k] = "***"
				continue
			}
			out[k] = redactSecrets(val)
		}
		return out
	case []any:
		out := make([]any, len(t))
		for i, item := range t {
			out[i] = redactSecrets(item)
		}
		return out
	default:
		return v
	}
}

// encodeResponseBody 解析响应原文（完整保留）。
func encodeResponseBody(raw []byte) any {
	if len(raw) == 0 {
		return nil
	}
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return string(raw)
	}
	return v
}
