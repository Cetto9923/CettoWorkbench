package schedule

import (
	"context"
	"testing"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func TestCountActualWorkdaysExcludesDragonBoatFestival(t *testing.T) {
	dsn := "root:zentao@tcp(wrk.oop.cc:3306)/zentao_changshu?charset=utf8mb4&parseTime=True&loc=Local"
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Skip("database unavailable:", err)
	}
	svc := NewService(NewRepo(db), nil)
	ctx := context.Background()

	workdays, err := svc.CountActualWorkdays(ctx, "2026-06-17", "2026-06-30")
	if err != nil {
		t.Fatal(err)
	}
	if workdays != 9 {
		t.Fatalf("workdays = %d, want 9 (exclude 2026-06-19~21 holiday)", workdays)
	}

	capacity, err := svc.CalcCapacity(ctx, "2026-06-17", "2026-06-30", 1)
	if err != nil {
		t.Fatal(err)
	}
	if capacity != 63 {
		t.Fatalf("capacity = %d, want 63 (9 days * 7h)", capacity)
	}
}
