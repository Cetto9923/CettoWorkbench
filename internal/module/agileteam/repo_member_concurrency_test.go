// =============================================================================
// 文件: internal/module/agileteam/repo_member_concurrency_test.go
// 模块: 敏捷小组治理
// 类型: integration test
// 职责: 在显式授权的隔离库验证成员原子 upsert 与加入日期保留。
// =============================================================================

package agileteam

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestMemberConcurrentUpsertPreservesIdentity(t *testing.T) {
	db := openLiveDB(t)
	const root, account = 16000000, "wb_test_member_race"
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	remove := func() {
		db.Exec("DELETE FROM zt_team WHERE root = ? AND type = ? AND account = ?", root, "teamgroup", account)
	}
	remove()
	t.Cleanup(remove)
	joined := time.Date(2020, 1, 2, 0, 0, 0, 0, time.Local)
	var wg sync.WaitGroup
	failures := make(chan error, 12)
	for i := 0; i < 12; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			failures <- upsertTeamMemberTx(db.WithContext(ctx), root, account, "测试", 6, joined)
		}()
	}
	wg.Wait()
	close(failures)
	for err := range failures {
		if err != nil {
			t.Fatal(err)
		}
	}
	if err := upsertTeamMemberTx(db.WithContext(ctx), root, account, "研发", 6, time.Now()); err != nil {
		t.Fatal(err)
	}
	var result struct {
		Count  int
		Hours  float64
		Joined string
	}
	if err := db.Raw("SELECT COUNT(*) AS count, MAX(hours) AS hours, MAX(DATE_FORMAT(`join`, '%Y-%m-%d')) AS joined FROM zt_team WHERE root = ? AND type = ? AND account = ?", root, "teamgroup", account).Scan(&result).Error; err != nil {
		t.Fatal(err)
	}
	if result.Count != 1 || result.Hours != 6 || result.Joined != "2020-01-02" {
		t.Fatalf("member identity changed: %+v", result)
	}
}

func TestBasicSaveDoesNotWaitForUnrelatedRow(t *testing.T) {
	db, table := setupTopologyTestTable(t, openLiveDB(t))
	seedTwoRoots(t, db, table)
	locked := db.Begin()
	if locked.Error != nil {
		t.Fatal(locked.Error)
	}
	defer locked.Rollback()
	if err := locked.Exec("SELECT id FROM "+table+" WHERE id = ? FOR UPDATE", 2).Error; err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	repo := NewRepo(db, db).WithTeamgroupTable(table)
	if err := repo.UpdateTeamgroupBasicAtomic(ctx, 1, "g1", "", "", "", nil, nil); err != nil {
		t.Fatalf("unrelated row lock blocked basic save: %v", err)
	}
}

func TestObjectAuthorizationReadsPrimary(t *testing.T) {
	primary, writer := newTestService(t)
	weak, reader := newTestService(t)
	repo := NewRepo(primary.repo.db, weak.repo.db)
	writer.ExpectQuery("(?s)SELECT tg.id.*WHERE tg.id =").WithArgs(uint(5)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "PO"}).AddRow(5, "current-owner"))
	group, err := repo.FindTeamgroupByID(t.Context(), 5)
	if err != nil || group.PO != "current-owner" {
		t.Fatalf("group authorization used stale reader: %+v, %v", group, err)
	}
	writer.ExpectQuery("(?s)SELECT t.account.*WHERE t.type =").WithArgs(uint(5), "current-member").
		WillReturnRows(sqlmock.NewRows([]string{"account", "role"}).AddRow("current-member", "研发"))
	member, err := repo.FindMember(t.Context(), 5, "current-member")
	if err != nil || member.Account != "current-member" {
		t.Fatalf("member validation used stale reader: %+v, %v", member, err)
	}
	writer.ExpectQuery("(?s)SELECT t.account.*WHERE t.type =").WithArgs(uint(5)).
		WillReturnRows(sqlmock.NewRows([]string{"account", "hours"}).AddRow("current-member", 5))
	members, err := repo.ListMembers(t.Context(), 5)
	if err != nil || len(members) != 1 || members[0].Hours != 5 {
		t.Fatalf("member detail used stale reader: %+v, %v", members, err)
	}
	writer.ExpectQuery("(?s)SELECT tg.id.*WHERE tg.deleted =").
		WillReturnRows(sqlmock.NewRows([]string{"id", "org_dept_id", "org_dept_name"}).AddRow(5, 14, "产品五部"))
	groups, err := repo.ListTeamgroups(t.Context())
	if err != nil || len(groups) != 1 || groups[0].OrgDeptID != 14 {
		t.Fatalf("organization mapping used stale reader: %+v, %v", groups, err)
	}
	for _, mock := range []sqlmock.Sqlmock{writer, reader} {
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatal(err)
		}
	}
}
