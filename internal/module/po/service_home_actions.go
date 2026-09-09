package po

import (
	"context"
	"errors"
	"strings"

	"workbench/internal/model"
)

var (
	errHomeActionNotFound  = errors.New("需求不存在")
	errHomeActionForbidden = errors.New("无权办理该需求")
	errHomeActionConflict  = errors.New("需求状态已变化，请刷新后重试")
)

func (s *Service) homeAction(ctx context.Context, actor *model.User, id uint, action string, comment string) error {
	if s == nil || s.repo == nil || id == 0 {
		return errHomeActionNotFound
	}
	account := ""
	if actor != nil {
		account = strings.TrimSpace(actor.Account)
	}
	if account == "" {
		return errHomeActionForbidden
	}
	row, err := s.repo.findHomeActionDemand(ctx, id)
	if err != nil {
		return err
	}
	if !s.repo.homeActionAuthorized(row, account, action) {
		return errHomeActionForbidden
	}
	switch action {
	case "clarify":
		if row.Status != "active" {
			return errHomeActionConflict
		}
		return s.repo.updateHomeDemandStatus(ctx, id, "active", "clarified", account, "clarify", homeActionComment(comment, "工作台澄清完成"), false)
	case "acceptance":
		if row.Status != "testing" && row.Status != "waitacceptance" {
			return errHomeActionConflict
		}
		return s.repo.updateHomeDemandStatus(ctx, id, row.Status, "acceptanced", account, "verified", homeActionComment(comment, "工作台验收完成"), true)
	case "deliver":
		if row.Status != "acceptanced" {
			return errHomeActionConflict
		}
		return s.repo.updateHomeDemandStatus(ctx, id, "acceptanced", "waitdeliver", account, "deliver", homeActionComment(comment, "工作台发起交付"), false)
	case "urge":
		return s.repo.insertHomeDemandAction(ctx, id, account, "reminded", homeActionComment(comment, "工作台催办验收"))
	default:
		return errHomeActionConflict
	}
}

func homeActionComment(comment, fallback string) string {
	if strings.TrimSpace(comment) == "" {
		return fallback
	}
	return strings.TrimSpace(comment)
}

func (s *Service) ClarifyHomeDemand(ctx context.Context, actor *model.User, id uint, comment string) error {
	return s.homeAction(ctx, actor, id, "clarify", comment)
}
func (s *Service) AcceptHomeDemand(ctx context.Context, actor *model.User, id uint, comment string) error {
	return s.homeAction(ctx, actor, id, "acceptance", comment)
}
func (s *Service) DeliverHomeDemand(ctx context.Context, actor *model.User, id uint, comment string) error {
	return s.homeAction(ctx, actor, id, "deliver", comment)
}
func (s *Service) UrgeHomeDemand(ctx context.Context, actor *model.User, id uint, comment string) error {
	return s.homeAction(ctx, actor, id, "urge", comment)
}
