package po

import (
	"context"
	"strings"

	"workbench/internal/model"
)

// SubmitTestPage loads a server-derived submit-test context. It intentionally
// does not accept Story IDs: the repository derives the complete unit scope.
func (s *Service) SubmitTestPage(ctx context.Context, actor *model.User, demandID uint) (*SubmitTestPageData, error) {
	if s == nil || s.repo == nil || actor == nil || strings.TrimSpace(actor.Account) == "" {
		return nil, errSubmitTestForbidden
	}
	if _, err := s.DetailService().GetDemandDetailAuthZ(ctx, actor, demandID); err != nil {
		return nil, err
	}
	return s.repo.SubmitTestPage(ctx, demandID)
}
