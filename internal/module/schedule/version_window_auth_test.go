package schedule

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"workbench/internal/model"

	"github.com/DATA-DOG/go-sqlmock"
)

const (
	testGetUserProductsRegex  = `(?s)SELECT id, name, code, status, PO, QD, RD, createdBy, whitelist\s+FROM zt_product\s+WHERE deleted = '0' AND status != 'closed'`
	testFindProductsByIDRegex = `(?s)SELECT id, name\s+FROM zt_product\s+WHERE id IN \((?:\?|,\s*|\?)+\)\s+AND deleted = '0'`
	testFindWindowByIDRegex   = `(?s)SELECT \* FROM ` + "`zt_versionwindow`" + ` WHERE id = \? AND ` + "`zt_versionwindow`" + `\.` + "`deletedAt`" + ` IS NULL`
	testGetMatchingPlansRegex = `(?s)SELECT id, product, title, begin, end, status\s+FROM zt_productplan\s+WHERE product = \? AND end = \? AND deleted = '0'`
)

// 1. CreateUnauthorizedProductRejected:
// actor attempts to create a window referencing an unauthorized product.
// Must reject before transaction / mutations with zero writes.
func TestVersionWindow_CreateUnauthorizedProductRejected(t *testing.T) {
	db, mock := setupMockDB(t)
	repo := NewRepo(db)
	svc := NewService(repo, nil)

	ctx := context.Background()
	actor := &model.User{Account: "alice"}

	// Actor has access to product 100 only; product 200 is unauthorized.
	mock.ExpectQuery(testGetUserProductsRegex).
		WithArgs("alice", "alice", "alice", "alice", "alice").
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "code", "status", "PO", "QD", "RD", "createdBy", "whitelist"}).
			AddRow(100, "Product 100", "p100", "normal", "alice", "", "", "alice", ""))

	mock.ExpectQuery(testFindProductsByIDRegex).
		WithArgs(200).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).
			AddRow(200, "Unauthorized System"))

	req := CreateReq{
		Name:        "Test Window",
		ReleaseDate: "2026-10-01",
		TeamgroupID: 1,
		GroupSize:   1,
		Products: []WindowProductInput{
			{ProductID: 200, SyncPlan: false},
		},
	}

	err := svc.Create(ctx, actor, req)
	if err == nil {
		t.Fatal("expected error for unauthorized product create, got nil")
	}

	var notice *ProductAccessNoticeError
	if !errors.As(err, &notice) {
		t.Fatalf("expected ProductAccessNoticeError, got: %v", err)
	}
	if len(notice.Products) != 1 || notice.Products[0].ID != 200 {
		t.Fatalf("expected notice for product 200, got: %+v", notice.Products)
	}
	if notice.Products[0].Name != "Unauthorized System" {
		t.Fatalf("expected product name 'Unauthorized System', got: %q", notice.Products[0].Name)
	}
	if !strings.Contains(err.Error(), "存在未在窗口中且无匹配计划的系统") {
		t.Fatalf("expected error message to contain '存在未在窗口中且无匹配计划的系统', got: %v", err)
	}

	// Met expectations prove NO ExpectBegin or write statements were called.
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet mock expectations: %v", err)
	}
}

// 2. CreateAuthorizedProductsStillWorks:
// actor creates a window with an authorized product.
// Transaction runs, writing version window and window product records.
func TestVersionWindow_CreateAuthorizedProductsStillWorks(t *testing.T) {
	db, mock := setupMockDB(t)
	repo := NewRepo(db)
	svc := NewService(repo, nil)

	ctx := context.Background()
	actor := &model.User{Account: "alice"}

	// Actor has access to product 100.
	mock.ExpectQuery(testGetUserProductsRegex).
		WithArgs("alice", "alice", "alice", "alice", "alice").
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "code", "status", "PO", "QD", "RD", "createdBy", "whitelist"}).
			AddRow(100, "Product 100", "p100", "normal", "alice", "", "", "alice", ""))

	mock.ExpectBegin()
	mock.ExpectExec("(?s)INSERT INTO `zt_versionwindow`").
		WillReturnResult(sqlmock.NewResult(1, 1))

	mock.ExpectQuery(testGetMatchingPlansRegex).
		WithArgs(100, "2026-10-01").
		WillReturnRows(sqlmock.NewRows([]string{"id", "product", "title", "begin", "end", "status"}))

	mock.ExpectExec("(?s)INSERT INTO `zt_productplan`").
		WillReturnResult(sqlmock.NewResult(10, 1))

	mock.ExpectExec("(?s)INSERT INTO `zt_versionwindowproduct`").
		WillReturnResult(sqlmock.NewResult(1, 1))

	mock.ExpectCommit()

	req := CreateReq{
		Name:        "Test Window",
		ReleaseDate: "2026-10-01",
		TeamgroupID: 1,
		GroupSize:   1,
		Products: []WindowProductInput{
			{ProductID: 100, SyncPlan: true, PlanTitle: "Auto Plan"},
		},
	}

	err := svc.Create(ctx, actor, req)
	if err != nil {
		t.Fatalf("expected nil error for authorized product create, got: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet mock expectations: %v", err)
	}
}

// 3. UpdateUnauthorizedProductRejected:
// actor updates a window but specifies an unauthorized product.
// Must reject before transaction / mutations with zero writes.
func TestVersionWindow_UpdateUnauthorizedProductRejected(t *testing.T) {
	db, mock := setupMockDB(t)
	repo := NewRepo(db)
	svc := NewService(repo, nil)

	ctx := context.Background()
	actor := &model.User{Account: "alice"}

	// Window exists and was created by alice.
	mock.ExpectQuery(testFindWindowByIDRegex).
		WithArgs(1, 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "releaseDate", "teamgroup", "groupSize", "createdBy", "status"}).
			AddRow(1, "Original Window", time.Now().AddDate(0, 1, 0), 1, 1, "alice", "planning"))

	// Actor does not have access to product 200.
	mock.ExpectQuery(testGetUserProductsRegex).
		WithArgs("alice", "alice", "alice", "alice", "alice").
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "code", "status", "PO", "QD", "RD", "createdBy", "whitelist"}))

	mock.ExpectQuery(testFindProductsByIDRegex).
		WithArgs(200).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).
			AddRow(200, "Unauthorized System"))

	req := UpdateReq{
		ID:          1,
		Name:        "Modified Window",
		ReleaseDate: "2026-10-01",
		TeamgroupID: 1,
		GroupSize:   1,
		Products: []WindowProductInput{
			{ProductID: 200, SyncPlan: false},
		},
	}

	err := svc.Update(ctx, actor, req)
	if err == nil {
		t.Fatal("expected error for unauthorized product update, got nil")
	}

	var notice *ProductAccessNoticeError
	if !errors.As(err, &notice) {
		t.Fatalf("expected ProductAccessNoticeError, got: %v", err)
	}
	if len(notice.Products) != 1 || notice.Products[0].ID != 200 {
		t.Fatalf("expected notice for product 200, got: %+v", notice.Products)
	}

	// Met expectations prove NO ExpectBegin, UPDATE, DELETE, or INSERT statements were executed.
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet mock expectations: %v", err)
	}
}

// 4. UpdateUnauthorizedLeavesExistingWindowUntouched:
// When unauthorized update is rejected, confirms existing window and relations in DB
// receive zero mutation calls (no Transaction, no UPDATE, no DELETE).
func TestVersionWindow_UpdateUnauthorizedLeavesExistingWindowUntouched(t *testing.T) {
	db, mock := setupMockDB(t)
	repo := NewRepo(db)
	svc := NewService(repo, nil)

	ctx := context.Background()
	actor := &model.User{Account: "alice"}

	origReleaseDate := time.Date(2026, 8, 15, 0, 0, 0, 0, time.Local)
	// Window exists with original data.
	mock.ExpectQuery(testFindWindowByIDRegex).
		WithArgs(1, 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "releaseDate", "teamgroup", "groupSize", "createdBy", "status"}).
			AddRow(1, "Original Window", origReleaseDate, 1, 1, "alice", "planning"))

	// Actor has access to product 100 only, not 999.
	mock.ExpectQuery(testGetUserProductsRegex).
		WithArgs("alice", "alice", "alice", "alice", "alice").
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "code", "status", "PO", "QD", "RD", "createdBy", "whitelist"}).
			AddRow(100, "Product 100", "p100", "normal", "alice", "", "", "alice", ""))

	mock.ExpectQuery(testFindProductsByIDRegex).
		WithArgs(999).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).
			AddRow(999, "Forbidden Product"))

	req := UpdateReq{
		ID:          1,
		Name:        "Attempted Malicious Rename",
		ReleaseDate: "2026-12-31",
		TeamgroupID: 2,
		GroupSize:   5,
		Products: []WindowProductInput{
			{ProductID: 999, SyncPlan: false},
		},
	}

	err := svc.Update(ctx, actor, req)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var notice *ProductAccessNoticeError
	if !errors.As(err, &notice) {
		t.Fatalf("expected ProductAccessNoticeError, got: %v", err)
	}

	// Strictly assert expectations: only read queries were issued; zero mutations.
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet mock expectations: %v", err)
	}
}

// 5. UpdateAuthorizedProductsStillWorks:
// actor updates their window with authorized products.
// Transaction runs, updating window and recreating window products.
func TestVersionWindow_UpdateAuthorizedProductsStillWorks(t *testing.T) {
	db, mock := setupMockDB(t)
	repo := NewRepo(db)
	svc := NewService(repo, nil)

	ctx := context.Background()
	actor := &model.User{Account: "alice"}

	mock.ExpectQuery(testFindWindowByIDRegex).
		WithArgs(1, 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "releaseDate", "teamgroup", "groupSize", "createdBy", "status"}).
			AddRow(1, "Original Window", time.Now().AddDate(0, 1, 0), 1, 1, "alice", "planning"))

	// Actor has access to product 100.
	mock.ExpectQuery(testGetUserProductsRegex).
		WithArgs("alice", "alice", "alice", "alice", "alice").
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "code", "status", "PO", "QD", "RD", "createdBy", "whitelist"}).
			AddRow(100, "Product 100", "p100", "normal", "alice", "", "", "alice", ""))

	mock.ExpectBegin()
	mock.ExpectExec("(?s)UPDATE `zt_versionwindow`").
		WillReturnResult(sqlmock.NewResult(1, 1))

	mock.ExpectExec("(?s)DELETE FROM `zt_versionwindowproduct` WHERE versionWindow = \\?").
		WithArgs(1).
		WillReturnResult(sqlmock.NewResult(0, 1))

	mock.ExpectQuery(testGetMatchingPlansRegex).
		WithArgs(100, "2026-10-01").
		WillReturnRows(sqlmock.NewRows([]string{"id", "product", "title", "begin", "end", "status"}).
			AddRow(50, 100, "Existing Plan", "2026-10-01", "2026-10-01", "wait"))

	mock.ExpectExec("(?s)INSERT INTO `zt_versionwindowproduct`").
		WillReturnResult(sqlmock.NewResult(1, 1))

	mock.ExpectCommit()

	req := UpdateReq{
		ID:          1,
		Name:        "Updated Window Name",
		ReleaseDate: "2026-10-01",
		TeamgroupID: 1,
		GroupSize:   1,
		Products: []WindowProductInput{
			{ProductID: 100, SyncPlan: false},
		},
	}

	err := svc.Update(ctx, actor, req)
	if err != nil {
		t.Fatalf("expected nil error for authorized product update, got: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet mock expectations: %v", err)
	}
}

// 6. MultipleProductsOneUnauthorizedRejectsWholeRequest:
// When request contains multiple products (e.g. 100, 200, 300) and only 200 is unauthorized,
// the entire request must be rejected prior to starting any transaction or mutation.
func TestVersionWindow_MultipleProductsOneUnauthorizedRejectsWholeRequest(t *testing.T) {
	ctx := context.Background()
	actor := &model.User{Account: "alice"}

	t.Run("Create", func(t *testing.T) {
		db, mock := setupMockDB(t)
		repo := NewRepo(db)
		svc := NewService(repo, nil)

		// Actor has access to 100 and 300, but lacks 200.
		mock.ExpectQuery(testGetUserProductsRegex).
			WithArgs("alice", "alice", "alice", "alice", "alice").
			WillReturnRows(sqlmock.NewRows([]string{"id", "name", "code", "status", "PO", "QD", "RD", "createdBy", "whitelist"}).
				AddRow(100, "Product 100", "p100", "normal", "alice", "", "", "alice", "").
				AddRow(300, "Product 300", "p300", "normal", "alice", "", "", "alice", ""))

		mock.ExpectQuery(testFindProductsByIDRegex).
			WithArgs(200).
			WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).
				AddRow(200, "Forbidden Product 200"))

		req := CreateReq{
			Name:        "Multi-product Window",
			ReleaseDate: "2026-10-01",
			TeamgroupID: 1,
			GroupSize:   1,
			Products: []WindowProductInput{
				{ProductID: 100, SyncPlan: false},
				{ProductID: 200, SyncPlan: false},
				{ProductID: 300, SyncPlan: false},
			},
		}

		err := svc.Create(ctx, actor, req)
		if err == nil {
			t.Fatal("expected rejection when one product of multiple is unauthorized, got nil")
		}

		var notice *ProductAccessNoticeError
		if !errors.As(err, &notice) {
			t.Fatalf("expected ProductAccessNoticeError, got: %v", err)
		}
		if len(notice.Products) != 1 || notice.Products[0].ID != 200 {
			t.Fatalf("expected offender product 200, got: %+v", notice.Products)
		}

		// Verify zero writes / zero transaction started.
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet mock expectations: %v", err)
		}
	})

	t.Run("Update", func(t *testing.T) {
		db, mock := setupMockDB(t)
		repo := NewRepo(db)
		svc := NewService(repo, nil)

		mock.ExpectQuery(testFindWindowByIDRegex).
			WithArgs(1, 1).
			WillReturnRows(sqlmock.NewRows([]string{"id", "name", "releaseDate", "teamgroup", "groupSize", "createdBy", "status"}).
				AddRow(1, "Original Window", time.Now().AddDate(0, 1, 0), 1, 1, "alice", "planning"))

		mock.ExpectQuery(testGetUserProductsRegex).
			WithArgs("alice", "alice", "alice", "alice", "alice").
			WillReturnRows(sqlmock.NewRows([]string{"id", "name", "code", "status", "PO", "QD", "RD", "createdBy", "whitelist"}).
				AddRow(100, "Product 100", "p100", "normal", "alice", "", "", "alice", "").
				AddRow(300, "Product 300", "p300", "normal", "alice", "", "", "alice", ""))

		mock.ExpectQuery(testFindProductsByIDRegex).
			WithArgs(200).
			WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).
				AddRow(200, "Forbidden Product 200"))

		req := UpdateReq{
			ID:          1,
			Name:        "Multi-product Window",
			ReleaseDate: "2026-10-01",
			TeamgroupID: 1,
			GroupSize:   1,
			Products: []WindowProductInput{
				{ProductID: 100, SyncPlan: false},
				{ProductID: 200, SyncPlan: false},
				{ProductID: 300, SyncPlan: false},
			},
		}

		err := svc.Update(ctx, actor, req)
		if err == nil {
			t.Fatal("expected rejection when one product of multiple is unauthorized, got nil")
		}

		var notice *ProductAccessNoticeError
		if !errors.As(err, &notice) {
			t.Fatalf("expected ProductAccessNoticeError, got: %v", err)
		}
		if len(notice.Products) != 1 || notice.Products[0].ID != 200 {
			t.Fatalf("expected offender product 200, got: %+v", notice.Products)
		}

		// Verify zero writes / zero transaction started.
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet mock expectations: %v", err)
		}
	})
}
