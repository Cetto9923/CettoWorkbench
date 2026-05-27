// =============================================================================
// 文件: cmd/server/main.go
// 模块: 应用入口
// 类型: entry
// 职责: 应用启动入口，触发 bootstrap 启动流程。
// 依赖: internal/bootstrap
// =============================================================================

package main

import (
	"log"

	"goframework/internal/bootstrap"
)

func main() {
	if err := bootstrap.Run(); err != nil {
		log.Fatal(err)
	}
}
