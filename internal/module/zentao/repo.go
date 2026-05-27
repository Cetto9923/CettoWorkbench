// =============================================================================
// 文件: internal/module/zentao/repo.go
// 模块: 禅道集成
// 类型: action
// 职责: 封装禅道 zt_user 等表的数据访问。
// 依赖: internal/model/zentao
// =============================================================================

package zentao

import (
	"context"

	"gorm.io/gorm"

	zentaomodel "goframework/internal/model/zentao"
)

// Repo 禅道数据访问。
type Repo struct {
	db *gorm.DB
}

// NewRepo 创建 Repo。
func NewRepo(db *gorm.DB) *Repo {
	return &Repo{db: db}
}

// FindAllActiveUsers 查询 zt_user 中未删除的用户。
func (r *Repo) FindAllActiveUsers(ctx context.Context) ([]zentaomodel.ZtUser, error) {
	var rows []zentaomodel.ZtUser
	err := r.db.WithContext(ctx).
		Model(&zentaomodel.ZtUser{}).
		Select("account", "realname").
		Where("deleted = ?", 0).
		Order("account ASC").
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	return rows, nil
}
