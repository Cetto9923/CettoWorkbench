package bootstrap

import (
	"strings"
	"testing"
)

type fakeOperationLogSchema struct {
	tables  map[string]bool
	columns map[string]map[string]bool
}

func (f fakeOperationLogSchema) HasTable(dst any) bool {
	name, ok := dst.(string)
	if !ok {
		return false
	}
	return f.tables[name]
}

func (f fakeOperationLogSchema) HasColumn(dst any, field string) bool {
	name, ok := dst.(string)
	if !ok {
		return false
	}
	return f.columns[name][field]
}

func completeOperationLogColumns() map[string]bool {
	cols := make(map[string]bool, len(operationLogRequiredColumns))
	for _, col := range operationLogRequiredColumns {
		cols[col] = true
	}
	return cols
}

func TestCheckOperationLogSchema_MissingTable(t *testing.T) {
	reader := fakeOperationLogSchema{
		tables:  map[string]bool{},
		columns: map[string]map[string]bool{operationLogTableName: completeOperationLogColumns()},
	}
	err := checkOperationLogSchema(reader)
	if err == nil {
		t.Fatal("expected missing-table error")
	}
	if !strings.Contains(err.Error(), "missing required table "+operationLogTableName) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCheckOperationLogSchema_MissingColumn(t *testing.T) {
	cols := completeOperationLogColumns()
	delete(cols, "createdAt")
	reader := fakeOperationLogSchema{
		tables:  map[string]bool{operationLogTableName: true},
		columns: map[string]map[string]bool{operationLogTableName: cols},
	}
	err := checkOperationLogSchema(reader)
	if err == nil {
		t.Fatal("expected missing-column error")
	}
	if !strings.Contains(err.Error(), "missing required column") || !strings.Contains(err.Error(), "createdAt") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCheckOperationLogSchema_Complete(t *testing.T) {
	cols := completeOperationLogColumns()
	cols["extraUnused"] = true
	reader := fakeOperationLogSchema{
		tables:  map[string]bool{operationLogTableName: true},
		columns: map[string]map[string]bool{operationLogTableName: cols},
	}
	if err := checkOperationLogSchema(reader); err != nil {
		t.Fatalf("expected complete schema to pass: %v", err)
	}
}

func TestEnsureOperationLogSchema_NilDB(t *testing.T) {
	err := ensureOperationLogSchema(nil)
	if err == nil {
		t.Fatal("expected nil database error")
	}
	if !strings.Contains(err.Error(), "database handle is nil") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestOperationLogRequiredColumns_MatchContract(t *testing.T) {
	want := []string{
		"id", "tenantId", "userId", "account", "method", "path",
		"query", "body", "ip", "userAgent", "statusCode", "createdAt",
	}
	if len(operationLogRequiredColumns) != len(want) {
		t.Fatalf("required columns len=%d want=%d", len(operationLogRequiredColumns), len(want))
	}
	for i, col := range want {
		if operationLogRequiredColumns[i] != col {
			t.Fatalf("required column[%d]=%q want %q", i, operationLogRequiredColumns[i], col)
		}
	}
}
