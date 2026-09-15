package schedule

import (
	"context"
	"errors"
	"strconv"
	"strings"
)

func formatOptionalUint(value uint) string {
	if value == 0 {
		return ""
	}
	return strconv.FormatUint(uint64(value), 10)
}

// UpdateLatestStorySpec 更新研发需求当前版本描述；若历史数据缺失 spec 行则补一行 v1。
func (r *Repo) UpdateLatestStorySpec(ctx context.Context, spec *ZtStorySpec) error {
	if spec == nil {
		return errors.New("story spec is nil")
	}
	if spec.Story == 0 {
		return errors.New("story id is invalid")
	}
	var latest struct {
		Version int `gorm:"column:version"`
	}
	const versionQuery = `SELECT COALESCE(MAX(version), 0) AS version FROM zt_storyspec WHERE story = ?`
	if err := r.db.WithContext(ctx).Raw(versionQuery, spec.Story).Scan(&latest).Error; err != nil {
		return err
	}
	if latest.Version == 0 {
		return r.CreateStorySpec(ctx, &ZtStorySpec{
			Story:   spec.Story,
			Version: 1,
			Title:   strings.TrimSpace(spec.Title),
			Spec:    spec.Spec,
		})
	}
	return r.db.WithContext(ctx).
		Table("zt_storyspec").
		Where("story = ? AND version = ?", spec.Story, latest.Version).
		Updates(map[string]interface{}{
			"title": strings.TrimSpace(spec.Title),
			"spec":  spec.Spec,
		}).Error
}
