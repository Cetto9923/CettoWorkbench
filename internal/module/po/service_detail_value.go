// =============================================================================
// 文件: internal/module/po/service_detail_value.go
// 模块: PO 工作台
// 类型: service
// 职责: 价值模型（ValueModel）启用判断、系数与费用区间计算。
// 依赖: internal/model, math
// =============================================================================

package po

import (
	"context"
	"math"
	"strings"
)

// MonthWorkDays 月计薪天数，对齐禅道 demandvalueModel::getMonthWorkDays()。
const MonthWorkDays = 21.75

// isValueEstimateSkippedCategory 判断需求类别是否在无AI故事点配置名单中。
func isValueEstimateSkippedCategory(category string, noAiCategories []string) bool {
	cat := strings.TrimSpace(category)
	for _, c := range noAiCategories {
		if strings.TrimSpace(c) == cat {
			return true
		}
	}
	return false
}

// calcCostInterval 按照禅道公式计算费用区间。
// fixed: 系数 × (1 ± 百分比) × 每人月费用 ÷ 21.75。
// holdout: 工作台环境无法读取禅道本地 manifest，标记 costAvailable=false。
func calcCostInterval(
	demandValue float64,
	cfg *DemandValueConfig,
) (*float64, *float64, bool, string) {
	if cfg == nil || cfg.CostPerMonth == nil {
		return nil, nil, false, "未配置每人月费用（后台 › 功能配置 › 需求池 › 每人月费用配置）"
	}
	if cfg.IntervalMethod == "holdout" {
		return nil, nil, false, "区间需在禅道查看"
	}
	if cfg.IntervalMethod == "fixed" {
		if cfg.IntervalFixedPercent == nil {
			return nil, nil, false, "固定比例未配置（后台 › 功能配置 › 需求池 › 需求价值模型配置）"
		}
		ratio := *cfg.IntervalFixedPercent / 100.0
		valLow := math.Round(demandValue*(1.0-ratio)*10000) / 10000
		valHigh := math.Round(demandValue*(1.0+ratio)*10000) / 10000

		costLow := math.Round(valLow*(*cfg.CostPerMonth)/MonthWorkDays*100) / 100
		costHigh := math.Round(valHigh*(*cfg.CostPerMonth)/MonthWorkDays*100) / 100
		return &costLow, &costHigh, true, ""
	}
	return nil, nil, false, "区间需在禅道查看"
}

// populateValueModel 装配详情页价值模型数据块。
// 当后台开关未开启时，返回 nil, nil（整块不出现，与旧版字节级一致）。
func (s *DetailService) populateValueModel(
	ctx context.Context,
	row *DemandDetailRow,
) (*DemandValueModel, error) {
	if s == nil || s.repo == nil || row == nil {
		return nil, nil
	}

	cfg, err := s.repo.LoadDemandValueConfig(ctx)
	if err != nil {
		return nil, err
	}
	if cfg == nil || !cfg.Enabled {
		return nil, nil
	}

	vm := &DemandValueModel{
		Enabled:                true,
		IntervalKind:           cfg.IntervalMethod,
		IntervalPercent:        cfg.IntervalFixedPercent,
		CostPerMonthConfigured: cfg.CostPerMonth != nil,
	}

	if row.Parent == -1 {
		handled, pErr := s.populateParentValueModel(ctx, row.ID, cfg, vm)
		if pErr != nil {
			return nil, pErr
		}
		if handled {
			return vm, nil
		}
	}

	if err := s.populateSingleValueModel(ctx, row, cfg, vm); err != nil {
		return nil, err
	}
	return vm, nil
}

func (s *DetailService) populateParentValueModel(
	ctx context.Context,
	demandID uint,
	cfg *DemandValueConfig,
	vm *DemandValueModel,
) (bool, error) {
	children, err := s.repo.FindChildDemandsForValue(ctx, demandID)
	if err != nil {
		return false, err
	}
	if len(children) == 0 {
		return false, nil
	}

	eligible, estimated, sumVal := aggregateChildValues(children, cfg.NoAiCategories)
	if eligible == 0 {
		return false, nil
	}

	vm.IsParent = true
	vm.ChildEligible = eligible
	vm.ChildEstimated = estimated
	vm.ChildSum = true

	if estimated == 0 {
		vm.CostAvailable = false
		vm.CostUnavailableReason = "子需求暂未估算"
		return true, nil
	}

	roundedSum := math.Round(sumVal*10000) / 10000
	vm.DemandValue = &roundedSum
	low, high, avail, reason := calcCostInterval(roundedSum, cfg)
	vm.CostLow = low
	vm.CostHigh = high
	vm.CostAvailable = avail
	vm.CostUnavailableReason = reason
	return true, nil
}

func aggregateChildValues(children []ChildDemandValueRow, noAiCategories []string) (int, int, float64) {
	eligible := 0
	estimated := 0
	sumVal := 0.0
	for _, child := range children {
		if isValueEstimateSkippedCategory(child.Category, noAiCategories) {
			continue
		}
		eligible++
		if child.DemandValue != nil && *child.DemandValue > 0 {
			estimated++
			sumVal += *child.DemandValue
		}
	}
	return eligible, estimated, sumVal
}

func (s *DetailService) populateSingleValueModel(
	ctx context.Context,
	row *DemandDetailRow,
	cfg *DemandValueConfig,
	vm *DemandValueModel,
) error {
	vm.IsParent = false
	vm.Skipped = isValueEstimateSkippedCategory(row.Category, cfg.NoAiCategories)
	if vm.Skipped {
		vm.CostAvailable = false
		vm.CostUnavailableReason = "该需求类别配置为无AI故事点，跳过需求价值估算。"
		return nil
	}

	val, date, err := s.repo.FindDemandValueByID(ctx, row.ID)
	if err != nil {
		return err
	}
	vm.DemandValueDate = date

	if val == nil || *val <= 0 {
		vm.CostAvailable = false
		vm.CostUnavailableReason = "暂未估算（澄清完成后自动计算）"
		return nil
	}

	roundedVal := math.Round(*val*10000) / 10000
	vm.DemandValue = &roundedVal
	low, high, avail, reason := calcCostInterval(roundedVal, cfg)
	vm.CostLow = low
	vm.CostHigh = high
	vm.CostAvailable = avail
	vm.CostUnavailableReason = reason
	return nil
}
