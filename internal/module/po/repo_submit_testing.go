package po

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"gorm.io/gorm"

	"workbench/internal/pkg/zentao"
)

var (
	errSubmitTestNotFound   = errors.New("submit test demand not found")
	errSubmitTestForbidden  = errors.New("submit test forbidden")
	errSubmitTestBadRequest = errors.New("submit test request is invalid")
)

type submitTestDemandRow struct {
	ID             uint   `gorm:"column:id"`
	Name           string `gorm:"column:name"`
	Status         string `gorm:"column:status"`
	Stage          string `gorm:"column:stage"`
	BRA            string `gorm:"column:BRA"`
	RD             string `gorm:"column:RD"`
	QD             string `gorm:"column:QD"`
	AssignedTo     string `gorm:"column:assignedTo"`
	ProductName    string `gorm:"column:product_name"`
	EstimateLaunch string `gorm:"column:estimate_launch"`
	BRAName        string `gorm:"column:bra_name"`
	RDName         string `gorm:"column:rd_name"`
	QDName         string `gorm:"column:qd_name"`
	HandlerName    string `gorm:"column:handler_name"`
}

type submitTestStoryRow struct {
	ID            uint   `gorm:"column:id"`
	Title         string `gorm:"column:title"`
	Product       uint   `gorm:"column:product"`
	ProductName   string `gorm:"column:product_name"`
	Branch        uint   `gorm:"column:branch"`
	Execution     uint   `gorm:"column:execution"`
	ExecutionName string `gorm:"column:execution_name"`
	Project       uint   `gorm:"column:project"`
}

func (r *Repo) submitTestReadDB() *gorm.DB {
	if r == nil {
		return nil
	}
	if r.db != nil {
		return r.db
	}
	return r.writeDB
}

func (r *Repo) findSubmitTestDemand(ctx context.Context, db *gorm.DB, demandID uint) (*submitTestDemandRow, error) {
	if db == nil || demandID == 0 {
		return nil, errSubmitTestNotFound
	}
	var row submitTestDemandRow
	if err := db.WithContext(ctx).Table("zt_demand AS d").
		Select(`d.id, d.name, d.status, d.stage, d.BRA, d.RD, d.QD, d.assignedTo,
			COALESCE(p.name, d.mainSystem, '') AS product_name,
			COALESCE(DATE_FORMAT(d.estimateLaunch, '%Y-%m-%d'), '') AS estimate_launch,
			COALESCE(bra.realname, d.BRA, '') AS bra_name,
			COALESCE(rd.realname, d.RD, '') AS rd_name,
			COALESCE(qd.realname, d.QD, '') AS qd_name,
			COALESCE(assignee.realname, d.assignedTo, '') AS handler_name`).
		Joins("LEFT JOIN zt_product p ON p.id = CAST(NULLIF(d.mainSystem, '') AS UNSIGNED) AND p.deleted = '0'").
		Joins("LEFT JOIN zt_user bra ON bra.account = d.BRA").
		Joins("LEFT JOIN zt_user rd ON rd.account = d.RD").
		Joins("LEFT JOIN zt_user qd ON qd.account = d.QD").
		Joins("LEFT JOIN zt_user assignee ON assignee.account = d.assignedTo").
		Where("d.id = ? AND d.deleted = '0'", demandID).Limit(1).Scan(&row).Error; err != nil {
		return nil, err
	}
	if row.ID == 0 {
		return nil, errSubmitTestNotFound
	}
	return &row, nil
}

func (r *Repo) findSubmitTestStories(ctx context.Context, db *gorm.DB, demandID uint) ([]submitTestStoryRow, error) {
	if db == nil || demandID == 0 {
		return nil, errSubmitTestBadRequest
	}
	var rows []submitTestStoryRow
	err := db.WithContext(ctx).Table("zt_story AS s").
		Select(`s.id, s.title, s.product, s.branch,
			COALESCE((SELECT t.execution FROM zt_task t WHERE t.story = s.id AND t.deleted = '0' ORDER BY t.id LIMIT 1), 0) AS execution,
			COALESCE((SELECT t.project FROM zt_task t WHERE t.story = s.id AND t.deleted = '0' ORDER BY t.id LIMIT 1), 0) AS project,
			COALESCE(p.name, CONCAT('产品#', s.product)) AS product_name,
			COALESCE((SELECT e.name FROM zt_project e WHERE e.id = (SELECT t.execution FROM zt_task t WHERE t.story = s.id AND t.deleted = '0' ORDER BY t.id LIMIT 1) AND e.deleted = '0'), '') AS execution_name`).
		Joins("LEFT JOIN zt_product p ON p.id = s.product AND p.deleted = '0'").
		Where(`s.deleted = '0' AND s.status <> 'closed' AND (s.fromDemand = ? OR s.fromDemand IN (SELECT d.id FROM zt_demand d WHERE d.parent = ? AND d.deleted = '0'))`, demandID, demandID).
		Order("s.product ASC, s.branch ASC, s.id ASC").Scan(&rows).Error
	return rows, err
}

func submitTestUnitKey(product, branch, execution uint) string {
	return fmt.Sprintf("%d/%d/%d", product, branch, execution)
}

func groupSubmitTestUnits(stories []submitTestStoryRow) []SubmitTestUnit {
	groups := make(map[string]*SubmitTestUnit)
	for _, story := range stories {
		key := submitTestUnitKey(story.Product, story.Branch, story.Execution)
		unit := groups[key]
		if unit == nil {
			unit = &SubmitTestUnit{
				Product: story.Product, ProductName: story.ProductName, Branch: story.Branch,
				Execution: story.Execution, ExecutionName: story.ExecutionName, Project: story.Project,
			}
			groups[key] = unit
		}
		unit.Stories = append(unit.Stories, SubmitTestStory{ID: story.ID, Title: story.Title})
	}
	units := make([]SubmitTestUnit, 0, len(groups))
	for _, unit := range groups {
		units = append(units, *unit)
	}
	sort.Slice(units, func(i, j int) bool {
		return submitTestUnitKey(units[i].Product, units[i].Branch, units[i].Execution) < submitTestUnitKey(units[j].Product, units[j].Branch, units[j].Execution)
	})
	return units
}

func (r *Repo) SubmitTestPage(ctx context.Context, demandID uint) (*SubmitTestPageData, error) {
	db := r.submitTestReadDB()
	demand, err := r.findSubmitTestDemand(ctx, db, demandID)
	if err != nil {
		return nil, err
	}
	stories, err := r.findSubmitTestStories(ctx, db, demandID)
	if err != nil {
		return nil, err
	}
	ready, err := r.allTasksCompletedTx(ctx, db, demandID)
	if err != nil {
		return nil, err
	}
	_, stageLabel := mapValueStage(demand.Stage, demand.Status)
	_, statusLabel := mapValueStage("", demand.Status)
	page := &SubmitTestPageData{DemandID: demand.ID, Title: demand.Name, Stage: stageLabel, RawStatus: statusLabel,
		MainSystemName: demand.ProductName, EstimateLaunch: demand.EstimateLaunch, BRAName: demand.BRAName,
		RDName: demand.RDName, QDName: demand.QDName, HandlerName: demand.HandlerName,
		Units: groupSubmitTestUnits(stories), ZentaoURL: zentao.DemandViewURL(demand.ID),
		SubmitUnavailable: "提测创建与状态流转请在禅道原生页面完成"}
	switch {
	case demand.Status != "developing":
		page.Reason = "当前需求不在研发中，暂不能进入禅道提测流程"
	case len(page.Units) == 0:
		page.Reason = "当前业务需求没有可提测的研发需求"
	case !ready:
		page.Reason = "研发任务尚未全部完成，或计划开发完成日未到"
	}
	page.Eligible = page.Reason == ""
	return page, nil
}

func (r *Repo) allTasksCompletedTx(ctx context.Context, tx *gorm.DB, demandID uint) (bool, error) {
	type countRow struct {
		Total  int    `gorm:"column:total"`
		Done   int    `gorm:"column:done"`
		Finish string `gorm:"column:finish"`
	}
	var row countRow
	err := tx.WithContext(ctx).Raw(`SELECT COUNT(*) AS total,
		SUM(CASE WHEN t.status IN ('done', 'closed') THEN 1 ELSE 0 END) AS done
		, MAX(NULLIF(s.developFinish, '0000-00-00')) AS finish
		FROM zt_story s JOIN zt_task t ON t.story = s.id AND t.deleted = '0'
		WHERE s.deleted = '0' AND s.status <> 'closed'
		AND (s.fromDemand = ? OR s.fromDemand IN (SELECT d.id FROM zt_demand d WHERE d.parent = ? AND d.deleted = '0'))`, demandID, demandID).Scan(&row).Error
	if err != nil {
		return false, err
	}
	return row.Total > 0 && row.Total == row.Done && row.Finish != "" && row.Finish <= time.Now().Format("2006-01-02"), nil
}
