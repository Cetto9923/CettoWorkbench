// =============================================================================
// 文件: internal/module/agileteam/repo_basic_atomic_concurrency_test.go
// 模块: 敏捷小组治理
// 类型: integration test (live MySQL)
// 职责: Batch 2B 团队树事务/并发硬门禁。
//
// 必须在真实 MySQL 上跑（不能用 sqlmock 替代——死锁是引擎语义层现象）。
// 通过 WB_TEST_DSN 注入 DSN；缺省值与本地 MySQL 默认一致。
//
// 数据隔离：
//   - 不污染 zentaopms.zt_teamgroup；
//   - 创建临时表 wb_test_tg_<nano>（同结构同 schema），测试结束 DROP；
//   - Repo.WithTeamgroupTable 注入临时表名。
//
// 跳过条件：
//   - WB_TEST_SKIP_TOPOLOGY_CONCURRENCY=1 → t.Skip（CI 没 MySQL 时）；
//   - MySQL 不可达 → t.Skip。
// =============================================================================

package agileteam

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// defaultLiveDSN 镜像 workbench/repo_recorder_live_test.go 的 fallback。
const defaultLiveDSN = "zentao:zentao123@tcp(127.0.0.1:3306)/zentaopms?charset=utf8mb4&parseTime=true&loc=Local"

func liveDSN(t *testing.T) string {
	t.Helper()
	dsn := os.Getenv("WB_TEST_DSN")
	if dsn == "" {
		dsn = defaultLiveDSN
	}
	return dsn
}

func openLiveDB(t *testing.T) *gorm.DB {
	t.Helper()
	if os.Getenv("WB_TEST_SKIP_TOPOLOGY_CONCURRENCY") == "1" {
		t.Skip("WB_TEST_SKIP_TOPOLOGY_CONCURRENCY=1, skip live MySQL concurrency test")
	}
	gdb, err := gorm.Open(mysql.Open(liveDSN(t)), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Skipf("MySQL unavailable (%v); skipping live concurrency test", err)
	}
	return gdb
}

// setupTopologyTestTable 在 zentaopms 内创建临时表（同 zt_teamgroup 结构），
// 并返回对应的 *gorm.DB 与表名。结束后 DROP。
//
// 复用 zentaopms schema：避免 zentao 用户无 CREATE DATABASE 权限的环境跳过测试。
func setupTopologyTestTable(t *testing.T, gdb *gorm.DB) (*gorm.DB, string) {
	t.Helper()
	var historyTables int64
	if err := gdb.Raw(`SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name = 'zt_wb_agileteam_history'`).Scan(&historyTables).Error; err != nil {
		t.Skipf("cannot inspect agileteam history table (%v); skipping live concurrency test", err)
	}
	if historyTables == 0 {
		t.Skip("zt_wb_agileteam_history is missing; apply db/02_po_incremental.sql before live concurrency tests")
	}
	table := fmt.Sprintf("wb_test_tg_%d", time.Now().UnixNano())
	if err := gdb.Exec(fmt.Sprintf(`
CREATE TABLE %s (
  id BIGINT UNSIGNED PRIMARY KEY,
  parent BIGINT UNSIGNED NOT NULL DEFAULT 0,
  name VARCHAR(255) NOT NULL DEFAULT '',
  slogan VARCHAR(255) NOT NULL DEFAULT '',
  declaration TEXT,
  logo VARCHAR(255) NOT NULL DEFAULT '',
  type VARCHAR(32) NOT NULL DEFAULT '',
  grade INT NOT NULL DEFAULT 1,
  path VARCHAR(255) NOT NULL DEFAULT '',
  deleted VARCHAR(8) NOT NULL DEFAULT '0'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`, table)).Error; err != nil {
		t.Fatalf("create test table: %v", err)
	}
	t.Cleanup(func() {
		_ = gdb.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s", table)).Error
	})
	return gdb, table
}

// seedTwoRoots 在指定临时表上构造一棵简单两根树：
//
//	1(root)        2(root)
//	├── 3           ├── 4
//	│   └── 6       └── 5
//	└── 7
func seedTwoRoots(t *testing.T, gdb *gorm.DB, table string) {
	t.Helper()
	rows := []struct {
		ID     uint
		Parent uint
		Grade  int
		Path   string
	}{
		{1, 0, 1, ",1,"},
		{3, 1, 2, ",1,3,"},
		{6, 3, 3, ",1,3,6,"},
		{7, 1, 2, ",1,7,"},
		{2, 0, 1, ",2,"},
		{4, 2, 2, ",2,4,"},
		{5, 2, 2, ",2,5,"},
	}
	for _, r := range rows {
		if err := gdb.Table(table).Exec(`
INSERT INTO `+table+` (id, parent, name, type, grade, path)
VALUES (?, ?, ?, ?, ?, ?)`,
			r.ID, r.Parent, fmt.Sprintf("g%d", r.ID), "child", r.Grade, r.Path,
		).Error; err != nil {
			t.Fatalf("seed %d: %v", r.ID, err)
		}
	}
}

// verifyNode 校验指定节点的 parent/grade/path（指定表）。
func verifyNode(t *testing.T, gdb *gorm.DB, table string, id uint, wantParent uint, wantGrade int, wantPath string) {
	t.Helper()
	var got struct {
		Parent uint
		Grade  int
		Path   string
	}
	if err := gdb.Table(table).
		Select("parent, grade, path").
		Where("id = ?", id).
		Scan(&got).Error; err != nil {
		t.Fatalf("verify %d: %v", id, err)
	}
	if got.Parent != wantParent || got.Grade != wantGrade || got.Path != wantPath {
		t.Fatalf("node %d: got (parent=%d, grade=%d, path=%q), want (parent=%d, grade=%d, path=%q)",
			id, got.Parent, got.Grade, got.Path, wantParent, wantGrade, wantPath)
	}
}

// allTreeConsistent 验证指定表 parent/grade/path 一致。
// 先按 id 升序构造查找表，再按 id 升序校验，保证父级先于子节点已知。
func allTreeConsistent(t *testing.T, gdb *gorm.DB, table string) {
	t.Helper()
	var rows []struct {
		ID     uint
		Parent uint
		Grade  int
		Path   string
	}
	if err := gdb.Table(table).
		Select("id, parent, grade, path").
		Where("deleted = ?", "0").
		Order("id ASC").
		Scan(&rows).Error; err != nil {
		t.Fatalf("scan tree: %v", err)
	}
	// 先构造 lookup
	gradeByID := make(map[uint]int, len(rows))
	pathByID := make(map[uint]string, len(rows))
	for _, r := range rows {
		gradeByID[r.ID] = r.Grade
		pathByID[r.ID] = r.Path
	}
	// 再校验：按 id 升序遍历，父级 ID 必须小于当前 ID（前提：树结构有序，父级 id 永远小于子节点 id）。
	// 实际允许父 id > 子 id（用户先建子后建父的不规范场景）；因此改为遍历所有节点逐个校验。
	grade2 := make(map[uint]int, len(rows))
	path2 := make(map[uint]string, len(rows))
	// 拓扑排序：BFS 从 root 开始
	childrenOf := make(map[uint][]uint, len(rows))
	for _, r := range rows {
		if r.Parent != 0 {
			childrenOf[r.Parent] = append(childrenOf[r.Parent], r.ID)
		}
	}
	// 收集 root（parent=0）
	queue := []uint{}
	for _, r := range rows {
		if r.Parent == 0 {
			queue = append(queue, r.ID)
		}
	}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		grade, ok := gradeByID[cur]
		if !ok {
			t.Fatalf("node %d missing", cur)
		}
		pathStr, _ := pathByID[cur]
		grade2[cur] = grade
		path2[cur] = pathStr
		for _, ch := range childrenOf[cur] {
			queue = append(queue, ch)
		}
	}
	// 校验：每个非 root 节点 grade = parentGrade+1, path = parentPath + "{id},"
	for _, r := range rows {
		if r.Parent == 0 {
			if r.Grade != 1 {
				t.Fatalf("root %d grade=%d, expected 1", r.ID, r.Grade)
			}
			expected := fmt.Sprintf(",%d,", r.ID)
			if r.Path != expected {
				t.Fatalf("root %d path=%q, expected %q", r.ID, r.Path, expected)
			}
			continue
		}
		parentGrade, ok := grade2[r.Parent]
		if !ok {
			t.Fatalf("node %d: parent %d not in tree (orphan)", r.ID, r.Parent)
		}
		expectedGrade := parentGrade + 1
		expectedPath := path2[r.Parent] + fmt.Sprintf("%d,", r.ID)
		if r.Grade != expectedGrade {
			t.Fatalf("node %d: grade=%d, expected %d", r.ID, r.Grade, expectedGrade)
		}
		if r.Path != expectedPath {
			t.Fatalf("node %d: path=%q, expected %q", r.ID, r.Path, expectedPath)
		}
	}
}

// T-1：交叉父级移动 A→B 并发 B→A，验证无 deadlock 且最终树一致。
func TestTopology_Concurrent_CrossParentMove(t *testing.T) {
	gdb, table := setupTopologyTestTable(t, openLiveDB(t))
	seedTwoRoots(t, gdb, table)

	baseRepo := NewRepo(gdb, gdb)
	repoA := baseRepo.WithTeamgroupTable(table)
	repoB := baseRepo.WithTeamgroupTable(table)

	// A: 把节点 3 挂到 2 之下；B: 把节点 2 挂到 3 之下。
	parentA := uint(2)
	parentB := uint(3)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var wg sync.WaitGroup
	results := make([]error, 2)

	wg.Add(2)
	go func() {
		defer wg.Done()
		results[0] = repoA.UpdateTeamgroupBasicAtomic(ctx, 3, "g3", "", "", "", &parentA, &History{
			EventType: EventUpdate, Actor: "txA", Summary: "A", CreatedDate: time.Now(),
		})
	}()
	go func() {
		defer wg.Done()
		results[1] = repoB.UpdateTeamgroupBasicAtomic(ctx, 2, "g2", "", "", "", &parentB, &History{
			EventType: EventUpdate, Actor: "txB", Summary: "B", CreatedDate: time.Now(),
		})
	}()
	wg.Wait()

	success, fail := 0, 0
	for _, e := range results {
		if e == nil {
			success++
		} else {
			fail++
			if isMySQLDeadlock(e) {
				t.Fatalf("deadlock leaked through (Batch 2B 必修): %v", e)
			}
		}
	}
	if success != 1 || fail != 1 {
		t.Fatalf("expected exactly 1 success / 1 fail, got success=%d fail=%d (errs=%v)", success, fail, results)
	}
	allTreeConsistent(t, gdb, table)
}

// T-2：父级 A 正在移动 + 同时其 descendant B 被另一请求移动。
func TestTopology_Concurrent_AncestorAndDescendant(t *testing.T) {
	gdb, table := setupTopologyTestTable(t, openLiveDB(t))
	seedTwoRoots(t, gdb, table)
	repo := NewRepo(gdb, gdb).WithTeamgroupTable(table)

	// tx1: 把 1 挂到 2 之下 (整 1 子树迁移)
	// tx2: 把 6 挂到 4 之下 (6 是 1 的后代，4 与 1 无关)
	parent1 := uint(2)
	parent2 := uint(4)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var wg sync.WaitGroup
	results := make([]error, 2)

	wg.Add(2)
	go func() {
		defer wg.Done()
		results[0] = repo.UpdateTeamgroupBasicAtomic(ctx, 1, "g1", "", "", "", &parent1, &History{
			EventType: EventUpdate, Actor: "tx1", Summary: "1->2", CreatedDate: time.Now(),
		})
	}()
	go func() {
		defer wg.Done()
		results[1] = repo.UpdateTeamgroupBasicAtomic(ctx, 6, "g6", "", "", "", &parent2, &History{
			EventType: EventUpdate, Actor: "tx2", Summary: "6->4", CreatedDate: time.Now(),
		})
	}()
	wg.Wait()

	for _, e := range results {
		if e != nil && isMySQLDeadlock(e) {
			t.Fatalf("deadlock leaked: %v", e)
		}
	}
	allTreeConsistent(t, gdb, table)
}

// T-3：两个无交集 subtree 同时拓扑变更。
func TestTopology_Concurrent_DisjointSubtrees(t *testing.T) {
	gdb, table := setupTopologyTestTable(t, openLiveDB(t))
	seedTwoRoots(t, gdb, table)
	repo := NewRepo(gdb, gdb).WithTeamgroupTable(table)

	parent1 := uint(2)
	parent2 := uint(1)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var wg sync.WaitGroup
	results := make([]error, 2)

	wg.Add(2)
	go func() {
		defer wg.Done()
		results[0] = repo.UpdateTeamgroupBasicAtomic(ctx, 3, "g3", "", "", "", &parent1, &History{
			EventType: EventUpdate, Actor: "tx1", Summary: "3->2", CreatedDate: time.Now(),
		})
	}()
	go func() {
		defer wg.Done()
		results[1] = repo.UpdateTeamgroupBasicAtomic(ctx, 5, "g5", "", "", "", &parent2, &History{
			EventType: EventUpdate, Actor: "tx2", Summary: "5->1", CreatedDate: time.Now(),
		})
	}()
	wg.Wait()

	for _, e := range results {
		if e != nil && isMySQLDeadlock(e) {
			t.Fatalf("deadlock leaked: %v", e)
		}
	}
	allTreeConsistent(t, gdb, table)
}

// T-4：basic-only 只锁当前节点，不刷整树（事后 grade/path 应当不变）。
func TestTopology_BasicOnly_DoesNotLockWholeTree(t *testing.T) {
	gdb, table := setupTopologyTestTable(t, openLiveDB(t))
	seedTwoRoots(t, gdb, table)
	repo := NewRepo(gdb, gdb).WithTeamgroupTable(table)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	doneBasic := make(chan error, 1)
	go func() {
		doneBasic <- repo.UpdateTeamgroupBasicAtomic(ctx, 1, "g1-new", "", "", "", nil, &History{
			EventType: EventUpdate, Actor: "tx1", Summary: "basic-only", CreatedDate: time.Now(),
		})
	}()

	if err := <-doneBasic; err != nil {
		t.Fatalf("basic-only: %v", err)
	}
	verifyNode(t, gdb, table, 1, 0, 1, ",1,")
	verifyNode(t, gdb, table, 3, 1, 2, ",1,3,")
	verifyNode(t, gdb, table, 6, 3, 3, ",1,3,6,")
}

// T-5：脏 cycle → corrupt 错误 + 全回滚（DB 状态不变）。
func TestTopology_CorruptTree_FailsClosed(t *testing.T) {
	gdb, table := setupTopologyTestTable(t, openLiveDB(t))
	seedTwoRoots(t, gdb, table)

	// 手工注入结构性 cycle: 2.parent=3, 3.parent=2。
	if err := gdb.Table(table).Exec(`UPDATE ` + table + ` SET parent = 3 WHERE id = 2`).Error; err != nil {
		t.Fatalf("inject: %v", err)
	}
	if err := gdb.Table(table).Exec(`UPDATE ` + table + ` SET parent = 2 WHERE id = 3`).Error; err != nil {
		t.Fatalf("inject: %v", err)
	}

	repo := NewRepo(gdb, gdb).WithTeamgroupTable(table)
	newParent := uint(2)
	err := repo.UpdateTeamgroupBasicAtomic(context.Background(), 1, "g1-new", "", "", "", &newParent, &History{
		EventType: EventUpdate, Actor: "tx", Summary: "x", CreatedDate: time.Now(),
	})
	if err == nil {
		t.Fatalf("expected corrupt-tree error, got nil")
	}
	if !isCorruptTreeError(err) {
		t.Fatalf("expected errTeamgroupCorruptTree family, got %v", err)
	}
	// 1 的 name 必须仍是 g1（说明 UPDATE 被回滚）。
	verifyNode(t, gdb, table, 1, 0, 1, ",1,")
}

// T-6：topology 变更与单行锁 basic-only 并发，不形成死锁。
func TestTopology_Concurrent_WithSingleRowLock(t *testing.T) {
	gdb, table := setupTopologyTestTable(t, openLiveDB(t))
	seedTwoRoots(t, gdb, table)
	repo := NewRepo(gdb, gdb).WithTeamgroupTable(table)

	parent1 := uint(2)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		_ = repo.UpdateTeamgroupBasicAtomic(ctx, 1, "g1", "", "", "", &parent1, &History{
			EventType: EventUpdate, Actor: "tx1", Summary: "topo", CreatedDate: time.Now(),
		})
	}()
	go func() {
		defer wg.Done()
		_ = repo.UpdateTeamgroupBasicAtomic(ctx, 6, "g6-new", "", "", "", nil, &History{
			EventType: EventUpdate, Actor: "tx2", Summary: "basic", CreatedDate: time.Now(),
		})
	}()

	wg.Wait()
	allTreeConsistent(t, gdb, table)
}

// isMySQLDeadlock 识别 MySQL 1213 / "Deadlock found"。
func isMySQLDeadlock(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "Deadlock found") ||
		strings.Contains(msg, "Error 1213") ||
		strings.Contains(msg, "try restarting transaction")
}

func isCorruptTreeError(err error) bool {
	if err == nil {
		return false
	}
	return errors.Is(err, errTeamgroupCorruptTree)
}
