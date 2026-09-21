// =============================================================================
// 文件: internal/module/debug/queries_cache.go
// 模块: SQL 性能分析
// 类型: readonly
// 职责: 按日期缓存 SQL 明细的 top-K 堆与已消费偏移量，日志新增时只解析增量部分。
// 依赖: container/heap
//       internal/pkg/sqllog
// =============================================================================

package debug

import (
	"container/heap"
	"os"
	"slices"
	"strings"
	"sync"

	"workbench/internal/pkg/sqllog"
)

// maxCachedDates 最多保留的日期数
const maxCachedDates = 8

// queriesCache 按日期缓存 SQL 明细。日志按天追加，除当天外内容不再变化，
// 因此重复请求通常只需一次 stat 比对。
type queriesCache struct {
	mu      sync.Mutex
	seq     uint64
	entries map[string]*queriesCacheEntry
}

func newQueriesCache() *queriesCache {
	return &queriesCache{entries: make(map[string]*queriesCacheEntry)}
}

// entry 取指定日期的缓存，没有则新建；超过上限时淘汰最久未使用的日期。
func (c *queriesCache) entry(date string) *queriesCacheEntry {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.seq++
	if e, ok := c.entries[date]; ok {
		e.used = c.seq
		return e
	}

	if len(c.entries) >= maxCachedDates {
		var oldestDate string
		var oldestUsed uint64
		for d, e := range c.entries {
			if oldestDate == "" || e.used < oldestUsed {
				oldestDate, oldestUsed = d, e.used
			}
		}
		delete(c.entries, oldestDate)
	}

	e := &queriesCacheEntry{used: c.seq}
	c.entries[date] = e
	return e
}

// queriesCacheEntry 单个日期的缓存：文件身份 + 已消费偏移量 + 累计行数 + top-K 堆。
type queriesCacheEntry struct {
	mu     sync.Mutex
	used   uint64      // 最近使用序号，只由 queriesCache.mu 保护
	info   os.FileInfo // 上次 stat 到的文件，用于判断是否还是同一份
	offset int64       // 已消费到的字节位置，只推进到最后一个完整行
	total  int64       // 累计解析行数（页面「共 N 条」）
	worst  queryHeap   // 堆顶是保留集合里最该淘汰的一条
}

// items 返回该日期耗时最高的 limit 条与累计行数，必要时只解析文件新增的部分。
func (e *queriesCacheEntry) items(path string, limit int) ([]QueryItem, int64, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	fi, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []QueryItem{}, 0, nil
		}
		return nil, 0, err
	}

	// 首次读取、文件被替换（inode 变了）、被截断，三种情况都从头重建。
	if e.info == nil || !os.SameFile(e.info, fi) || fi.Size() < e.offset {
		e.worst, e.total, e.offset = nil, 0, 0
	}

	if fi.Size() > e.offset {
		next, err := sqllog.ScanQueryEntries(path, e.offset, func(q sqllog.QueryEntry) {
			e.total++
			e.add(toQueryItem(q))
		})
		if err != nil {
			// 半途出错时本次结果不完整，丢弃状态让下次请求重建，避免偏移量与计数错位。
			e.worst, e.total, e.offset = nil, 0, 0
			return nil, 0, err
		}
		e.offset = next
	}
	e.info = fi

	return e.top(limit), e.total, nil
}

// add 把一个条目并入 top-K 堆，堆满后只保留更靠前的。
func (e *queriesCacheEntry) add(item QueryItem) {
	if len(e.worst) < maxQueryLimit {
		heap.Push(&e.worst, item)
		return
	}
	if worse(e.worst[0], item) {
		e.worst[0] = item
		heap.Fix(&e.worst, 0)
	}
}

// top 取出耗时最高的 limit 条。堆内本来就没有顺序，就地排序取前若干条，再用 heap.Init 还原。
// 结果为空时返回非 nil 空切片，保证 JSON 里 queries 是 [] 而不是 null。
func (e *queriesCacheEntry) top(limit int) []QueryItem {
	if len(e.worst) == 0 || limit <= 0 {
		return []QueryItem{}
	}

	slices.SortFunc(e.worst, func(a, b QueryItem) int {
		switch {
		case a.ElapsedMS > b.ElapsedMS:
			return -1
		case a.ElapsedMS < b.ElapsedMS:
			return 1
		default:
			return strings.Compare(b.Time, a.Time)
		}
	})

	if limit > len(e.worst) {
		limit = len(e.worst)
	}
	out := slices.Clone(e.worst[:limit])

	heap.Init(&e.worst)
	return out
}

// queryHeap 最小堆：堆顶是「耗时更低、同耗时时间更早」的一条，即列表排序时最靠后的那个。
type queryHeap []QueryItem

func (h queryHeap) Len() int           { return len(h) }
func (h queryHeap) Less(i, j int) bool { return worse(h[i], h[j]) }
func (h queryHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }

func (h *queryHeap) Push(x any) { *h = append(*h, x.(QueryItem)) }

func (h *queryHeap) Pop() any {
	old := *h
	n := len(old)
	item := old[n-1]
	*h = old[:n-1]
	return item
}

// worse 报告 a 是否排在 b 之后：耗时更低者靠后，同耗时则时间更早者靠后。
func worse(a, b QueryItem) bool {
	if a.ElapsedMS != b.ElapsedMS {
		return a.ElapsedMS < b.ElapsedMS
	}
	return a.Time < b.Time
}
