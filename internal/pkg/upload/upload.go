// =============================================================================
// 文件: internal/pkg/upload/upload.go
// 模块: 基础设施
// 类型: infra
// 职责: 提供通用文件上传存储接口与本地存储实现。
// 依赖: 无
// =============================================================================

package upload

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const sniffLen = 512

// Storage 定义上传存储接口。
type Storage interface {
	Save(ctx context.Context, file multipart.File, header *multipart.FileHeader) (url string, err error)
}

// LocalStorage 本地文件存储实现。
type LocalStorage struct {
	maxSizeBytes int64
	allowedTypes map[string]bool
	localDir     string
}

// NewLocalStorage 创建本地存储实现。
func NewLocalStorage(maxSizeMB int, allowedTypes []string, localDir string) *LocalStorage {
	if maxSizeMB <= 0 {
		maxSizeMB = 10
	}
	dir := strings.TrimSpace(localDir)
	if dir == "" {
		dir = "uploads"
	}
	allowed := make(map[string]bool)
	for _, t := range allowedTypes {
		key := strings.TrimSpace(strings.ToLower(t))
		if key != "" {
			allowed[key] = true
		}
	}
	if len(allowed) == 0 {
		allowed["image/jpeg"] = true
		allowed["image/png"] = true
		allowed["application/pdf"] = true
	}
	return &LocalStorage{
		maxSizeBytes: int64(maxSizeMB) * 1024 * 1024,
		allowedTypes: allowed,
		localDir:     dir,
	}
}

// Save 保存上传文件并返回访问 URL。
func (s *LocalStorage) Save(ctx context.Context, file multipart.File, header *multipart.FileHeader) (string, error) {
	if s == nil {
		return "", errors.New("storage is nil")
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if file == nil || header == nil {
		return "", errors.New("file is nil")
	}

	if header.Size > 0 && header.Size > s.maxSizeBytes {
		return "", fmt.Errorf("file too large: max %d bytes", s.maxSizeBytes)
	}

	head := make([]byte, sniffLen)
	n, err := io.ReadFull(file, head)
	if err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, io.ErrUnexpectedEOF) {
		return "", err
	}
	contentType := strings.ToLower(strings.TrimSpace(http.DetectContentType(head[:n])))
	if !s.allowedTypes[contentType] {
		return "", fmt.Errorf("unsupported content type: %s", contentType)
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return "", err
	}

	now := time.Now()
	subdir := filepath.Join(fmt.Sprintf("%04d", now.Year()), fmt.Sprintf("%02d", int(now.Month())))
	targetDir := filepath.Join(s.localDir, subdir)
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return "", err
	}

	filename := newUUID() + Ext(header.Filename)
	dstPath := filepath.Join(targetDir, filename)

	if strings.HasPrefix(contentType, "image/") {
		if err := s.saveReencodedImage(file, dstPath, contentType); err != nil {
			return "", err
		}
	} else {
		if err := s.saveRaw(file, dstPath); err != nil {
			return "", err
		}
	}

	if err := ensureFileSizeWithin(dstPath, s.maxSizeBytes); err != nil {
		_ = os.Remove(dstPath)
		return "", err
	}

	rel := filepath.ToSlash(filepath.Join(s.localDir, subdir, filename))
	return "/" + strings.TrimPrefix(rel, "/"), nil
}

func (s *LocalStorage) saveRaw(file multipart.File, dstPath string) error {
	dst, err := os.OpenFile(dstPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	defer dst.Close()

	limited := &io.LimitedReader{R: file, N: s.maxSizeBytes + 1}
	written, err := io.Copy(dst, limited)
	if err != nil {
		return err
	}
	if written > s.maxSizeBytes {
		return fmt.Errorf("file too large: max %d bytes", s.maxSizeBytes)
	}
	return nil
}

func (s *LocalStorage) saveReencodedImage(file multipart.File, dstPath, contentType string) error {
	img, _, err := image.Decode(file)
	if err != nil {
		return fmt.Errorf("decode image: %w", err)
	}
	dst, err := os.OpenFile(dstPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	defer dst.Close()

	switch contentType {
	case "image/jpeg":
		if err := jpeg.Encode(dst, img, &jpeg.Options{Quality: 90}); err != nil {
			return fmt.Errorf("encode jpeg: %w", err)
		}
	case "image/png":
		encoder := png.Encoder{CompressionLevel: png.DefaultCompression}
		if err := encoder.Encode(dst, img); err != nil {
			return fmt.Errorf("encode png: %w", err)
		}
	default:
		return fmt.Errorf("unsupported image type: %s", contentType)
	}
	return nil
}

func ensureFileSizeWithin(path string, max int64) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if info.Size() > max {
		return fmt.Errorf("file too large: max %d bytes", max)
	}
	return nil
}

func newUUID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	buf := make([]byte, 36)
	hex.Encode(buf[0:8], b[0:4])
	buf[8] = '-'
	hex.Encode(buf[9:13], b[4:6])
	buf[13] = '-'
	hex.Encode(buf[14:18], b[6:8])
	buf[18] = '-'
	hex.Encode(buf[19:23], b[8:10])
	buf[23] = '-'
	hex.Encode(buf[24:36], b[10:16])
	return string(buf)
}
