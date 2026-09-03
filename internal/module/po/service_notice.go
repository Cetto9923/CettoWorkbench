// =============================================================================
// 文件: internal/module/po/service_notice.go
// 模块: PO 工作台
// 类型: action
// 职责: 通知中心服务。workbench 为主: 不强套 V10.1 6 类, 分类由 zt_action + objectType 启发式。
//       严守 V10.1 03 节契约: 已读 ≠ 已处理。
// 依赖: 无
// =============================================================================

package po

import (
	"context"
	"strings"

	"workbench/internal/model"
)

// NoticeList 通知中心列表 + quick view 计数。
func (s *Service) NoticeList(ctx context.Context, actor *model.User, req NoticeListReq) (*NoticeBucketResp, error) {
	if actor == nil || strings.TrimSpace(actor.Account) == "" {
		return &NoticeBucketResp{Items: []NoticeItem{}, Page: req.Page, PageSize: req.PageSize}, nil
	}
	items, total, unread, action, abnormal, today, err := s.repo.FindNotices(ctx, actor.Account, req)
	if err != nil {
		return nil, err
	}
	return &NoticeBucketResp{
		Items:    items,
		Total:    total,
		Unread:   unread,
		Action:   action,
		Abnormal: abnormal,
		Today:    today,
		Page:     req.Page,
		PageSize: req.PageSize,
	}, nil
}

// NoticeMarkRead 标记单条已读。
func (s *Service) NoticeMarkRead(ctx context.Context, actor *model.User, notifyID int64) error {
	if actor == nil || strings.TrimSpace(actor.Account) == "" {
		return nil
	}
	return s.repo.MarkNoticeRead(ctx, actor.Account, notifyID)
}

// NoticeMarkAllRead 标记全部已读。
func (s *Service) NoticeMarkAllRead(ctx context.Context, actor *model.User) (int64, error) {
	if actor == nil || strings.TrimSpace(actor.Account) == "" {
		return 0, nil
	}
	return s.repo.MarkAllNoticesRead(ctx, actor.Account)
}

