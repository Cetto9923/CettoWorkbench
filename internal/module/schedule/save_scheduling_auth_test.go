package schedule

import (
	"context"
	"database/sql/driver"
	"errors"
	"strings"
	"testing"

	"workbench/internal/model"

	"github.com/DATA-DOG/go-sqlmock"
)

func mockDemandScopeQueries(mock sqlmock.Sqlmock, demandID uint, childDemands []ZtDemand, stories []ZtStory, windowID uint) {
	// 1. FindChildDemandsByParents
	childQuery := mock.ExpectQuery(`(?s)SELECT\s+id, name, pri, status, assignedTo, mainSystem, teamGroup,\s+BRA, QD, RD, createdBy, pool, parent, hang, category, estimateLaunch\s+FROM zt_demand\s+WHERE deleted = '0'\s+AND parent IN \(\?\)\s+ORDER BY parent ASC, id ASC`).
		WithArgs(demandID)
	childRows := sqlmock.NewRows([]string{"id", "name", "pri", "status", "assignedTo", "mainSystem", "teamGroup", "BRA", "QD", "RD", "createdBy", "pool", "parent", "hang", "category", "estimateLaunch"})
	demandIDs := []driver.Value{demandID}
	for _, cd := range childDemands {
		childRows.AddRow(cd.ID, cd.Name, cd.Pri, cd.Status, cd.AssignedTo, cd.MainSystem, 0, cd.BRA, cd.QD, cd.RD, cd.CreatedBy, 0, cd.Parent, "0", "", "")
		demandIDs = append(demandIDs, cd.ID)
	}
	childQuery.WillReturnRows(childRows)

	// 2. FindStoriesByDemands
	storyQuery := mock.ExpectQuery(`(?s)SELECT\s+id,\s+title,\s+pri,\s+product,\s+plan,\s+stage,\s+status,\s+fromDemand,\s+sourceType,\s+parent,\s+CAST\(IFNULL\(NULLIF\(isMainSystemAssociation, ''\), '0'\) AS SIGNED\) AS isMainSystemAssociation,\s+assignedTo\s+FROM zt_story\s+WHERE fromDemand IN \((?:\?|,\s*|\?)+\)\s+AND sourceType = 'demandpool'\s+AND type = 'story'\s+AND deleted = '0'\s+ORDER BY fromDemand ASC, isMainSystemAssociation DESC, id ASC`).
		WithArgs(demandIDs...)
	storyRows := sqlmock.NewRows([]string{"id", "title", "pri", "product", "plan", "stage", "status", "fromDemand", "sourceType", "parent", "isMainSystemAssociation", "assignedTo"})
	storyIDs := make([]driver.Value, 0, len(stories))
	for _, s := range stories {
		storyRows.AddRow(s.ID, s.Title, s.Pri, s.Product, s.Plan, s.Stage, s.Status, s.FromDemand, s.SourceType, s.Parent, s.IsMainSystemAssociation, s.AssignedTo)
		storyIDs = append(storyIDs, s.ID)
	}
	storyQuery.WillReturnRows(storyRows)

	// 3. FindDemandWindowMappings
	dwQuery := mock.ExpectQuery(`(?s)SELECT dw\.demand, dw\.versionWindow AS windowID, vw\.name AS windowName\s+FROM zt_demandwindow dw\s+INNER JOIN zt_versionwindow vw ON vw\.id = dw\.versionWindow AND vw\.deletedAt IS NULL\s+WHERE dw\.demand IN \((?:\?|,\s*|\?)+\)\s+AND dw\.story = 0\s+AND dw\.deletedAt IS NULL\s+AND dw\.versionWindow > 0\s+ORDER BY dw\.demand ASC, dw\.updatedDate DESC, dw\.id DESC`).
		WithArgs(demandIDs...)
	dwRows := sqlmock.NewRows([]string{"demand", "windowID", "windowName"})
	if windowID > 0 {
		dwRows.AddRow(demandID, windowID, "Window 1")
	}
	dwQuery.WillReturnRows(dwRows)

	// 4. FindStoryWindowMappings (only if stories non-empty)
	if len(storyIDs) > 0 {
		swQuery := mock.ExpectQuery(`(?s)SELECT ps\.story, vw\.id AS windowID, vw\.name AS windowName, vw\.teamgroup AS teamgroupID\s+FROM zt_planstory ps\s+INNER JOIN zt_versionwindowproduct vwp\s+ON vwp\.plan = ps\.plan AND vwp\.deletedAt IS NULL\s+INNER JOIN zt_versionwindow vw\s+ON vw\.id = vwp\.versionWindow AND vw\.deletedAt IS NULL\s+WHERE ps\.story IN \((?:\?|,\s*|\?)+\)\s+ORDER BY ps\.story ASC, vw\.id ASC`).
			WithArgs(storyIDs...)
		swQuery.WillReturnRows(sqlmock.NewRows([]string{"story", "windowID", "windowName", "teamgroupID"}))
	}
}

// 1. DeleteForeignStoryRejected:
// Request asks to delete Story 999, but Story 999 does not belong to the allowed demand scope.
// Must reject with taskMutationForbidden before any mutation or transaction.
func TestSaveScheduling_DeleteForeignStoryRejected(t *testing.T) {
	db, mock := setupMockDB(t)
	repo := NewRepo(db)
	svc := NewService(repo, nil)

	ctx := context.Background()
	actor := &model.User{Account: "alice"}
	demandID := uint(10)

	existingStories := []ZtStory{
		{ID: 101, Title: "Story 101", Product: 100, FromDemand: 10, SourceType: "demandpool", Status: "active"},
	}
	mockDemandScopeQueries(mock, demandID, nil, existingStories, 1)

	req := &SaveSchedulingReq{
		WindowID: 1,
		Stories: []SaveSchedulingStory{
			{Action: "delete", ID: 999}, // Foreign story
		},
	}

	err := svc.SaveScheduling(ctx, actor, demandID, req)
	if err == nil {
		t.Fatal("expected error for foreign story delete, got nil")
	}
	var mutationErr *TaskMutationError
	if !errors.As(err, &mutationErr) || mutationErr.Code != taskMutationForbidden {
		t.Fatalf("expected taskMutationForbidden, got %v", err)
	}
	if !strings.Contains(err.Error(), "999") {
		t.Fatalf("expected error mentioning story 999, got %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet mock expectations: %v", err)
	}
}

// 2. EditForeignStoryRejected:
// Client submits an edit for Story 999 with a ProductID the user has access to.
// Must still be rejected because Story 999 does not belong to the current demand scope.
func TestSaveScheduling_EditForeignStoryRejected(t *testing.T) {
	db, mock := setupMockDB(t)
	repo := NewRepo(db)
	svc := NewService(repo, nil)

	ctx := context.Background()
	actor := &model.User{Account: "alice"}
	demandID := uint(10)

	existingStories := []ZtStory{
		{ID: 101, Title: "Story 101", Product: 100, FromDemand: 10, SourceType: "demandpool", Status: "active"},
	}
	mockDemandScopeQueries(mock, demandID, nil, existingStories, 1)

	req := &SaveSchedulingReq{
		WindowID: 1,
		Stories: []SaveSchedulingStory{
			{Action: "edit", ID: 999, ProductID: 100, Title: "Foreign Story Edit"},
		},
	}

	err := svc.SaveScheduling(ctx, actor, demandID, req)
	if err == nil {
		t.Fatal("expected error for foreign story edit, got nil")
	}
	var mutationErr *TaskMutationError
	if !errors.As(err, &mutationErr) || mutationErr.Code != taskMutationForbidden {
		t.Fatalf("expected taskMutationForbidden, got %v", err)
	}
	if !strings.Contains(err.Error(), "999") {
		t.Fatalf("expected error mentioning story 999, got %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet mock expectations: %v", err)
	}
}

// 3. DeleteUnauthorizedCurrentProductRejected:
// Story 101 belongs to Demand 10, but its current database product is 200, which alice has no access to.
// Delete must be rejected via product access precheck.
func TestSaveScheduling_DeleteUnauthorizedCurrentProductRejected(t *testing.T) {
	db, mock := setupMockDB(t)
	repo := NewRepo(db)
	svc := NewService(repo, nil)

	ctx := context.Background()
	actor := &model.User{Account: "alice"}
	demandID := uint(10)

	existingStories := []ZtStory{
		{ID: 101, Title: "Story 101", Product: 200, FromDemand: 10, SourceType: "demandpool", Status: "active"},
	}
	mockDemandScopeQueries(mock, demandID, nil, existingStories, 1)

	mock.ExpectQuery(`SELECT mainSystem FROM zt_demand WHERE id = \? AND deleted = '0' LIMIT 1`).
		WithArgs(demandID).
		WillReturnRows(sqlmock.NewRows([]string{"mainSystem"}).AddRow("100"))

	mock.ExpectQuery(`(?s)SELECT id, name, code, status, PO, QD, RD, createdBy, whitelist\s+FROM zt_product`).
		WithArgs("alice", "alice", "alice", "alice", "alice").
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "code", "status", "PO", "QD", "RD", "createdBy", "whitelist"}).
			AddRow(100, "Authorized Product", "AUTH", "normal", "alice", "alice", "alice", "alice", ""))

	mock.ExpectQuery(`(?s)SELECT id, name\s+FROM zt_product\s+WHERE id IN \(\?\)\s+AND deleted = '0'`).
		WithArgs(200).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).AddRow(200, "Restricted Product"))

	req := &SaveSchedulingReq{
		WindowID: 1,
		Stories: []SaveSchedulingStory{
			{Action: "delete", ID: 101},
		},
	}

	err := svc.SaveScheduling(ctx, actor, demandID, req)
	if err == nil {
		t.Fatal("expected product access notice error, got nil")
	}
	var notice *ProductAccessNoticeError
	if !errors.As(err, &notice) {
		t.Fatalf("expected ProductAccessNoticeError, got %v", err)
	}
	if len(notice.Products) != 1 || notice.Products[0].ID != 200 {
		t.Fatalf("expected notice for product 200, got: %+v", notice.Products)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet mock expectations: %v", err)
	}
}

// 4. EditAuthorizedExistingStoryStillWorks:
// Story 101 belongs to Demand 10 with Product 100.
// Alice has access to Product 100.
// Edit should proceed through transaction, updating story, action, plan, and demand.
func TestSaveScheduling_EditAuthorizedExistingStoryStillWorks(t *testing.T) {
	db, mock := setupMockDB(t)
	repo := NewRepo(db)
	svc := NewService(repo, nil)

	ctx := context.Background()
	actor := &model.User{Account: "alice"}
	demandID := uint(10)

	existingStories := []ZtStory{
		{ID: 101, Title: "Story 101", Product: 100, FromDemand: 10, SourceType: "demandpool", Status: "active"},
	}
	mockDemandScopeQueries(mock, demandID, nil, existingStories, 1)

	mock.ExpectQuery(`SELECT mainSystem FROM zt_demand WHERE id = \? AND deleted = '0' LIMIT 1`).
		WithArgs(demandID).
		WillReturnRows(sqlmock.NewRows([]string{"mainSystem"}).AddRow("100"))

	mock.ExpectQuery(`(?s)SELECT id, name, code, status, PO, QD, RD, createdBy, whitelist\s+FROM zt_product`).
		WithArgs("alice", "alice", "alice", "alice", "alice").
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "code", "status", "PO", "QD", "RD", "createdBy", "whitelist"}).
			AddRow(100, "Authorized Product", "AUTH", "normal", "alice", "alice", "alice", "alice", ""))

	// Transaction begin
	mock.ExpectBegin()

	// applySchedulingStory: edit
	mock.ExpectExec(`UPDATE `+"`zt_story`"+` SET .* WHERE id = \?`).
		WithArgs("", "alice", sqlmock.AnyArg(), 100, "Updated Title", 101).
		WillReturnResult(sqlmock.NewResult(1, 1))

	mock.ExpectExec(`INSERT INTO ` + "`zt_action`" + ` .*`).
		WillReturnResult(sqlmock.NewResult(1, 1))

	// resolvePlanForProduct
	mock.ExpectQuery(`(?s)SELECT \* FROM `+"`zt_versionwindowproduct`"+` WHERE \(?versionWindow = \? AND product = \?\)? AND `+"`zt_versionwindowproduct`"+`\.`+"`deletedAt`"+` IS NULL ORDER BY `+"`zt_versionwindowproduct`"+`\.`+"`id`"+` LIMIT \?`).
		WithArgs(1, 100, 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "versionWindow", "product", "plan"}).
			AddRow(1, 1, 100, 50))

	// RemoveStoryFromOtherPlans -> ListOtherPlansOfStory
	mock.ExpectQuery(`SELECT plan FROM zt_planstory WHERE story = \? AND plan <> \?`).
		WithArgs(101, 50).
		WillReturnRows(sqlmock.NewRows([]string{"plan"}))

	// LinkStoryToPlan -> PlanStoryExists
	mock.ExpectQuery(`SELECT 1 AS ok FROM zt_planstory WHERE plan = \? AND story = \? LIMIT 1`).
		WithArgs(50, 101).
		WillReturnRows(sqlmock.NewRows([]string{"ok"}).AddRow(1))

	// EnsurePlanStoryRelation
	mock.ExpectExec(`INSERT IGNORE INTO zt_planstory .*`).
		WithArgs(50, 101).
		WillReturnResult(sqlmock.NewResult(1, 1))

	// applySchedulingTasks -> ValidateStoryForTaskMutation
	mock.ExpectQuery(`SELECT id, deleted FROM zt_story WHERE id = \? FOR UPDATE`).
		WithArgs(101).
		WillReturnRows(sqlmock.NewRows([]string{"id", "deleted"}).AddRow(101, "0"))

	// UpdateDemandScheduling
	mock.ExpectExec(`UPDATE `+"`zt_demand`"+` SET .* WHERE id = \?`).
		WithArgs("", "", nil, "", nil, "alice", sqlmock.AnyArg(), nil, demandID).
		WillReturnResult(sqlmock.NewResult(1, 1))

	// SaveDemandLevelWindow
	mock.ExpectExec(`DELETE FROM ` + "`zt_demandwindow`" + ` WHERE demand = \? AND story = 0`).
		WithArgs(demandID).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(`INSERT INTO ` + "`zt_demandwindow`" + ` .*`).
		WillReturnResult(sqlmock.NewResult(1, 1))

	mock.ExpectCommit()

	req := &SaveSchedulingReq{
		WindowID: 1,
		Stories: []SaveSchedulingStory{
			{Action: "edit", ID: 101, ProductID: 100, Title: "Updated Title"},
		},
	}

	err := svc.SaveScheduling(ctx, actor, demandID, req)
	if err != nil {
		t.Fatalf("expected success for authorized edit, got: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet mock expectations: %v", err)
	}
}

// 5. DeleteAuthorizedExistingStoryStillWorks:
// Story 101 belongs to Demand 10 with Product 100.
// Alice has access to Product 100.
// Delete should close the story and create the closed action.
func TestSaveScheduling_DeleteAuthorizedExistingStoryStillWorks(t *testing.T) {
	db, mock := setupMockDB(t)
	repo := NewRepo(db)
	svc := NewService(repo, nil)

	ctx := context.Background()
	actor := &model.User{Account: "alice"}
	demandID := uint(10)

	existingStories := []ZtStory{
		{ID: 101, Title: "Story 101", Product: 100, FromDemand: 10, SourceType: "demandpool", Status: "active"},
	}
	mockDemandScopeQueries(mock, demandID, nil, existingStories, 1)

	mock.ExpectQuery(`SELECT mainSystem FROM zt_demand WHERE id = \? AND deleted = '0' LIMIT 1`).
		WithArgs(demandID).
		WillReturnRows(sqlmock.NewRows([]string{"mainSystem"}).AddRow("100"))

	mock.ExpectQuery(`(?s)SELECT id, name, code, status, PO, QD, RD, createdBy, whitelist\s+FROM zt_product`).
		WithArgs("alice", "alice", "alice", "alice", "alice").
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "code", "status", "PO", "QD", "RD", "createdBy", "whitelist"}).
			AddRow(100, "Authorized Product", "AUTH", "normal", "alice", "alice", "alice", "alice", ""))

	// Transaction begin
	mock.ExpectBegin()

	// applySchedulingStory: delete
	mock.ExpectExec(`UPDATE `+"`zt_story`"+` SET .* WHERE id = \?`).
		WithArgs("alice", sqlmock.AnyArg(), "done", "closed", 101).
		WillReturnResult(sqlmock.NewResult(1, 1))

	mock.ExpectExec(`INSERT INTO ` + "`zt_action`" + ` .*`).
		WillReturnResult(sqlmock.NewResult(1, 1))

	// UpdateDemandScheduling
	mock.ExpectExec(`UPDATE `+"`zt_demand`"+` SET .* WHERE id = \?`).
		WithArgs("", "", nil, "", nil, "alice", sqlmock.AnyArg(), nil, demandID).
		WillReturnResult(sqlmock.NewResult(1, 1))

	// SaveDemandLevelWindow
	mock.ExpectExec(`DELETE FROM ` + "`zt_demandwindow`" + ` WHERE demand = \? AND story = 0`).
		WithArgs(demandID).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(`INSERT INTO ` + "`zt_demandwindow`" + ` .*`).
		WillReturnResult(sqlmock.NewResult(1, 1))

	mock.ExpectCommit()

	req := &SaveSchedulingReq{
		WindowID: 1,
		Stories: []SaveSchedulingStory{
			{Action: "delete", ID: 101},
		},
	}

	err := svc.SaveScheduling(ctx, actor, demandID, req)
	if err != nil {
		t.Fatalf("expected success for authorized delete, got: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet mock expectations: %v", err)
	}
}

// 6. ChildDemandExistingStory:
// Story 201 belongs to Child Demand 20 (where Parent == 10).
// Deleting Story 201 via Demand 10's SaveScheduling must succeed, proving that
// child demand stories are correctly included in the allowed scope.
func TestSaveScheduling_ChildDemandExistingStory(t *testing.T) {
	db, mock := setupMockDB(t)
	repo := NewRepo(db)
	svc := NewService(repo, nil)

	ctx := context.Background()
	actor := &model.User{Account: "alice"}
	demandID := uint(10)

	childDemands := []ZtDemand{
		{ID: 20, Name: "Child Demand 20", Parent: 10},
	}
	existingStories := []ZtStory{
		{ID: 201, Title: "Story 201", Product: 100, FromDemand: 20, SourceType: "demandpool", Status: "active"},
	}
	mockDemandScopeQueries(mock, demandID, childDemands, existingStories, 1)

	mock.ExpectQuery(`SELECT mainSystem FROM zt_demand WHERE id = \? AND deleted = '0' LIMIT 1`).
		WithArgs(demandID).
		WillReturnRows(sqlmock.NewRows([]string{"mainSystem"}).AddRow("100"))

	mock.ExpectQuery(`(?s)SELECT id, name, code, status, PO, QD, RD, createdBy, whitelist\s+FROM zt_product`).
		WithArgs("alice", "alice", "alice", "alice", "alice").
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "code", "status", "PO", "QD", "RD", "createdBy", "whitelist"}).
			AddRow(100, "Authorized Product", "AUTH", "normal", "alice", "alice", "alice", "alice", ""))

	// Transaction begin
	mock.ExpectBegin()

	// applySchedulingStory: delete child story 201
	mock.ExpectExec(`UPDATE `+"`zt_story`"+` SET .* WHERE id = \?`).
		WithArgs("alice", sqlmock.AnyArg(), "done", "closed", 201).
		WillReturnResult(sqlmock.NewResult(1, 1))

	mock.ExpectExec(`INSERT INTO ` + "`zt_action`" + ` .*`).
		WillReturnResult(sqlmock.NewResult(1, 1))

	// UpdateDemandScheduling
	mock.ExpectExec(`UPDATE `+"`zt_demand`"+` SET .* WHERE id = \?`).
		WithArgs("", "", nil, "", nil, "alice", sqlmock.AnyArg(), nil, demandID).
		WillReturnResult(sqlmock.NewResult(1, 1))

	// SaveDemandLevelWindow
	mock.ExpectExec(`DELETE FROM ` + "`zt_demandwindow`" + ` WHERE demand = \? AND story = 0`).
		WithArgs(demandID).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(`INSERT INTO ` + "`zt_demandwindow`" + ` .*`).
		WillReturnResult(sqlmock.NewResult(1, 1))

	mock.ExpectCommit()

	req := &SaveSchedulingReq{
		WindowID: 1,
		Stories: []SaveSchedulingStory{
			{Action: "delete", ID: 201},
		},
	}

	err := svc.SaveScheduling(ctx, actor, demandID, req)
	if err != nil {
		t.Fatalf("expected success for child demand story delete, got: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet mock expectations: %v", err)
	}
}
