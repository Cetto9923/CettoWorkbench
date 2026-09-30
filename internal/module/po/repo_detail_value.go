// =============================================================================
// 文件: internal/module/po/repo_detail_value.go
// 模块: PO 工作台
// 类型: repository
// 职责: 价值模型系统配置（zt_config）与需求价值系数、子需求查询。
// 依赖: gorm.io/gorm
// =============================================================================

package po

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"
)

type configKV struct {
	Key   string `gorm:"column:key"`
	Value string `gorm:"column:value"`
}

// LoadDemandValueConfig 读取价值模型总开关及相关参数。
func (r *DemandDetailRepo) LoadDemandValueConfig(ctx context.Context) (*DemandValueConfig, error) {
	if r == nil || r.db == nil {
		return &DemandValueConfig{
			IntervalMethod: "holdout",
			NoAiCategories: []string{"datachange", "dataexport"},
		}, nil
	}

	var configs []configKV
	err := r.db.WithContext(ctx).Table("zt_config").
		Select("`key`, `value`").
		Where("owner = ? AND module = ? AND section = ?", "system", "demand", "demandvalue").
		Find(&configs).Error
	if err != nil {
		return nil, fmt.Errorf("load demandvalue configs: %w", err)
	}

	cfg := &DemandValueConfig{
		IntervalMethod: "holdout", // 禅道默认 holdout
		NoAiCategories: r.loadNoAiCategories(ctx),
	}
	for _, item := range configs {
		applyDemandValueItem(cfg, item.Key, strings.TrimSpace(item.Value))
	}

	return cfg, nil
}

func applyDemandValueItem(cfg *DemandValueConfig, key, val string) {
	switch key {
	case "enabled":
		cfg.Enabled = val == "1"
	case "costPerMonth":
		cfg.CostPerMonth = parseNonNegativeFloat(val)
	case "intervalMethod":
		if val == "fixed" {
			cfg.IntervalMethod = "fixed"
		} else {
			cfg.IntervalMethod = "holdout"
		}
	case "intervalFixedPercent":
		cfg.IntervalFixedPercent = parsePercentFloat(val)
	}
}

func parseNonNegativeFloat(val string) *float64 {
	if val == "" {
		return nil
	}
	f, err := strconv.ParseFloat(val, 64)
	if err != nil || f < 0 {
		return nil
	}
	rounded := math.Round(f*100) / 100
	return &rounded
}

func parsePercentFloat(val string) *float64 {
	if val == "" {
		return nil
	}
	f, err := strconv.ParseFloat(val, 64)
	if err != nil || f <= 0 || f >= 100 {
		return nil
	}
	rounded := math.Round(f*100) / 100
	return &rounded
}

func (r *DemandDetailRepo) loadNoAiCategories(ctx context.Context) []string {
	var noAiRaw string
	err := r.db.WithContext(ctx).Table("zt_config").
		Select("`value`").
		Where("owner = ? AND module = ? AND section = ? AND `key` = ?", "system", "custom", "clarifyCategoryAIConfig", "noAiCategory").
		Scan(&noAiRaw).Error
	if err == nil && strings.TrimSpace(noAiRaw) != "" {
		var list []string
		if jsonErr := json.Unmarshal([]byte(noAiRaw), &list); jsonErr == nil && len(list) > 0 {
			return list
		}
	}
	return []string{"datachange", "dataexport"}
}

// HasDemandValueColumn 检查 zt_demand 是否已存在 demandValue 列。
func (r *DemandDetailRepo) HasDemandValueColumn(ctx context.Context) (bool, error) {
	if r == nil || r.db == nil {
		return false, nil
	}
	var count int64
	err := r.db.WithContext(ctx).Raw(
		"SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'zt_demand' AND COLUMN_NAME = 'demandValue'",
	).Scan(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// FindDemandValueByID 查询单个需求的价值系数与估算时间。
func (r *DemandDetailRepo) FindDemandValueByID(ctx context.Context, demandID uint) (*float64, string, error) {
	if r == nil || r.db == nil || demandID == 0 {
		return nil, "", nil
	}
	hasCol, err := r.HasDemandValueColumn(ctx)
	if err != nil || !hasCol {
		return nil, "", err
	}

	var row struct {
		DemandValue     *float64   `gorm:"column:demandValue"`
		DemandValueDate *time.Time `gorm:"column:demandValueDate"`
	}
	err = r.db.WithContext(ctx).Table("zt_demand").
		Select("demandValue, demandValueDate").
		Where("id = ? AND deleted = '0'", demandID).
		Take(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, "", nil
		}
		return nil, "", err
	}
	dateStr := ""
	if row.DemandValueDate != nil && !row.DemandValueDate.IsZero() {
		dateStr = row.DemandValueDate.Format("2006-01-02 15:04")
	}
	return row.DemandValue, dateStr, nil
}

// FindChildDemandsForValue 查询父需求下的全部未删除子需求。
func (r *DemandDetailRepo) FindChildDemandsForValue(ctx context.Context, parentID uint) ([]ChildDemandValueRow, error) {
	if r == nil || r.db == nil || parentID == 0 {
		return nil, nil
	}
	hasCol, err := r.HasDemandValueColumn(ctx)
	if err != nil {
		return nil, err
	}

	if !hasCol {
		var rows []struct {
			ID       uint   `gorm:"column:id"`
			Category string `gorm:"column:category"`
		}
		err = r.db.WithContext(ctx).Table("zt_demand").
			Select("id, category").
			Where("parent = ? AND deleted = '0'", parentID).
			Find(&rows).Error
		if err != nil {
			return nil, err
		}
		out := make([]ChildDemandValueRow, len(rows))
		for i, row := range rows {
			out[i] = ChildDemandValueRow{
				ID:       row.ID,
				Category: row.Category,
			}
		}
		return out, nil
	}

	var rows []struct {
		ID          uint     `gorm:"column:id"`
		Category    string   `gorm:"column:category"`
		DemandValue *float64 `gorm:"column:demandValue"`
	}
	err = r.db.WithContext(ctx).Table("zt_demand").
		Select("id, category, demandValue").
		Where("parent = ? AND deleted = '0'", parentID).
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]ChildDemandValueRow, len(rows))
	for i, row := range rows {
		out[i] = ChildDemandValueRow{
			ID:          row.ID,
			Category:    row.Category,
			DemandValue: row.DemandValue,
		}
	}
	return out, nil
}
