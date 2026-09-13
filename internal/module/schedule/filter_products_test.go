package schedule

import (
	"context"
	"errors"
	"testing"

	"workbench/internal/model"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestListFilterProducts_ParticipatingProductsSortedFirst(t *testing.T) {
	db, mock := setupMockDB(t)
	repo := NewRepo(db)
	svc := NewService(repo, nil)

	ctx := context.Background()
	actor := &model.User{Account: "alice"}

	const listAllRegex = `(?s)SELECT id, name\s+FROM zt_product\s+WHERE deleted = '0'\s+ORDER BY id DESC`

	// ListAllProducts returns products 4, 3, 2, 1
	mock.ExpectQuery(listAllRegex).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).
			AddRow(4, "Product Delta").
			AddRow(3, "Product Gamma").
			AddRow(2, "Product Beta").
			AddRow(1, "Product Alpha"))

	// GetUserProducts for "alice" returns products 2 and 4 (user is PO or whitelist)
	mock.ExpectQuery(testGetUserProductsRegex).
		WithArgs("alice", "alice", "alice", "alice", "alice").
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "code", "status", "PO", "QD", "RD", "createdBy", "whitelist"}).
			AddRow(2, "Product Beta", "p2", "normal", "alice", "", "", "alice", "").
			AddRow(4, "Product Delta", "p4", "normal", "", "alice", "", "admin", ""))

	products, err := svc.ListFilterProducts(ctx, actor)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(products) != 4 {
		t.Fatalf("expected 4 products, got %d", len(products))
	}

	// Products 4 and 2 must be sorted first with IsMyProduct = true
	if !products[0].IsMyProduct || products[0].ID != 4 {
		t.Errorf("expected product 4 first with IsMyProduct=true, got ID=%d, isMy=%v", products[0].ID, products[0].IsMyProduct)
	}
	if !products[1].IsMyProduct || products[1].ID != 2 {
		t.Errorf("expected product 2 second with IsMyProduct=true, got ID=%d, isMy=%v", products[1].ID, products[1].IsMyProduct)
	}
	// Products 3 and 1 must follow with IsMyProduct = false
	if products[2].IsMyProduct || products[2].ID != 3 {
		t.Errorf("expected product 3 third with IsMyProduct=false, got ID=%d, isMy=%v", products[2].ID, products[2].IsMyProduct)
	}
	if products[3].IsMyProduct || products[3].ID != 1 {
		t.Errorf("expected product 1 fourth with IsMyProduct=false, got ID=%d, isMy=%v", products[3].ID, products[3].IsMyProduct)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestListFilterProducts_NilOrEmptyActor(t *testing.T) {
	db, mock := setupMockDB(t)
	repo := NewRepo(db)
	svc := NewService(repo, nil)

	ctx := context.Background()

	const listAllRegex = `(?s)SELECT id, name\s+FROM zt_product\s+WHERE deleted = '0'\s+ORDER BY id DESC`

	mock.ExpectQuery(listAllRegex).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).
			AddRow(2, "Product Beta").
			AddRow(1, "Product Alpha"))

	products, err := svc.ListFilterProducts(ctx, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(products) != 2 {
		t.Fatalf("expected 2 products, got %d", len(products))
	}
	if products[0].ID != 2 || products[0].IsMyProduct {
		t.Errorf("expected product 2 first with IsMyProduct=false, got %v", products[0])
	}
	if products[1].ID != 1 || products[1].IsMyProduct {
		t.Errorf("expected product 1 second with IsMyProduct=false, got %v", products[1])
	}
}

func TestListFilterProducts_ErrorPropagation(t *testing.T) {
	db, mock := setupMockDB(t)
	repo := NewRepo(db)
	svc := NewService(repo, nil)

	ctx := context.Background()
	const listAllRegex = `(?s)SELECT id, name\s+FROM zt_product\s+WHERE deleted = '0'\s+ORDER BY id DESC`

	mock.ExpectQuery(listAllRegex).
		WillReturnError(errors.New("db connection failure"))

	_, err := svc.ListFilterProducts(ctx, &model.User{Account: "alice"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
