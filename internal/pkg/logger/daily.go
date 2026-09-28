// =============================================================================
// 文件: internal/pkg/logger/daily.go
// 模块: 基础设施
// 类型: infra
// 职责: 按日滚动的日志文件 WriteSyncer（app-YYYY-MM-DD.log）。
// 依赖: 无
// =============================================================================

package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// dailyWriteSyncer 将日志写入 prefix+日期+suffix，跨日自动切换文件。
type dailyWriteSyncer struct {
	mu     sync.Mutex
	dir    string
	prefix string
	suffix string
	day    string
	file   *os.File
	now    func() time.Time
}

func newDailyWriteSyncer(dir, prefix, suffix string) (*dailyWriteSyncer, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create log dir %q: %w", dir, err)
	}
	w := &dailyWriteSyncer{
		dir:    dir,
		prefix: prefix,
		suffix: suffix,
		now:    time.Now,
	}
	if err := w.ensureLocked(); err != nil {
		return nil, err
	}
	return w, nil
}

func (w *dailyWriteSyncer) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if err := w.ensureLocked(); err != nil {
		return 0, err
	}
	return w.file.Write(p)
}

func (w *dailyWriteSyncer) Sync() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.file == nil {
		return nil
	}
	return w.file.Sync()
}

func (w *dailyWriteSyncer) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.file == nil {
		return nil
	}
	err := w.file.Close()
	w.file = nil
	w.day = ""
	return err
}

func (w *dailyWriteSyncer) ensureLocked() error {
	nowFn := w.now
	if nowFn == nil {
		nowFn = time.Now
	}
	day := nowFn().Format("2006-01-02")
	path := filepath.Join(w.dir, w.prefix+day+w.suffix)
	if openFileAlive(w.file, path) && w.day == day {
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
	w.day = day
	return nil
}

// openFileAlive 判断已打开的文件句柄是否仍指向 path 上的同一 inode。
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
