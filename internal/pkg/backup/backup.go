// =============================================================================
// 文件: internal/pkg/backup/backup.go
// 模块: 工具组件
// 类型: infra
// 职责: 封装 mysqldump 执行、gzip 压缩与备份记录写入。
// 依赖: internal/model
// =============================================================================

package backup

import (
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"

	"workbrench/internal/model"
)

// Config 备份执行配置。
type Config struct {
	DB       *gorm.DB
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
	Dir      string
	KeepDays int
}

// Run 执行 mysqldump，压缩输出并写入备份记录。
func Run(cfg Config) (filename string, err error) {
	dir := strings.TrimSpace(cfg.Dir)
	if dir == "" {
		dir = "backups"
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("create backup dir failed: %w", err)
	}

	now := time.Now()
	name := now.Format("2006-01-02_15-04-05") + ".sql.gz"
	fullpath := filepath.Join(dir, name)
	file, err := os.OpenFile(fullpath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return "", fmt.Errorf("open backup file failed: %w", err)
	}
	defer func() { _ = file.Close() }()

	args := []string{
		"-h", strings.TrimSpace(cfg.Host),
		"-P", strconv.Itoa(cfg.Port),
		"-u", strings.TrimSpace(cfg.User),
		"--single-transaction",
		"--routines",
		"--events",
		"--triggers",
		strings.TrimSpace(cfg.DBName),
	}
	cmd := exec.Command("mysqldump", args...)
	if strings.TrimSpace(cfg.Password) != "" {
		cmd.Env = append(os.Environ(), "MYSQL_PWD="+cfg.Password)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", fmt.Errorf("create mysqldump stdout pipe failed: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("start mysqldump failed: %w", err)
	}

	gz := gzip.NewWriter(file)
	_, copyErr := io.Copy(gz, stdout)
	closeErr := gz.Close()
	waitErr := cmd.Wait()
	if copyErr != nil {
		_ = os.Remove(fullpath)
		return "", fmt.Errorf("compress dump stream failed: %w", copyErr)
	}
	if closeErr != nil {
		_ = os.Remove(fullpath)
		return "", fmt.Errorf("close gzip writer failed: %w", closeErr)
	}
	if waitErr != nil {
		_ = os.Remove(fullpath)
		return "", fmt.Errorf("mysqldump execution failed: %w", waitErr)
	}

	stat, err := file.Stat()
	if err != nil {
		_ = os.Remove(fullpath)
		return "", fmt.Errorf("stat backup file failed: %w", err)
	}
	sizeByte := stat.Size()
	if cfg.DB != nil {
		row := model.BackupRecord{
			Filename:  name,
			SizeByte:  sizeByte,
			CreatedBy: 0,
			UpdatedBy: 0,
			Deleted:   false,
		}
		if err := cfg.DB.Create(&row).Error; err != nil {
			_ = os.Remove(fullpath)
			return "", fmt.Errorf("write backup record failed: %w", err)
		}
	}
	cleanupExpired(cfg)
	return name, nil
}

func cleanupExpired(cfg Config) {
	if cfg.KeepDays <= 0 {
		return
	}
	dir := strings.TrimSpace(cfg.Dir)
	if dir == "" {
		dir = "backups"
	}
	cutoff := time.Now().AddDate(0, 0, -cfg.KeepDays)
	pattern := filepath.Join(dir, "*.sql.gz")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return
	}
	for _, f := range matches {
		info, statErr := os.Stat(f)
		if statErr != nil {
			continue
		}
		if info.ModTime().Before(cutoff) {
			_ = os.Remove(f)
		}
	}
}
