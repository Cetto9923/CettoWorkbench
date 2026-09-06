// =============================================================================
// 文件: internal/module/po/servicereview.go
// 模块: PO 工作台
// 类型: action
// 职责: 业需评审业务规则（对照禅道 demand->review，不含 OA 同步与转工单）。
// 依赖: internal/model
//       internal/pkg/errorx
// =============================================================================

package po

import (
	"context"
	"errors"
	"strings"

	"workbench/internal/model"
	"workbench/internal/pkg/errorx"
)

// ReviewDemand 提交业需评审。
//
// 读代码时可按 PHP/Java 这样理解：
//  1. actor 就是当前登录用户（Service 里不碰 gin.Context）
//  2. 先校验「待评审 + 当前账号是未出结果的业务评审人」（与 assignedTo 无关）
//  3. 再按禅道规则改三张表：评审人结果、需求字段、操作历史
func (s *Service) ReviewDemand(ctx context.Context, actor *model.User, req ReviewDemandReq) (ReviewDemandResp, error) {
	empty := ReviewDemandResp{}
	if actor == nil || strings.TrimSpace(actor.Account) == "" {
		return empty, errorx.New(errorx.ErrCodeForbidden, "请先登录")
	}
	account := strings.TrimSpace(actor.Account)

	demand, err := s.repo.FindDemandForReview(ctx, req.ID)
	if err != nil {
		return empty, err
	}
	if demand == nil || demand.Deleted != "0" {
		return empty, errorx.New(errorx.ErrCodeNotFound, "需求不存在")
	}
	// 对齐禅道：status=wait 且 zt_demandreview 里有当前账号、result 仍为空。
	if strings.TrimSpace(demand.Status) != "wait" {
		return empty, errorx.New(errorx.ErrCodeConflict, "该需求不是待评审状态")
	}
	reviewResult, found, revErr := s.repo.FindDemandReviewerResult(ctx, req.ID, account)
	if revErr != nil {
		return empty, revErr
	}
	if !found {
		return empty, errorx.New(errorx.ErrCodeForbidden, "只有业务评审人可以评审")
	}
	if strings.TrimSpace(reviewResult) != "" {
		return empty, errorx.New(errorx.ErrCodeConflict, "您已评审过该需求")
	}

	reviewedBy := appendUniqueAccount(demand.ReviewedBy, account)
	mailto := normalizeMailto(req.Mailto)

	// 拒绝：立刻变成 refuse，并指派回创建人。
	// 通过：事务里锁需求行，再统计尚未 pass 的评审人（含 NULL）；为 0 才改成 active。
	newStatus := ""
	statusAction := ""
	assignBack := ""
	if req.Result == "refuse" {
		newStatus = "refuse"
		statusAction = "reviewrejected"
		assignBack = strings.TrimSpace(demand.CreatedBy)
	}

	if err := s.repo.SaveDemandReview(ctx, saveDemandReviewIn{
		DemandID:     req.ID,
		Account:      account,
		Result:       req.Result,
		IsNeedFocus:  req.IsNeedFocus,
		Mailto:       mailto,
		ReviewedBy:   reviewedBy,
		Comment:      req.Comment,
		Product:      demand.Product,
		NewStatus:    newStatus,
		StatusAction: statusAction,
		AssignBackTo: assignBack,
	}); err != nil {
		if errors.Is(err, errDemandNotReviewable) || errors.Is(err, errAlreadyReviewed) {
			return empty, errorx.New(errorx.ErrCodeConflict, err.Error())
		}
		if isLockWait(err) {
			return empty, errorx.New(errorx.ErrCodeConflict, "该需求正在被他人编辑，请稍后重试")
		}
		return empty, err
	}

	return ReviewDemandResp{ID: req.ID}, nil
}

// appendUniqueAccount 把当前账号追加进 reviewedBy（逗号分隔，去重）。
func appendUniqueAccount(old, account string) string {
	seen := map[string]bool{}
	var out []string
	for _, part := range strings.Split(old, ",") {
		p := strings.TrimSpace(part)
		if p == "" || seen[p] {
			continue
		}
		seen[p] = true
		out = append(out, p)
	}
	account = strings.TrimSpace(account)
	if account != "" && !seen[account] {
		out = append(out, account)
	}
	return strings.Join(out, ",")
}

// normalizeMailto 把「张三, 004861」收成禅道常用的逗号串；空则不改库。
func normalizeMailto(raw string) *string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var parts []string
	for _, p := range strings.Split(raw, ",") {
		p = strings.TrimSpace(p)
		if p != "" {
			parts = append(parts, p)
		}
	}
	if len(parts) == 0 {
		return nil
	}
	joined := strings.Join(parts, ",")
	return &joined
}

func isLockWait(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "lock wait timeout") || strings.Contains(msg, "deadlock")
}
