package agileteam

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestSetOrgTeamMappingWritesEffectiveMappingAndHistoryAtomically(t *testing.T) {
	svc, mock := newTestService(t)
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT id FROM zt_teamgroup WHERE id = \\? AND deleted = '0' FOR UPDATE").
		WithArgs(uint(5)).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(uint(5)))
	mock.ExpectQuery("SELECT d.id FROM zt_dept d WHERE d.id = \\?.*NOT EXISTS").
		WithArgs(uint(3)).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(uint(3)))
	mock.ExpectExec("UPDATE zt_wb_agileteam_orgmap SET status = 'inactive'").
		WithArgs("pmo1", uint(5)).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("INSERT INTO zt_wb_agileteam_orgmap").
		WithArgs(uint(5), uint(3), "pmo1", "pmo1").WillReturnResult(sqlmock.NewResult(8, 1))
	mock.ExpectExec("INSERT INTO zt_wb_agileteam_history").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	if err := svc.repo.SetOrgTeamMapping(context.Background(), 5, 3, "pmo1"); err != nil {
		t.Fatalf("SetOrgTeamMapping() error = %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("SQL expectations: %v", err)
	}
}

func TestSetOrgTeamMappingRejectsNonLeafDepartmentAndRollsBack(t *testing.T) {
	svc, mock := newTestService(t)
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT id FROM zt_teamgroup WHERE id = \\? AND deleted = '0' FOR UPDATE").
		WithArgs(uint(5)).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(uint(5)))
	mock.ExpectQuery("SELECT d.id FROM zt_dept d WHERE d.id = \\?.*NOT EXISTS").
		WithArgs(uint(3)).WillReturnRows(sqlmock.NewRows([]string{"id"}))
	mock.ExpectRollback()

	if err := svc.repo.SetOrgTeamMapping(context.Background(), 5, 3, "pmo1"); err == nil {
		t.Fatal("expected non-leaf department to be rejected")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("SQL expectations: %v", err)
	}
}

func TestListTeamgroupsUsesChildMappingBeforeInheritedParentMapping(t *testing.T) {
	svc, mock := newTestService(t)
	mock.ExpectQuery("(?s)COALESCE\\(own_map.deptId, parent_map.deptId, 0\\).*LEFT JOIN zt_wb_agileteam_orgmap own_map").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "name", "parent", "parent_name", "org_dept_id", "org_dept_name", "org_dept_inherited",
			"type", "grade", "path", "PO", "manager", "slogan", "declaration", "logo", "status", "createdDate",
		}).AddRow(uint(12), "子组", uint(3), "父团队", uint(34), "研发团队", true,
			"child", 2, ",3,12,", "po1", "coach1", "", "", "", "enable", "2026-09-01"))

	rows, err := svc.repo.ListTeamgroups(context.Background())
	if err != nil {
		t.Fatalf("ListTeamgroups() error = %v", err)
	}
	if len(rows) != 1 || rows[0].OrgDeptID != 34 || rows[0].OrgDeptName != "研发团队" || !rows[0].OrgDeptInherited {
		t.Fatalf("unexpected inherited mapping: %+v", rows)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("SQL expectations: %v", err)
	}
}
