// =============================================================================
// 文件: internal/module/build/repoauthz.go
// 模块: 版本管理
// 类型: action
// 职责: 查询所选研发需求的业务来源，拒绝缺失或已删除的研发需求。
// 依赖: internal/model/zentao, internal/pkg/errorx
// =============================================================================
package build

import (
	"context"

	ztmodel "workbench/internal/model/zentao"
	"workbench/internal/pkg/errorx"
)

func (r *Repo) findStoryOrigins(ctx context.Context, ids []string) ([]linkableStoryRow, error) {
	var rows []linkableStoryRow
	err := r.db.WithContext(ctx).Model(&ztmodel.ZtStory{}).
		Select("id, fromDemand").Where("id IN ? AND deleted = ? AND type = ?", ids, "0", "story").
		Order("id ASC").Find(&rows).Error
	if err != nil {
		return nil, err
	}
	if len(rows) != len(ids) {
		return nil, errorx.New(errorx.ErrCodeNotFound, "研发需求不存在或已删除")
	}
	return rows, nil
}
