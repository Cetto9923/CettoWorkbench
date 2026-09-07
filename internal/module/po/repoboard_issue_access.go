package po

import (
	"context"
	"strings"

	"workbench/internal/pkg/errorx"
)

// CheckBoardIssueAccess 校验当前用户是否可操作该禅道问题；状态变更仍由禅道原生接口执行。
func (r *Repo) CheckBoardIssueAccess(ctx context.Context, account string, issueID int64) error {
	if r == nil || r.db == nil || strings.TrimSpace(account) == "" || issueID <= 0 {
		return errorx.New(errorx.ErrCodeForbidden, "无权操作该问题")
	}
	var visible int64
	err := r.db.WithContext(ctx).Table("zt_issue").
		Where("id = ? AND deleted = ?", issueID, "0").
		Where("(createdBy = ? OR assignedTo = ?)", account, account).
		Count(&visible).Error
	if err != nil {
		return err
	}
	if visible == 0 {
		return errorx.New(errorx.ErrCodeForbidden, "无权操作该问题")
	}
	return nil
}
