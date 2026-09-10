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
	Name           string `gorm:"column:name"`
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
	Originator     string `gorm:"column:originator"`
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
	err = db.WithContext(ctx).Table("zt_demand").Select("id, name, status, deleted, assignedTo, distributedBy, createdBy, submitedBy, submitBy, QD, RD, BRA, originator, accepter, veriFier, mainDevelopers, product").Where("id = ?", id).Take(&row).Error
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
	case "acceptance":
		return accept
	case "deliver":
		return owner || hasHomeAccount(row.BRA, account)
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

// insertAcceptanceUrge records an immutable reminder action and in-app notices.
// It deliberately never updates zt_demand.status.
func (r *Repo) insertAcceptanceUrge(ctx context.Context, row *homeActionDemandRow, actor string, recipients []string, comment string, within time.Duration) (bool, error) {
	db, err := r.homeActionWriter()
	if err != nil {
		return false, err
	}
	if row == nil || row.ID == 0 {
		return false, errHomeActionNotFound
	}
	duplicate := false
	err = db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var locked homeActionDemandRow
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Table("zt_demand").Select("id, status, deleted, product").Where("id = ?", row.ID).Take(&locked).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errHomeActionNotFound
			}
			return err
		}
		if locked.Deleted != "0" {
			return errHomeActionNotFound
		}
		if locked.Status != "testing" && locked.Status != "waitacceptance" {
			return errHomeActionConflict
		}
		var existing int64
		if err := tx.Raw(`SELECT COUNT(*) FROM zt_action a INNER JOIN zt_notify n ON n.action = a.id WHERE a.objectType='demand' AND a.objectID=? AND a.action='reminded' AND a.extra='urge:accept' AND n.toList IN ? AND a.date >= ?`, row.ID, recipients, time.Now().Add(-within)).Scan(&existing).Error; err != nil {
			return err
		}
		if acceptanceUrgeIsDuplicate(existing, recipients) {
			duplicate = true
			return errUrgeDuplicate
		}
		product := ",0,"
		if locked.Product != "" && locked.Product != "0" {
			product = "," + locked.Product + ","
		}
		action := demandActionRow{ObjectType: "demand", ObjectID: row.ID, Product: product, Actor: actor, Action: "reminded", Date: time.Now(), Comment: comment, Extra: "urge:accept"}
		if err := tx.Create(&action).Error; err != nil {
			return err
		}
		for _, recipient := range recipients {
			if recipient == actor {
				continue
			}
			if err := tx.Exec(`INSERT INTO zt_notify (objectType, objectID, action, ccList, data, createdBy, createdDate, status, failReason, toList, subject) VALUES ('demand', ?, ?, '', ?, ?, NOW(), 'wait', '', ?, '催办')`, row.ID, action.ID, comment, actor, recipient).Error; err != nil {
				return err
			}
		}
		return nil
	})
	return duplicate, err
}

func acceptanceUrgeIsDuplicate(existing int64, recipients []string) bool {
	return len(recipients) > 0 && existing >= int64(len(recipients))
}

func (r *Repo) homeUserLabels(ctx context.Context, accounts []string) (map[string]string, error) {
	out := map[string]string{}
	if r == nil || len(accounts) == 0 {
		return out, nil
	}
	db, err := r.reader()
	if err != nil || db == nil {
		return out, nil
	}
	clean := make([]string, 0, len(accounts))
	seen := map[string]bool{}
	for _, a := range accounts {
		a = strings.TrimSpace(a)
		if a == "" || seen[a] {
			continue
		}
		seen[a] = true
		clean = append(clean, a)
	}
	if len(clean) == 0 {
		return out, nil
	}
	type row struct {
		Account  string `gorm:"column:account"`
		Realname string `gorm:"column:realname"`
	}
	var rows []row
	if err := db.WithContext(ctx).Table("zt_user").Select("account, realname").Where("deleted = '0' AND account IN ?", clean).Find(&rows).Error; err != nil {
		return out, err
	}
	for _, item := range rows {
		label := strings.TrimSpace(item.Realname)
		if label == "" {
			label = item.Account
		} else if !strings.HasSuffix(label, "("+item.Account+")") {
			label = label + "(" + item.Account + ")"
		}
		out[item.Account] = label
	}
	for _, a := range clean {
		if _, ok := out[a]; !ok {
			out[a] = a
		}
	}
	return out, nil
}
