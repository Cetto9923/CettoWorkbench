//go:build integration

// =============================================================================
// 文件: tests/integration/synthetic_fixtures.go
// 模块: 性能基线测试
// 类型: test
// 职责: 生成可重复、固定种子的合成性能数据集（通知、待办、窗口）。
// 依赖: workbench/internal/module/po
// =============================================================================

package integration

import (
	"fmt"
	"math/rand"
	"time"

	"workbench/internal/module/po"
)

// SyntheticNoticeRow 表示用于基线测试的合成通知行。
type SyntheticNoticeRow struct {
	ID          int64
	ObjectType  string
	ObjectID    int64
	Subject     string
	Data        string
	ActionCode  string
	Actor       string
	CreatedBy   string
	CreatedDate time.Time
	ToList      string
	IsRead      int
	Deleted     int
}

// GenerateSyntheticNotices 依据固定种子生成规模为 N 的通知集合。
func GenerateSyntheticNotices(n int, seed int64, baseTime time.Time) []SyntheticNoticeRow {
	r := rand.New(rand.NewSource(seed))
	rows := make([]SyntheticNoticeRow, 0, n)

	objectTypes := []string{"demand", "story", "task", "bug", "approval"}
	actionCodes := []string{"created", "assigned", "closed", "commented", "reviewed"}
	actors := []string{"user_a", "user_b", "user_c", "admin"}

	for i := 1; i <= n; i++ {
		actor := actors[r.Intn(len(actors))]
		objType := objectTypes[r.Intn(len(objectTypes))]
		action := actionCodes[r.Intn(len(actionCodes))]
		createdDate := baseTime.Add(-time.Duration(i*30) * time.Minute)

		subject := fmt.Sprintf("通知标题-%s-%04d", objType, i)
		if i%20 == 0 {
			subject = fmt.Sprintf("进度达成 100%% 验收通知-%04d", i) // % 通配符测试
		} else if i%25 == 0 {
			subject = fmt.Sprintf("order_v2 联调提醒-%04d", i) // _ 通配符测试
		} else if i%30 == 0 {
			subject = fmt.Sprintf("排期交付·核心系统升级 🚀-%04d", i) // Unicode 测试
		}

		toList := actor
		if i%10 == 0 {
			toList = fmt.Sprintf("%s,user_b", actor) // 多人逗号分割
		}

		deleted := 0
		if i%50 == 0 {
			deleted = 1 // 软删除数据
		}

		rows = append(rows, SyntheticNoticeRow{
			ID:          int64(i),
			ObjectType:  objType,
			ObjectID:    int64(1000 + i),
			Subject:     subject,
			Data:        fmt.Sprintf("{\"content\":\"这是第%d条通知详细负载内容，模拟大文本体\"}", i),
			ActionCode:  action,
			Actor:       actor,
			CreatedBy:   "system",
			CreatedDate: createdDate,
			ToList:      toList,
			IsRead:      i % 3,
			Deleted:     deleted,
		})
	}
	return rows
}

// GenerateSyntheticTodos 依据固定种子生成规模为 N 的待办项目。
func GenerateSyntheticTodos(n int, seed int64, baseTime time.Time) []po.TodoItem {
	r := rand.New(rand.NewSource(seed))
	items := make([]po.TodoItem, 0, n)

	kinds := []string{"demand", "task", "bug"}
	priorities := []string{"P0", "P1", "P2", "P3", "P4"}
	stages := []string{"clarify", "developing", "testing", "waitacceptance"}
	relations := []string{"我负责", "我配合", "我关注"}
	responsibilities := []string{"待我处理", "待我跟进"}
	owners := []string{"user_a", "user_b"}

	for i := 1; i <= n; i++ {
		kind := kinds[r.Intn(len(kinds))]
		priority := priorities[r.Intn(len(priorities))]
		owner := owners[r.Intn(len(owners))]

		deadline := baseTime.AddDate(0, 0, (i%15)-5).Format("2006-01-02")
		if i%12 == 0 {
			deadline = "" // 空日期
		} else if i%35 == 0 {
			deadline = "0000-00-00" // 零日期
		}

		title := fmt.Sprintf("待办项目-%s-%04d", kind, i)
		if i%20 == 0 {
			title = fmt.Sprintf("修复 order_v2 缺陷-%04d", i)
		} else if i%25 == 0 {
			title = fmt.Sprintf("达成 100%% 覆盖率目标-%04d", i)
		}

		items = append(items, po.TodoItem{
			Kind:           kind,
			ID:             int64(i),
			DisplayID:      fmt.Sprintf("%s-%d", stringsToUpper(kind), i),
			Title:          title,
			Type:           kind,
			Stage:          stages[r.Intn(len(stages))],
			Priority:       priority,
			Relation:       relations[r.Intn(len(relations))],
			Responsibility: responsibilities[r.Intn(len(responsibilities))],
			Deadline:       deadline,
			Owner:          owner,
			Blocked:        i%15 == 0,
		})
	}
	return items
}

func stringsToUpper(s string) string {
	switch s {
	case "demand":
		return "US"
	case "task":
		return "TASK"
	case "bug":
		return "BUG"
	default:
		return "ITEM"
	}
}
