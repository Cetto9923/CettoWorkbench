// =============================================================================
// 文件: internal/bootstrap/operationlog_schema.go
// 模块: 基础设施
// 类型: infra
// 职责: 启动时只读检查 zt_operation_logs 表结构，禁止运行期 DDL。
// 依赖: gorm.io/gorm
// =============================================================================

package bootstrap

import (
	"fmt"
	"strings"

	"gorm.io/gorm"
)

const operationLogTableName = "zt_operation_logs"

// operationLogRequiredColumns matches model.OperationLog columns and the
// existing VM table. Startup must fail closed if any of these are missing.
var operationLogRequiredColumns = []string{
	"id",
	"tenantId",
	"userId",
	"account",
	"method",
	"path",
	"query",
	"body",
	"ip",
	"userAgent",
	"statusCode",
	"createdAt",
}

// operationLogSchemaReader is the read-only Migrator surface used at startup.
type operationLogSchemaReader interface {
	HasTable(dst any) bool
	HasColumn(dst any, field string) bool
}

func ensureOperationLogSchema(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("database handle is nil")
	}
	return checkOperationLogSchema(db.Migrator())
}

func checkOperationLogSchema(reader operationLogSchemaReader) error {
	if reader == nil {
		return fmt.Errorf("schema reader is nil")
	}
	if !reader.HasTable(operationLogTableName) {
		return fmt.Errorf("missing required table %s", operationLogTableName)
	}
	var missing []string
	for _, col := range operationLogRequiredColumns {
		if !reader.HasColumn(operationLogTableName, col) {
			missing = append(missing, col)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("table %s missing required column(s): %s", operationLogTableName, strings.Join(missing, ", "))
	}
	return nil
}
