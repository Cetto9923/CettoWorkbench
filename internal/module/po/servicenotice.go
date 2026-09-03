// =============================================================================
// 文件: internal/module/po/servicenotice.go
// 模块: PO 工作台
// 类型: action
// 职责: 通知中心服务。按服务端事件分类口径聚合真实通知与已读状态。
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
	repoResp, err := s.repo.FindNotices(ctx, actor.Account, req)
	if err != nil {
		return nil, err
	}
	return &NoticeBucketResp{
		Items:      repoResp.Items,
		Total:      repoResp.Total,
		Filtered:   repoResp.Filtered,
		Unread:     repoResp.Unread,
		Action:     repoResp.Action,
		Abnormal:   repoResp.Abnormal,
		Today:      repoResp.Today,
		Categories: repoResp.Categories,
		Page:       req.Page,
		PageSize:   req.PageSize,
	}, nil
}

// NoticeMarkRead 标记单条已读。
func (s *Service) NoticeMarkRead(ctx context.Context, actor *model.User, notifyID int64) error {
	if actor == nil || strings.TrimSpace(actor.Account) == "" {
		return nil
	}
	return s.repo.SaveNoticeRead(ctx, actor.Account, notifyID)
}

// NoticeMarkAllRead 标记全部已读。
func (s *Service) NoticeMarkAllRead(ctx context.Context, actor *model.User) (int64, error) {
	if actor == nil || strings.TrimSpace(actor.Account) == "" {
		return 0, nil
	}
	return s.repo.SaveAllNoticeReads(ctx, actor.Account)
}
