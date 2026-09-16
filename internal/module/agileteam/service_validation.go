// =============================================================================
// 文件: internal/module/agileteam/service_validation.go
// 模块: 敏捷小组治理
// 类型: validation
// 职责: Service 层独立校验写入边界，避免仅依赖 HTTP Handler 校验。
// =============================================================================

package agileteam

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"workbench/internal/pkg/errorx"
)

func hasControlRune(value string) bool {
	return strings.IndexFunc(value, func(r rune) bool { return unicode.IsControl(r) }) >= 0
}

func validateAdjustmentWriteBoundary(req SubmitAdjustmentReq) error {
	if req.TeamgroupID == 0 {
		return errorx.New("invalid", "小组 ID 无效")
	}
	if len(req.Items) == 0 {
		return errorx.New("invalid", "请至少提交一条调整")
	}
	if len(req.Items) > 100 {
		return errorx.New("invalid", "单次成员调整不能超过 100 条")
	}
	if utf8.RuneCountInString(strings.TrimSpace(req.Reason)) > 500 {
		return errorx.New("invalid", "调整说明不能超过 500 个字符")
	}

	seen := map[string]bool{}
	for _, item := range req.Items {
		account := strings.TrimSpace(item.Account)
		role := strings.TrimSpace(item.Role)
		action := strings.TrimSpace(item.ActionType)
		if account == "" || utf8.RuneCountInString(account) > 30 || hasControlRune(account) {
			return errorx.New("invalid", "成员账号为空、过长或包含非法字符")
		}
		if seen[account] {
			return errorx.New("invalid", "同一调整单内账号不能重复："+account)
		}
		seen[account] = true
		if action != ActionAdd && action != ActionRemove && action != ActionRoleChange {
			return errorx.New("invalid", "调整类型无效："+action)
		}
		if utf8.RuneCountInString(role) > 64 || hasControlRune(role) {
			return errorx.New("invalid", "成员角色过长或包含非法字符")
		}
		if action != ActionRemove && (item.AvailableHours < 0 || item.AvailableHours > 24) {
			return errorx.New("invalid", account+" 的可用工时必须在 0 到 24 之间")
		}
	}
	return nil
}

func invalidLogoURL(logo string) bool {
	lower := strings.ToLower(strings.TrimSpace(logo))
	if strings.HasPrefix(lower, "javascript:") || strings.HasPrefix(lower, "vbscript:") {
		return true
	}
	return strings.HasPrefix(lower, "data:") && strings.Contains(lower, "html")
}

func validateBasicWriteBoundary(req UpdateBasicReq) error {
	if req.ID == 0 {
		return errorx.New("invalid", "小组 ID 无效")
	}
	name := strings.TrimSpace(req.Name)
	if name == "" || utf8.RuneCountInString(name) > 100 || hasControlRune(name) {
		return errorx.New("invalid", "团队名称不能为空、过长或包含非法字符")
	}
	if utf8.RuneCountInString(strings.TrimSpace(req.Slogan)) > 200 || hasControlRune(req.Slogan) {
		return errorx.New("invalid", "团队口号过长或包含非法字符")
	}
	if utf8.RuneCountInString(strings.TrimSpace(req.Declaration)) > 2000 || hasControlRune(req.Declaration) {
		return errorx.New("invalid", "团队信条过长或包含非法字符")
	}
	logo := strings.TrimSpace(req.Logo)
	if utf8.RuneCountInString(logo) > 500 || hasControlRune(logo) || invalidLogoURL(logo) {
		return errorx.New("invalid", "Logo 过长、包含非法字符或使用了不允许的地址")
	}
	return nil
}
