// =============================================================================
// 文件: internal/module/agileteam/repo_basic_atomic_test.go
// 模块: 敏捷小组治理
// 类型: test
// 职责: 基本信息 + 层级 + 历史必须处于同一事务。
// =============================================================================

package agileteam

import (
	"context"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

// readTreeIDsRe 用于匹配 gorm 生成的"无锁读 id 列表"语句。
// 真实 SQL: SELECT id FROM `zt_teamgroup` WHERE deleted = ? ORDER BY id ASC
var readTreeIDsRe = regexp.MustCompile("(?s)SELECT id FROM `zt_teamgroup` WHERE deleted = \\? ORDER BY id ASC")

// lockTreeRowsRe 用于匹配 gorm 生成的"IN (...) FOR UPDATE"全行锁语句。
// 真实 SQL: SELECT id, parent, grade, COALESCE(path,”) AS path FROM `zt_teamgroup` WHERE id IN (?,?,...) AND deleted = ? ORDER BY id ASC FOR UPDATE
var lockTreeRowsRe = regexp.MustCompile("(?s)SELECT id, parent, grade.*FROM `zt_teamgroup` WHERE id IN \\(\\?.*\\) AND deleted = \\? ORDER BY id ASC FOR UPDATE")

// updateBasicRe 匹配 gorm 生成的 basic UPDATE。
// 真实 SQL: UPDATE `zt_teamgroup` SET `declaration`=?,`logo`=?,`name`=?,`slogan`=? WHERE id = ? AND deleted = ?
var updateBasicRe = regexp.MustCompile("(?s)UPDATE `zt_teamgroup` SET `declaration`.*`name`.*`slogan`.*WHERE id = \\? AND deleted = \\?")

// updateParentRe 匹配 gorm 生成的 parent/type/grade/path UPDATE。
// 真实 SQL: UPDATE `zt_teamgroup` SET `grade`=?,`parent`=?,`path`=?,`type`=? WHERE id = ? AND deleted = ?
var updateParentRe = regexp.MustCompile("(?s)UPDATE `zt_teamgroup` SET `grade`.*`parent`.*`path`.*`type`.*WHERE id = \\? AND deleted = \\?")

// updateDescendantRe 匹配 gorm 生成的子孙 grade/path UPDATE。
// 真实 SQL: UPDATE `zt_teamgroup` SET `grade`=?,`path`=? WHERE id = ? AND deleted = ?
var updateDescendantRe = regexp.MustCompile("(?s)UPDATE `zt_teamgroup` SET `grade`.*`path`.*WHERE id = \\? AND deleted = \\?")

// insertHistoryRe 匹配 gorm 生成的 history INSERT。
var insertHistoryRe = regexp.MustCompile("(?s)INSERT INTO `zt_wb_agileteam_history`")

func TestUpdateTeamgroupBasicAtomicCommitsWithHistory(t *testing.T) {
	svc, mock := newTestService(t)
	history := &History{
		TeamgroupID: 3,
		EventType:   EventUpdate,
		Actor:       "po1",
		Summary:     "基本信息已更新",
		CreatedDate: time.Now(),
	}

	mock.ExpectBegin()
	// 1) 无锁读 id 列表
	mock.ExpectQuery(readTreeIDsRe.String()).
		WithArgs("0").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).
			AddRow(uint(3)))
	// 2) IN FOR UPDATE 锁全行
	mock.ExpectQuery(lockTreeRowsRe.String()).
		WithArgs(uint(3), "0").
		WillReturnRows(sqlmock.NewRows([]string{"id", "parent", "grade", "path"}).
			AddRow(uint(3), uint(0), 1, ",3,"))
	// 3) basic UPDATE
	mock.ExpectExec(updateBasicRe.String()).
		WithArgs("宣言", "logo.png", "团队三", "口号", uint(3), "0").
		WillReturnResult(sqlmock.NewResult(0, 1))
	// 4) history
	mock.ExpectExec(insertHistoryRe.String()).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	if err := svc.repo.UpdateTeamgroupBasicAtomic(
		context.Background(), 3, "团队三", "口号", "宣言", "logo.png", nil, history,
	); err != nil {
		t.Fatalf("atomic basic update: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

func TestUpdateTeamgroupBasicAtomicRollsBackWhenHistoryFails(t *testing.T) {
	svc, mock := newTestService(t)
	history := &History{
		TeamgroupID: 3,
		EventType:   EventUpdate,
		Actor:       "po1",
		Summary:     "基本信息已更新",
		CreatedDate: time.Now(),
	}
	writeErr := errors.New("history write failed")

	mock.ExpectBegin()
	mock.ExpectQuery(readTreeIDsRe.String()).
		WithArgs("0").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(uint(3)))
	mock.ExpectQuery(lockTreeRowsRe.String()).
		WithArgs(uint(3), "0").
		WillReturnRows(sqlmock.NewRows([]string{"id", "parent", "grade", "path"}).
			AddRow(uint(3), uint(0), 1, ",3,"))
	mock.ExpectExec(updateBasicRe.String()).
		WithArgs("宣言", "logo.png", "团队三", "口号", uint(3), "0").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(insertHistoryRe.String()).
		WillReturnError(writeErr)
	mock.ExpectRollback()

	err := svc.repo.UpdateTeamgroupBasicAtomic(
		context.Background(), 3, "团队三", "口号", "宣言", "logo.png", nil, history,
	)
	if !errors.Is(err, writeErr) {
		t.Fatalf("want history error, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

// TestUpdateTeamgroupBasicAtomic_TopologyChange_TwoLevelTree 验证 parent 变更走 topology 分支。
func TestUpdateTeamgroupBasicAtomic_TopologyChange_TwoLevelTree(t *testing.T) {
	svc, mock := newTestService(t)
	history := &History{
		TeamgroupID: 3,
		EventType:   EventUpdate,
		Actor:       "po1",
		Summary:     "基本信息+父级更新",
		CreatedDate: time.Now(),
	}
	newParent := uint(2)

	mock.ExpectBegin()
	// 1) 无锁读 id 列表
	mock.ExpectQuery(readTreeIDsRe.String()).
		WithArgs("0").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).
			AddRow(uint(1)).
			AddRow(uint(2)).
			AddRow(uint(3)).
			AddRow(uint(4)))
	// 2) IN FOR UPDATE 锁全行（升序）
	mock.ExpectQuery(lockTreeRowsRe.String()).
		WithArgs(uint(1), uint(2), uint(3), uint(4), "0").
		WillReturnRows(sqlmock.NewRows([]string{"id", "parent", "grade", "path"}).
			AddRow(uint(1), uint(0), 1, ",1,").
			AddRow(uint(2), uint(0), 1, ",2,").
			AddRow(uint(3), uint(0), 1, ",3,").
			AddRow(uint(4), uint(3), 2, ",3,4,"))
	// 3) basic UPDATE
	mock.ExpectExec(updateBasicRe.String()).
		WithArgs("宣言", "logo.png", "团队三", "口号", uint(3), "0").
		WillReturnResult(sqlmock.NewResult(0, 1))
	// 4) parent/type/grade/path UPDATE (SET 顺序: grade,parent,path,type)
	mock.ExpectExec(updateParentRe.String()).
		WithArgs(2, newParent, ",2,3,", "child", uint(3), "0").
		WillReturnResult(sqlmock.NewResult(0, 1))
	// 5) iterative refresh: 刷 current 的子孙 4 (grade 2→3, path ",3,4," → ",2,3,4,")
	mock.ExpectExec(updateDescendantRe.String()).
		WithArgs(3, ",2,3,4,", uint(4), "0").
		WillReturnResult(sqlmock.NewResult(0, 1))
	// 6) history
	mock.ExpectExec(insertHistoryRe.String()).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	if err := svc.repo.UpdateTeamgroupBasicAtomic(
		context.Background(), 3, "团队三", "口号", "宣言", "logo.png", &newParent, history,
	); err != nil {
		t.Fatalf("topology update: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

// TestUpdateTeamgroupBasicAtomic_TopologyChange_SelfParent 拒绝自指。
func TestUpdateTeamgroupBasicAtomic_TopologyChange_SelfParent(t *testing.T) {
	svc, mock := newTestService(t)
	history := &History{
		TeamgroupID: 3,
		EventType:   EventUpdate,
		Actor:       "po1",
		Summary:     "x",
		CreatedDate: time.Now(),
	}
	newParent := uint(3)

	mock.ExpectBegin()
	mock.ExpectQuery(readTreeIDsRe.String()).
		WithArgs("0").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(uint(3)))
	mock.ExpectQuery(lockTreeRowsRe.String()).
		WithArgs(uint(3), "0").
		WillReturnRows(sqlmock.NewRows([]string{"id", "parent", "grade", "path"}).
			AddRow(uint(3), uint(0), 1, ",3,"))
	mock.ExpectRollback()

	err := svc.repo.UpdateTeamgroupBasicAtomic(
		context.Background(), 3, "团队三", "口号", "宣言", "logo.png", &newParent, history,
	)
	if !errors.Is(err, errTeamgroupSelfParent) {
		t.Fatalf("want self-parent error, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

// TestUpdateTeamgroupBasicAtomic_TopologyChange_TargetMissing 父级不存在。
func TestUpdateTeamgroupBasicAtomic_TopologyChange_TargetMissing(t *testing.T) {
	svc, mock := newTestService(t)
	history := &History{
		TeamgroupID: 3,
		EventType:   EventUpdate,
		Actor:       "po1",
		Summary:     "x",
		CreatedDate: time.Now(),
	}
	newParent := uint(99)

	mock.ExpectBegin()
	mock.ExpectQuery(readTreeIDsRe.String()).
		WithArgs("0").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(uint(3)))
	mock.ExpectQuery(lockTreeRowsRe.String()).
		WithArgs(uint(3), "0").
		WillReturnRows(sqlmock.NewRows([]string{"id", "parent", "grade", "path"}).
			AddRow(uint(3), uint(0), 1, ",3,"))
	mock.ExpectRollback()

	err := svc.repo.UpdateTeamgroupBasicAtomic(
		context.Background(), 3, "团队三", "口号", "宣言", "logo.png", &newParent, history,
	)
	if !errors.Is(err, errTeamgroupParentMissing) {
		t.Fatalf("want parent-missing error, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

// TestUpdateTeamgroupBasicAtomic_TopologyChange_CycleDetected 目标父级是 current 的子孙。
func TestUpdateTeamgroupBasicAtomic_TopologyChange_CycleDetected(t *testing.T) {
	svc, mock := newTestService(t)
	history := &History{
		TeamgroupID: 3,
		EventType:   EventUpdate,
		Actor:       "po1",
		Summary:     "x",
		CreatedDate: time.Now(),
	}
	newParent := uint(4)

	mock.ExpectBegin()
	mock.ExpectQuery(readTreeIDsRe.String()).
		WithArgs("0").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).
			AddRow(uint(3)).
			AddRow(uint(4)))
	mock.ExpectQuery(lockTreeRowsRe.String()).
		WithArgs(uint(3), uint(4), "0").
		WillReturnRows(sqlmock.NewRows([]string{"id", "parent", "grade", "path"}).
			AddRow(uint(3), uint(0), 1, ",3,").
			AddRow(uint(4), uint(3), 2, ",3,4,"))
	mock.ExpectRollback()

	err := svc.repo.UpdateTeamgroupBasicAtomic(
		context.Background(), 3, "团队三", "口号", "宣言", "logo.png", &newParent, history,
	)
	if !errors.Is(err, errTeamgroupParentCycle) {
		t.Fatalf("want parent-cycle error, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

// TestUpdateTeamgroupBasicAtomic_TopologyChange_CorruptTree 结构性 cycle → corrupt。
func TestUpdateTeamgroupBasicAtomic_TopologyChange_CorruptTree(t *testing.T) {
	svc, mock := newTestService(t)
	history := &History{
		TeamgroupID: 3,
		EventType:   EventUpdate,
		Actor:       "po1",
		Summary:     "x",
		CreatedDate: time.Now(),
	}
	newParent := uint(5)

	mock.ExpectBegin()
	mock.ExpectQuery(readTreeIDsRe.String()).
		WithArgs("0").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).
			AddRow(uint(3)).
			AddRow(uint(5)).
			AddRow(uint(6)).
			AddRow(uint(7)))
	mock.ExpectQuery(lockTreeRowsRe.String()).
		WithArgs(uint(3), uint(5), uint(6), uint(7), "0").
		WillReturnRows(sqlmock.NewRows([]string{"id", "parent", "grade", "path"}).
			AddRow(uint(3), uint(0), 1, ",3,").
			AddRow(uint(5), uint(6), 1, ",5,").
			AddRow(uint(6), uint(7), 1, ",6,").
			AddRow(uint(7), uint(5), 1, ",7,"))
	mock.ExpectRollback()

	err := svc.repo.UpdateTeamgroupBasicAtomic(
		context.Background(), 3, "团队三", "口号", "宣言", "logo.png", &newParent, history,
	)
	if !errors.Is(err, errTeamgroupCorruptTree) {
		t.Fatalf("want corrupt-tree error, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}
