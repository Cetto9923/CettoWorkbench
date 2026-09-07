// =============================================================================
// 文件: internal/module/po/repokpi.go
// 模块: PO 工作台
// 类型: action
// 职责: 首页 5 个焦点摘要 KPI 真实计数（今日必推/阻塞/超期/挂起；actor role scope）。
//       MyPending 来自价值流「全部」阶段计数，由 Service 拼装。
// 依赖: 无
// =============================================================================

package po

import (
	"context"
	"strings"
	"time"
)

func (r *Repo) CountKPIToday(ctx context.Context, account string) (int64, error) {
	if r == nil || r.db == nil || strings.TrimSpace(account) == "" {
		return 0, nil
	}
	today := time.Now().Format("2006-01-02")
	var total int64
	err := r.roleDemandBase(ctx, account).
		Where("deadline IS NOT NULL AND deadline != '0000-00-00' AND deadline <= ?", today).
		Count(&total).Error
	return total, err
}

// CountKPIOverdue 统计超期：today > deadline 且未完成，缺日期不算。
// V10.1 01 节：today > deadline 且未完成（缺日期不算超期）。
func (r *Repo) CountKPIOverdue(ctx context.Context, account string) (int64, error) {
	if r == nil || r.db == nil || strings.TrimSpace(account) == "" {
		return 0, nil
	}
	today := time.Now().Format("2006-01-02")
	var total int64
	err := r.roleDemandBase(ctx, account).
		Where("deadline IS NOT NULL AND deadline != '0000-00-00' AND deadline < ?", today).
		Count(&total).Error
	return total, err
}

// CountKPISuspended 统计挂起：hang='1' 且未关闭。
// V10.1 01 节：存在未闭合 hangup 区间。zentao hang='1' 即为挂起中。
func (r *Repo) CountKPISuspended(ctx context.Context, account string) (int64, error) {
	if r == nil || r.db == nil || strings.TrimSpace(account) == "" {
		return 0, nil
	}
	var total int64
	err := r.roleDemandBase(ctx, account).
		Where("hang = ?", "1").
		Count(&total).Error
	return total, err
}

// CountKPIBlocked 统计当前仍处于驳回状态的需求。
// 与参考项目 deriveRisks 一致：历史拒绝及单纯超期不代表当前阻塞。
// 尚无经确认的其他阻塞/解除事实源，不从历史审批 JSON 推断当前状态。
func (r *Repo) CountKPIBlocked(ctx context.Context, account string) (int64, error) {
	if r == nil || r.db == nil || strings.TrimSpace(account) == "" {
		return 0, nil
	}
	var n int64
	err := r.roleDemandBase(ctx, account).Where("status = ?", "refuse").Count(&n).Error
	return n, err
}

// TodoScopeFilter 我的待办过滤条件（V10.1 02 节 7 维 AND 简化版）。
type TodoScopeFilter struct {
	Keyword string // 关键词：标题 / ID
}

// FindTodoItems 查询我的待办列表（去重后按对象 ID 返回）。
// V10.1 02 节：待办 = Tab ∩ 场景 ∩ 阶段 ∩ 对象 ∩ 我的关系 ∩ 办理责任 ∩ 关键词。
// 本期实现：actor scope（PM in clarify ∪ QD ∪ RD ∪ BRA）+ 关键词 + 对象（业务需求/任务/Bug）。
// 研发需求/测试单/审批作为扩展点预留。
