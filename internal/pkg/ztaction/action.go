// =============================================================================
// 文件: internal/pkg/ztaction/action.go
// 模块: 基础设施
// 类型: infra
// 职责: 按禅道 action/logHistory 写入 zt_action 与 zt_history。使用调用方传入的 db，以便进入同一事务。
// 依赖: internal/model/zentao
// =============================================================================

package ztaction

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"

	ztmodel "workbench/internal/model/zentao"
)

// Record 一条禅道操作日志。Action、ObjectType 写入前会转成小写，对齐 actionModel::create。
type Record struct {
	ObjectType string
	ObjectID   uint
	ProductID  uint
	Product    string
	Project    uint
	Execution  uint
	Actor      string
	Action     string
	Comment    string
	Extra      string
	Date       time.Time
}

// Change 一条字段差异，对应 zt_history 的一行。
type Change struct {
	Field string
	Old   string
	New   string
	Diff  string
}

// FormatProductID 把单个产品 ID 收成禅道 zt_action.product 的 ",12," 形式。0 记为 ",0,"。
func FormatProductID(id uint) string {
	if id == 0 {
		return ",0,"
	}
	return fmt.Sprintf(",%d,", id)
}

// FormatProduct 把产品字符串收成 ",id,"。已带首尾逗号的原样保留，空或 0 记为 ",0,"。
func FormatProduct(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "0" {
		return ",0,"
	}
	if strings.HasPrefix(raw, ",") && strings.HasSuffix(raw, ",") {
		return raw
	}
	return "," + raw + ","
}

// Create 插入 zt_action 并返回 ID。
func Create(ctx context.Context, db *gorm.DB, rec Record) (uint, error) {
	if db == nil {
		return 0, errors.New("db is nil")
	}
	at := rec.Date
	if at.IsZero() {
		at = time.Now()
	}
	product := strings.TrimSpace(rec.Product)
	if product == "" {
		product = FormatProductID(rec.ProductID)
	} else {
		product = FormatProduct(product)
	}
	row := ztmodel.ZtAction{
		ObjectType: strings.ToLower(strings.TrimSpace(rec.ObjectType)),
		ObjectID:   rec.ObjectID,
		Product:    product,
		Project:    rec.Project,
		Execution:  rec.Execution,
		Actor:      rec.Actor,
		Action:     strings.ToLower(strings.TrimSpace(rec.Action)),
		Date:       at,
		Comment:    rec.Comment,
		Extra:      rec.Extra,
	}
	err := db.WithContext(ctx).Session(&gorm.Session{NewDB: true}).Create(&row).Error
	if err != nil {
		return 0, err
	}
	return row.ID, nil
}

// LogHistory 把字段差异写入 zt_history。actionID 为 0 或没有差异时不写。
func LogHistory(ctx context.Context, db *gorm.DB, actionID uint, changes []Change) error {
	if actionID == 0 || len(changes) == 0 {
		return nil
	}
	if db == nil {
		return errors.New("db is nil")
	}
	rows := make([]ztmodel.ZtHistory, 0, len(changes))
	for _, change := range changes {
		if strings.TrimSpace(change.Field) == "" || change.Old == change.New {
			continue
		}
		rows = append(rows, ztmodel.ZtHistory{
			Action: actionID,
			Field:  change.Field,
			Old:    change.Old,
			New:    change.New,
			Diff:   change.Diff,
		})
	}
	if len(rows) == 0 {
		return nil
	}
	return db.WithContext(ctx).Session(&gorm.Session{NewDB: true}).Create(&rows).Error
}

// LogEdited 有字段变化时写 action=edited 及 zt_history；无变化不写。
func LogEdited(ctx context.Context, db *gorm.DB, rec Record, changes []Change) error {
	kept := make([]Change, 0, len(changes))
	for _, change := range changes {
		if strings.TrimSpace(change.Field) == "" || change.Old == change.New {
			continue
		}
		kept = append(kept, change)
	}
	if len(kept) == 0 {
		return nil
	}
	rec.Action = "edited"
	actionID, err := Create(ctx, db, rec)
	if err != nil {
		return err
	}
	return LogHistory(ctx, db, actionID, kept)
}
