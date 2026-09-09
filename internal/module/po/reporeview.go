// =============================================================================
// 文件: internal/module/po/reporeview.go
// 模块: PO 工作台
// 类型: action
// 职责: 业需评审写库（zt_demand / zt_demandreview / zt_action）。只走主库 writeDB。
// 依赖: 无
// =============================================================================

package po

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	errDemandNotReviewable = errors.New("该需求不是待评审状态")
	errAlreadyReviewed     = errors.New("您已评审过该需求")
)

// demandReviewRow 对应禅道 zt_demand 评审时用到的列（不必映射整张宽表）。
type demandReviewRow struct {
	ID          int64  `gorm:"column:id"`
	Status      string `gorm:"column:status"`
	Deleted     string `gorm:"column:deleted"`
	CreatedBy   string `gorm:"column:createdBy"`
	AssignedTo  string `gorm:"column:assignedTo"`
	ReviewedBy  string `gorm:"column:reviewedBy"`
	Mailto      string `gorm:"column:mailto"`
	IsNeedFocus string `gorm:"column:isNeedFocus"`
	Product     string `gorm:"column:product"`
}

func (demandReviewRow) TableName() string { return "zt_demand" }

// demandReviewerRow 对应 zt_demandreview：每个评审人一行结果。
type demandReviewerRow struct {
	Demand     int64      `gorm:"column:demand"`
	Version    int        `gorm:"column:version"`
	Reviewer   string     `gorm:"column:reviewer"`
	Result     string     `gorm:"column:result"`
	ReviewDate *time.Time `gorm:"column:reviewDate"`
}

func (demandReviewerRow) TableName() string { return "zt_demandreview" }

// demandActionRow 对应 zt_action，给禅道详情页「历史记录」用。
type demandActionRow struct {
	ID         uint      `gorm:"column:id;primaryKey;autoIncrement"`
	ObjectType string    `gorm:"column:objectType"`
	ObjectID   uint      `gorm:"column:objectID"`
	Product    string    `gorm:"column:product"`
	Project    uint      `gorm:"column:project"`
	Execution  uint      `gorm:"column:execution"`
	Actor      string    `gorm:"column:actor"`
	Action     string    `gorm:"column:action"`
	Date       time.Time `gorm:"column:date"`
	Comment    string    `gorm:"column:comment"`
	Extra      string    `gorm:"column:extra"`
}

func (demandActionRow) TableName() string { return "zt_action" }

func (r *Repo) writer() (*gorm.DB, error) {
	if r == nil || r.writeDB == nil {
		return nil, errors.New("主库未就绪，无法评审")
	}
	return r.writeDB, nil
}

// FindDemandForReview 按 ID 读业需（主库，避免备库延迟看到旧状态）。
func (r *Repo) FindDemandForReview(ctx context.Context, id int64) (*demandReviewRow, error) {
	db, err := r.writer()
	if err != nil {
		return nil, err
	}
	var row demandReviewRow
	err = db.WithContext(ctx).
		Select("id, status, deleted, createdBy, assignedTo, reviewedBy, mailto, isNeedFocus, product").
		Where("id = ?", id).
		Take(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("load demand %d: %w", id, err)
	}
	return &row, nil
}

// FindDemandReviewerResult 查当前账号在 zt_demandreview 中的一行。found=false 表示不是业务评审人。
func (r *Repo) FindDemandReviewerResult(ctx context.Context, demandID int64, account string) (result string, found bool, err error) {
	db, err := r.writer()
	if err != nil {
		return "", false, err
	}
	var row demandReviewerRow
	err = db.WithContext(ctx).
		Select("result").
		Where("demand = ? AND reviewer = ?", demandID, account).
		Take(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("load demandreview %d %s: %w", demandID, account, err)
	}
	return row.Result, true, nil
}

// FindPendingReviewDemandIDs 当前账号尚未给出结果的业务评审需求（用于列表「评审」按钮）。
func (r *Repo) FindPendingReviewDemandIDs(ctx context.Context, account string, demandIDs []int) (map[int]struct{}, error) {
	out := map[int]struct{}{}
	if r == nil || r.db == nil || strings.TrimSpace(account) == "" || len(demandIDs) == 0 {
		return out, nil
	}
	var ids []int
	err := r.db.WithContext(ctx).
		Model(&demandReviewerRow{}).
		Where("demand IN ? AND reviewer = ? AND (result = ? OR result IS NULL)", demandIDs, account, "").
		Pluck("demand", &ids).Error
	if err != nil {
		return nil, err
	}
	for _, id := range ids {
		out[id] = struct{}{}
	}
	return out, nil
}

// countUnpassedReviewers 统计尚未给出 pass 的评审人。
// result 为 NULL / 空串 / 非 pass 都算未通过；MySQL 里 NULL <> 'pass' 不是 true，必须单独写出 IS NULL。
func countUnpassedReviewers(tx *gorm.DB, demandID int64) (int64, error) {
	var n int64
	err := tx.Model(&demandReviewerRow{}).
		Where("demand = ? AND reviewer <> ? AND (result IS NULL OR result <> ?)", demandID, "", "pass").
		Count(&n).Error
	return n, err
}

// SaveDemandReview 一次事务写完评审，失败全部回滚。
func (r *Repo) SaveDemandReview(ctx context.Context, in saveDemandReviewIn) error {
	db, err := r.writer()
	if err != nil {
		return err
	}
	now := time.Now()
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 每次从干净 session 起查，避免上一条 SQL 的 Where/Select 串到下一句。
		q := func() *gorm.DB { return tx.Session(&gorm.Session{NewDB: true}) }

		var locked demandReviewRow
		if err := q().Clauses(clause.Locking{Strength: "UPDATE"}).
			Select("id, status, deleted, createdBy, assignedTo, reviewedBy, mailto, isNeedFocus, product").
			Where("id = ?", in.DemandID).
			Take(&locked).Error; err != nil {
			return fmt.Errorf("lock demand %d: %w", in.DemandID, err)
		}
		if locked.Deleted != "0" || strings.TrimSpace(locked.Status) != "wait" {
			return errDemandNotReviewable
		}

		var mine demandReviewerRow
		if err := q().Model(&demandReviewerRow{}).
			Select("result").
			Where("demand = ? AND reviewer = ?", in.DemandID, in.Account).
			Take(&mine).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errDemandNotReviewable
			}
			return fmt.Errorf("load demandreview %d %s: %w", in.DemandID, in.Account, err)
		}
		if strings.TrimSpace(mine.Result) != "" {
			return errAlreadyReviewed
		}

		// 1) 当前登录人在 zt_demandreview 里的那一行：写下结果和时间
		if err := q().Table("zt_demandreview").
			Where("demand = ? AND reviewer = ?", in.DemandID, in.Account).
			Updates(map[string]any{
				"result":     in.Result,
				"reviewDate": now,
			}).Error; err != nil {
			return fmt.Errorf("update demandreview: %w", err)
		}

		newStatus := in.NewStatus
		statusAction := in.StatusAction
		assignBackTo := in.AssignBackTo
		if in.Result == "pass" {
			left, countErr := countUnpassedReviewers(q(), in.DemandID)
			if countErr != nil {
				return countErr
			}
			if left == 0 {
				newStatus = "active"
				statusAction = "reviewpassed"
			} else {
				newStatus = ""
				statusAction = ""
			}
		}
		if newStatus == "refuse" && assignBackTo == "" {
			assignBackTo = strings.TrimSpace(locked.CreatedBy)
		}

		updates := map[string]any{
			"reviewedDate": now,
			"reviewedBy":   in.ReviewedBy,
			"isNeedFocus":  in.IsNeedFocus,
			"editedBy":     in.Account,
			"editedDate":   now,
		}
		if in.Mailto != nil {
			updates["mailto"] = *in.Mailto
		}
		if newStatus != "" {
			updates["status"] = newStatus
			if newStatus == "refuse" && assignBackTo != "" {
				updates["assignedTo"] = assignBackTo
			}
		}
		if err := q().Table("zt_demand").Where("id = ?", in.DemandID).Updates(updates).Error; err != nil {
			return fmt.Errorf("update demand: %w", err)
		}

		productField := ",0,"
		if p := strings.TrimSpace(in.Product); p != "" && p != "0" {
			productField = "," + p + ","
		}
		// extra 存 pass/refuse，禅道历史里用来显示「确认通过/拒绝」
		if err := q().Create(&demandActionRow{
			ObjectType: "demand",
			ObjectID:   uint(in.DemandID),
			Product:    productField,
			Actor:      in.Account,
			Action:     "reviewed",
			Date:       now,
			Comment:    in.Comment,
			Extra:      in.Result,
		}).Error; err != nil {
			return fmt.Errorf("insert action reviewed: %w", err)
		}

		if statusAction != "" {
			if err := q().Create(&demandActionRow{
				ObjectType: "demand",
				ObjectID:   uint(in.DemandID),
				Product:    productField,
				Actor:      in.Account,
				Action:     statusAction,
				Date:       now,
			}).Error; err != nil {
				return fmt.Errorf("insert action %s: %w", statusAction, err)
			}
		}
		return nil
	})
}

type saveDemandReviewIn struct {
	DemandID     int64
	Account      string
	Result       string
	IsNeedFocus  string
	Mailto       *string
	ReviewedBy   string
	Comment      string
	Product      string
	NewStatus    string // 空=暂不改状态（还有其他人没评完）
	StatusAction string // reviewpassed / reviewrejected
	AssignBackTo string // 拒绝时指派回创建人
}
