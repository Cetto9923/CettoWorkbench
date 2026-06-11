package schedule

import (
	"context"
	"testing"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"workbench/internal/model"
)

func TestDeleteWindowProductsAllowsReinsert(t *testing.T) {
	dsn := "root:zentao@tcp(wrk.oop.cc:3306)/zentao_changshu?charset=utf8mb4&parseTime=True&loc=Local"
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Skip("database unavailable:", err)
	}

	repo := NewRepo(db)
	ctx := context.Background()
	const windowID uint64 = 1

	var before []model.VersionWindowProduct
	if err := db.Unscoped().Where("versionWindow = ?", windowID).Find(&before).Error; err != nil {
		t.Fatal(err)
	}
	if len(before) == 0 {
		t.Skip("window 1 has no products to test")
	}

	if err := repo.DeleteWindowProducts(ctx, windowID); err != nil {
		t.Fatal(err)
	}

	var afterDelete int64
	if err := db.Unscoped().Model(&model.VersionWindowProduct{}).
		Where("versionWindow = ?", windowID).
		Count(&afterDelete).Error; err != nil {
		t.Fatal(err)
	}
	if afterDelete != 0 {
		t.Fatalf("expected hard delete, got %d rows", afterDelete)
	}

	for _, row := range before {
		wp := &model.VersionWindowProduct{
			WindowID:   windowID,
			ProductID:  row.ProductID,
			PlanID:     row.PlanID,
			PlanSynced: row.PlanSynced,
			CreatedBy:  "admin",
			UpdatedBy:  "admin",
		}
		if err := repo.CreateWindowProduct(ctx, wp); err != nil {
			t.Fatalf("reinsert product %d: %v", row.ProductID, err)
		}
	}
}
