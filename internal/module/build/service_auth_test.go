package build

import (
	"context"
	"testing"

	"workbench/internal/pkg/errorx"
)

func TestListLinkedStoriesRequiresAuthenticatedActor(t *testing.T) {
	svc := NewService(nil, nil, nil, nil)

	_, err := svc.ListLinkedStories(context.Background(), nil, 12)
	if err == nil {
		t.Fatal("expected authentication error, got nil")
	}
	bizErr, ok := errorx.IsBizError(err)
	if !ok || bizErr.Code != errorx.ErrCodeForbidden {
		t.Fatalf("expected %q, got %v", errorx.ErrCodeForbidden, err)
	}
}
