//go:build integration

// =============================================================================
// 文件: tests/integration/db_seed.go
// 模块: 性能基线测试
// 类型: test
// 职责: 向隔离测试数据库灌入确定性合成基准测试数据。
// 依赖: gorm.io/gorm
// =============================================================================

package integration

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// SeedSyntheticDatabase 在隔离库中重置并填充合成数据。
func SeedSyntheticDatabase(ctx context.Context, db *gorm.DB, nNotices int, nTodos int, nWindows int) error {
	tables := []string{
		"zt_workbench_notify_reads", "zt_notify", "zt_action", "zt_user",
		"zt_demandclarify", "zt_demand", "zt_task", "zt_story", "zt_bug",
		"zt_approvalnode", "zt_approvalobject",
		"zt_demandwindow", "zt_versionwindowproduct", "zt_planstory",
		"zt_versionwindow", "zt_dept",
	}

	for _, t := range tables {
		if err := db.WithContext(ctx).Exec(fmt.Sprintf("DELETE FROM %s", t)).Error; err != nil {
			return fmt.Errorf("failed to truncate table %s: %w", t, err)
		}
	}

	baseTime, _ := time.Parse(time.RFC3339, "2026-09-05T00:00:00Z")

	// 1. Seed Users
	users := []map[string]interface{}{
		{"id": 1, "account": "user_a", "realname": "张三 (PO)", "role": "po", "deleted": "0"},
		{"id": 2, "account": "user_b", "realname": "李四 (Dev)", "role": "dev", "deleted": "0"},
		{"id": 3, "account": "user_c", "realname": "王五 (QA)", "role": "qa", "deleted": "0"},
	}
	for _, u := range users {
		if err := db.WithContext(ctx).Table("zt_user").Create(&u).Error; err != nil {
			return fmt.Errorf("failed to seed user: %w", err)
		}
	}

	// 2. Seed Teamgroups (zt_dept & zt_teamgroup)
	depts := []map[string]interface{}{
		{"id": 1, "name": "研发中心", "parent": 0, "path": ",1,", "grade": 1, "order": 1},
		{"id": 2, "name": "敏捷一组", "parent": 1, "path": ",1,2,", "grade": 2, "order": 1},
		{"id": 3, "name": "敏捷二组", "parent": 1, "path": ",1,3,", "grade": 2, "order": 2},
	}
	for _, d := range depts {
		if err := db.WithContext(ctx).Table("zt_dept").Create(&d).Error; err != nil {
			return fmt.Errorf("failed to seed dept: %w", err)
		}
	}

	teamgroups := []map[string]interface{}{
		{"id": 1, "name": "核心小组", "parent": 0, "path": ",1,", "deleted": "0"},
		{"id": 2, "name": "敏捷二组", "parent": 0, "path": ",2,", "deleted": "0"},
	}
	for _, tg := range teamgroups {
		if err := db.WithContext(ctx).Table("zt_teamgroup").Create(&tg).Error; err != nil {
			return fmt.Errorf("failed to seed teamgroup: %w", err)
		}
	}

	products := []map[string]interface{}{
		{"id": 1, "name": "核心产品", "code": "CP01", "status": "normal", "PO": "user_a", "deleted": "0"},
	}
	for _, p := range products {
		if err := db.WithContext(ctx).Table("zt_product").Create(&p).Error; err != nil {
			return fmt.Errorf("failed to seed product: %w", err)
		}
	}

	// 3. Seed Notices
	syntheticNotices := GenerateSyntheticNotices(nNotices, 42, baseTime)
	for _, n := range syntheticNotices {
		actionID := uint(0)
		if n.ID%2 == 0 {
			actionID = uint(n.ID)
			act := map[string]interface{}{
				"id":         actionID,
				"objectType": n.ObjectType,
				"objectID":   n.ObjectID,
				"actor":      n.Actor,
				"action":     n.ActionCode,
				"date":       n.CreatedDate,
				"read":       "0",
			}
			_ = db.WithContext(ctx).Table("zt_action").Create(&act).Error
		}

		notifyRow := map[string]interface{}{
			"id":          n.ID,
			"objectType":  n.ObjectType,
			"objectID":    n.ObjectID,
			"action":      actionID,
			"toList":      n.ToList,
			"subject":     n.Subject,
			"data":        n.Data,
			"createdBy":   n.CreatedBy,
			"createdDate": n.CreatedDate,
			"status":      "sent",
		}
		if err := db.WithContext(ctx).Table("zt_notify").Create(&notifyRow).Error; err != nil {
			return fmt.Errorf("failed to seed notify: %w", err)
		}

		if n.IsRead == 1 {
			readRow := map[string]interface{}{
				"notify":  n.ID,
				"account": "user_a",
				"readAt":  n.CreatedDate.Add(5 * time.Minute),
			}
			_ = db.WithContext(ctx).Table("zt_workbench_notify_reads").Create(&readRow).Error
		}
	}

	// 4. Seed Demands, Tasks, Bugs
	for i := 1; i <= nTodos; i++ {
		demandRow := map[string]interface{}{
			"id":          i,
			"parent":      0,
			"name":        fmt.Sprintf("业务需求-%04d", i),
			"assignedTo":  "user_a",
			"pri":         fmt.Sprintf("%d", (i%4)+1),
			"status":      "active",
			"stage":       "developing",
			"deadline":    baseTime.AddDate(0, 0, i%10).Format("2006-01-02"),
			"createdDate": baseTime.Add(-time.Duration(i) * time.Hour),
			"deleted":     "0",
		}
		if i%20 == 0 {
			demandRow["deleted"] = "1"
		}
		_ = db.WithContext(ctx).Table("zt_demand").Create(&demandRow).Error

		taskRow := map[string]interface{}{
			"id":         i,
			"name":       fmt.Sprintf("开发任务-%04d", i),
			"assignedTo": "user_a",
			"pri":        (i % 4) + 1,
			"status":     "doing",
			"deadline":   baseTime.AddDate(0, 0, i%10).Format("2006-01-02"),
			"deleted":    "0",
			"consumed":   float64(i % 8),
		}
		_ = db.WithContext(ctx).Table("zt_task").Create(&taskRow).Error

		bugRow := map[string]interface{}{
			"id":         i,
			"title":      fmt.Sprintf("缺陷问题-%04d", i),
			"assignedTo": "user_a",
			"pri":        (i % 4) + 1,
			"status":     "active",
			"deadline":   baseTime.AddDate(0, 0, i%10).Format("2006-01-02"),
			"deleted":    "0",
		}
		_ = db.WithContext(ctx).Table("zt_bug").Create(&bugRow).Error
	}

	// 5. Seed Windows
	for i := 1; i <= nWindows; i++ {
		releaseDate := baseTime.AddDate(0, 0, i*14)
		startDate := releaseDate.AddDate(0, 0, -14)
		wRow := map[string]interface{}{
			"id":          i,
			"name":        fmt.Sprintf("2026-%02d 窗口", i),
			"releaseDate": releaseDate.Format("2006-01-02"),
			"startDate":   startDate.Format("2006-01-02"),
			"teamgroup":   2,
			"groupSize":   5,
			"createdBy":   "user_a",
			"status":      "planning",
			"order":       i,
		}
		_ = db.WithContext(ctx).Table("zt_versionwindow").Create(&wRow).Error

		// Window demand links
		dwRow := map[string]interface{}{
			"demand":        i,
			"story":         0,
			"versionWindow": i,
			"createdBy":     "user_a",
		}
		_ = db.WithContext(ctx).Table("zt_demandwindow").Create(&dwRow).Error
	}

	return nil
}
