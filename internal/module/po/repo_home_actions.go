package po

import (
	"context"
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type homeActionDemandRow struct {
	ID             uint   `gorm:"column:id"`
	Status         string `gorm:"column:status"`
	Deleted        string `gorm:"column:deleted"`
	AssignedTo     string `gorm:"column:assignedTo"`
	DistributedBy  string `gorm:"column:distributedBy"`
	CreatedBy      string `gorm:"column:createdBy"`
	SubmitedBy     string `gorm:"column:submitedBy"`
	SubmitBy       string `gorm:"column:submitBy"`
	QD             string `gorm:"column:QD"`
	RD             string `gorm:"column:RD"`
	BRA            string `gorm:"column:BRA"`
	Accepter       string `gorm:"column:accepter"`
	VeriFier       string `gorm:"column:veriFier"`
	MainDevelopers string `gorm:"column:mainDevelopers"`
	Product        string `gorm:"column:product"`
}

func (r *Repo) homeActionWriter() (*gorm.DB, error) {
	if r == nil || r.writeDB == nil {
		return nil, errors.New("主库未就绪，无法办理首页动作")
	}
	return r.writeDB, nil
}

func (r *Repo) findHomeActionDemand(ctx context.Context, id uint) (*homeActionDemandRow, error) {
	db, err := r.homeActionWriter()
	if err != nil {
		return nil, err
	}
	var row homeActionDemandRow
	err = db.WithContext(ctx).Table("zt_demand").Select("id, status, deleted, assignedTo, distributedBy, createdBy, submitedBy, submitBy, QD, RD, BRA, accepter, veriFier, mainDevelopers, product").Where("id = ?", id).Take(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errHomeActionNotFound
	}
	if err != nil {
		return nil, err
	}
	if row.Deleted != "0" {
		return nil, errHomeActionNotFound
	}
	return &row, nil
}

func hasHomeAccount(value, account string) bool {
	return strings.TrimSpace(value) == strings.TrimSpace(account) && strings.TrimSpace(account) != ""
}

func hasHomeCSV(value, account string) bool {
	if strings.TrimSpace(account) == "" {
		return false
	}
	for _, part := range strings.Split(strings.ReplaceAll(value, " ", ""), ",") {
		if strings.TrimSpace(part) == account {
			return true
		}
	}
	return false
}

func (r *Repo) homeActionAuthorized(row *homeActionDemandRow, account, action string) bool {
	if row == nil || strings.TrimSpace(account) == "" {
		return false
	}
	owner := hasHomeAccount(row.AssignedTo, account) || hasHomeAccount(row.DistributedBy, account)
	submit := hasHomeAccount(row.CreatedBy, account) || hasHomeAccount(row.SubmitedBy, account) || hasHomeCSV(row.SubmitBy, account)
	accept := hasHomeAccount(row.Accepter, account) || hasHomeCSV(row.VeriFier, account)
	switch action {
	case "clarify":
		return owner || hasHomeAccount(row.QD, account) || hasHomeAccount(row.RD, account) || hasHomeAccount(row.BRA, account)
	case "acceptance":
		return accept
	case "deliver":
		return owner
	case "urge":
		return owner || submit || accept || hasHomeAccount(row.QD, account) || hasHomeAccount(row.RD, account) || hasHomeAccount(row.BRA, account) || hasHomeCSV(row.MainDevelopers, account)
	default:
		return false
	}
}

func (r *Repo) updateHomeDemandStatus(ctx context.Context, id uint, expected, next, actor, action, comment string, verify bool) error {
	db, err := r.homeActionWriter()
	if err != nil {
		return err
	}
	now := time.Now()
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var row homeActionDemandRow
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Table("zt_demand").Select("id, status, deleted, product").Where("id = ?", id).Take(&row).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errHomeActionNotFound
			}
			return err
		}
		if row.Deleted != "0" {
			return errHomeActionNotFound
		}
		if row.Status != expected {
			return errHomeActionConflict
		}
		updates := map[string]interface{}{"status": next, "editedBy": actor, "editedDate": now}
		if verify {
			updates["verifyFinish"] = now.Format("2006-01-02")
		}
		if err := tx.Table("zt_demand").Where("id = ? AND status = ? AND deleted = '0'", id, expected).Updates(updates).Error; err != nil {
			return err
		}
		product := ",0,"
		if strings.TrimSpace(row.Product) != "" && row.Product != "0" {
			product = "," + row.Product + ","
		}
		return tx.Create(&demandActionRow{ObjectType: "demand", ObjectID: id, Product: product, Actor: actor, Action: action, Date: now, Comment: comment}).Error
	})
}

func (r *Repo) insertHomeDemandAction(ctx context.Context, id uint, actor, action, comment string) error {
	db, err := r.homeActionWriter()
	if err != nil {
		return err
	}
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var row homeActionDemandRow
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Table("zt_demand").Select("id, status, deleted, product").Where("id = ?", id).Take(&row).Error; err != nil {
			return err
		}
		if row.Deleted == "0" && row.ID > 0 {
			product := ",0,"
			if strings.TrimSpace(row.Product) != "" && row.Product != "0" {
				product = "," + row.Product + ","
			}
			return tx.Create(&demandActionRow{ObjectType: "demand", ObjectID: id, Product: product, Actor: actor, Action: action, Date: time.Now(), Comment: comment}).Error
		}
		return errHomeActionNotFound
	})
}
