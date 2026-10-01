// =============================================================================
// 文件: internal/module/po/serviceparallel.go
// 模块: 产品经理工作台
// 类型: action
// 职责: 为并发取数提供统一 panic 边界及 WaitGroup 收尾。
// 依赖: 无
// =============================================================================
package po

import (
	"fmt"
	"sync"
)

// 每个取数任务有独立错误槽；panic 转成错误，不能作为成功的空数据返回。
func runQuerySafely(wg *sync.WaitGroup, err *error, query func()) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer func() {
			if recovered := recover(); recovered != nil {
				*err = fmt.Errorf("parallel query panic: %v", recovered)
			}
		}()
		query()
	}()
}
