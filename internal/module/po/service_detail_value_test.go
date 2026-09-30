// =============================================================================
// 文件: internal/module/po/service_detail_value_test.go
// 模块: PO 工作台
// 类型: test
// 职责: 验证价值模型配置解析、费用区间计算及父需求汇总单测。
// =============================================================================

package po

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

// 1. 开关关闭：整块不出现，JSON 序列化后无 valueModel 字段。
func TestPopulateValueModel_SwitchDisabled(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	repo := NewDemandDetailRepo(gormDB)
	svc := NewDetailService(repo)

	// 配置表返回 enabled = '0'
	mock.ExpectQuery(`SELECT[\s\S]*FROM.*zt_config.*WHERE owner = \? AND module = \? AND section = \?`).
		WithArgs("system", "demand", "demandvalue").
		WillReturnRows(sqlmock.NewRows([]string{"key", "value"}).AddRow("enabled", "0"))
	// noAiCategory 查询
	mock.ExpectQuery(`SELECT[\s\S]*FROM.*zt_config.*WHERE owner = \? AND module = \? AND section = \? AND.*key.*= \?`).
		WithArgs("system", "custom", "clarifyCategoryAIConfig", "noAiCategory").
		WillReturnRows(sqlmock.NewRows([]string{"value"}).AddRow(`["datachange"]`))

	row := &DemandDetailRow{ID: 100, Category: "feature"}
	vm, err := svc.populateValueModel(context.Background(), row)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if vm != nil {
		t.Fatalf("expected nil ValueModel when disabled, got %+v", vm)
	}

	resp := &DemandDetailResp{Success: true, ValueModel: vm}
	b, err := json.Marshal(resp)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != `{"success":true,"mode":"","summary":{"id":"","code":"","demandId":0,"title":"","source":"","sourceNote":"","category":"","bsa":"","duration":"","feedbackedBy":"","proposerName":"","proposerDept":"","proposeDept":"","originator":"","ownerName":"","testOwner":"","product":"","poolName":"","priority":"","status":"","zentaoStatus":"","valueStage":"","valueStageLabel":"","estimateLaunch":"","acceptOwner":"","rdName":"","reviewer":"","currentOwner":"","assignedTo":"","assignedToName":"","leadDept":"","teamGroup":"","flags":{"hang":false,"changing":false,"returning":false},"flagNotice":"","mainSystem":"","mainSystemName":"","moduleName":"","desc":"","verifyPlan":"","estimateDelivery":0,"developFinish":"","testFinish":"","verifyFinish":"","storiesCount":0,"tasksDone":0,"tasksTotal":0,"casesExecuted":0,"casesTotal":0,"bugsUnresolved":0,"acceptanceStatus":"","createdDate":"","editedDate":"","createdBy":"","createdName":"","isCreator":false,"isAssignee":false,"canReview":false,"canWithdrawReview":false,"canEdit":false,"hasReviewed":false,"reviewedCount":0,"editDisabledReason":"","zentaoEditUrl":"","zentaoUrl":""},"config":{"deliveryCycleTargetDays":0}}` {
		// 验证没有 valueModel 键
		var rawMap map[string]interface{}
		_ = json.Unmarshal(b, &rawMap)
		if _, exists := rawMap["valueModel"]; exists {
			t.Fatal("valueModel key should be omitted when nil")
		}
	}
}

// 2. 类别跳过（noAiCategory 名单内需求不估算）。
func TestPopulateValueModel_SkippedCategory(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	repo := NewDemandDetailRepo(gormDB)
	svc := NewDetailService(repo)

	mock.ExpectQuery(`SELECT[\s\S]*FROM.*zt_config.*WHERE owner = \? AND module = \? AND section = \?`).
		WithArgs("system", "demand", "demandvalue").
		WillReturnRows(sqlmock.NewRows([]string{"key", "value"}).
			AddRow("enabled", "1").
			AddRow("costPerMonth", "21750").
			AddRow("intervalMethod", "fixed").
			AddRow("intervalFixedPercent", "30"))
	mock.ExpectQuery(`SELECT[\s\S]*FROM.*zt_config.*WHERE owner = \? AND module = \? AND section = \? AND.*key.*= \?`).
		WithArgs("system", "custom", "clarifyCategoryAIConfig", "noAiCategory").
		WillReturnRows(sqlmock.NewRows([]string{"value"}).AddRow(`["datachange","dataexport"]`))

	row := &DemandDetailRow{ID: 101, Category: "datachange", Parent: 0}
	vm, err := svc.populateValueModel(context.Background(), row)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if vm == nil {
		t.Fatal("expected non-nil ValueModel when enabled")
	}
	if !vm.Skipped {
		t.Fatal("expected Skipped to be true")
	}
	if vm.DemandValue != nil {
		t.Fatalf("skipped category should have nil demandValue, got %v", *vm.DemandValue)
	}
	if vm.CostAvailable {
		t.Fatal("skipped category should have CostAvailable=false")
	}
	if vm.CostUnavailableReason != "该需求类别配置为无AI故事点，跳过需求价值估算。" {
		t.Fatalf("unexpected reason: %q", vm.CostUnavailableReason)
	}
}

// 3. 普通需求：未估算场景（demandValue 为空或 <= 0）。
func TestPopulateValueModel_Unestimated(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	repo := NewDemandDetailRepo(gormDB)
	svc := NewDetailService(repo)

	mock.ExpectQuery(`SELECT[\s\S]*FROM.*zt_config.*WHERE owner = \? AND module = \? AND section = \?`).
		WithArgs("system", "demand", "demandvalue").
		WillReturnRows(sqlmock.NewRows([]string{"key", "value"}).
			AddRow("enabled", "1").
			AddRow("costPerMonth", "21750").
			AddRow("intervalMethod", "fixed").
			AddRow("intervalFixedPercent", "30"))
	mock.ExpectQuery(`SELECT[\s\S]*FROM.*zt_config.*WHERE owner = \? AND module = \? AND section = \? AND.*key.*= \?`).
		WithArgs("system", "custom", "clarifyCategoryAIConfig", "noAiCategory").
		WillReturnRows(sqlmock.NewRows([]string{"value"}).AddRow(`["datachange"]`))
	// 检查列是否存在
	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM information_schema\.COLUMNS`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	// 查询单个需求价值（库中为空）
	mock.ExpectQuery(`SELECT.*demandValue.*FROM.*zt_demand.*WHERE id = \? AND deleted = '0'`).
		WithArgs(uint(102), 1).
		WillReturnRows(sqlmock.NewRows([]string{"demandValue", "demandValueDate"}))

	row := &DemandDetailRow{ID: 102, Category: "feature", Parent: 0}
	vm, err := svc.populateValueModel(context.Background(), row)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if vm == nil {
		t.Fatal("expected non-nil ValueModel")
	}
	if vm.DemandValue != nil {
		t.Fatalf("unestimated should have nil demandValue, got %v", *vm.DemandValue)
	}
	if vm.CostAvailable {
		t.Fatal("unestimated should have CostAvailable=false")
	}
	if vm.CostUnavailableReason != "暂未估算（澄清完成后自动计算）" {
		t.Fatalf("unexpected reason: %q", vm.CostUnavailableReason)
	}
}

// 4. 普通需求：已估算且 fixed 方式费用区间正确计算。
func TestPopulateValueModel_NormalDemand_FixedCalculation(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	repo := NewDemandDetailRepo(gormDB)
	svc := NewDetailService(repo)

	mock.ExpectQuery(`SELECT[\s\S]*FROM.*zt_config.*WHERE owner = \? AND module = \? AND section = \?`).
		WithArgs("system", "demand", "demandvalue").
		WillReturnRows(sqlmock.NewRows([]string{"key", "value"}).
			AddRow("enabled", "1").
			AddRow("costPerMonth", "21750").
			AddRow("intervalMethod", "fixed").
			AddRow("intervalFixedPercent", "30"))
	mock.ExpectQuery(`SELECT[\s\S]*FROM.*zt_config.*WHERE owner = \? AND module = \? AND section = \? AND.*key.*= \?`).
		WithArgs("system", "custom", "clarifyCategoryAIConfig", "noAiCategory").
		WillReturnRows(sqlmock.NewRows([]string{"value"}).AddRow(`["datachange"]`))
	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM information_schema\.COLUMNS`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	val := 10.0000
	mock.ExpectQuery(`SELECT.*demandValue.*FROM.*zt_demand.*WHERE id = \? AND deleted = '0'`).
		WithArgs(uint(103), 1).
		WillReturnRows(sqlmock.NewRows([]string{"demandValue", "demandValueDate"}).
			AddRow(val, time.Date(2026, 9, 30, 14, 0, 0, 0, time.UTC)))

	row := &DemandDetailRow{ID: 103, Category: "feature", Parent: 0}
	vm, err := svc.populateValueModel(context.Background(), row)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if vm == nil || vm.DemandValue == nil {
		t.Fatalf("expected non-nil demandValue, got %+v", vm)
	}
	if *vm.DemandValue != 10.0000 {
		t.Fatalf("want 10.0000, got %f", *vm.DemandValue)
	}
	if !vm.CostAvailable {
		t.Fatalf("expected CostAvailable=true, reason=%q", vm.CostUnavailableReason)
	}
	if vm.IntervalKind != "fixed" || vm.IntervalPercent == nil || *vm.IntervalPercent != 30.0 {
		t.Fatalf("unexpected interval config: kind=%s, percent=%v", vm.IntervalKind, vm.IntervalPercent)
	}
	// 验证金额：
	// valLow = round(10.0 * 0.7, 4) = 7.0000
	// valHigh = round(10.0 * 1.3, 4) = 13.0000
	// costLow = round(7.0000 * 21750 / 21.75, 2) = 7000.00
	// costHigh = round(13.0000 * 21750 / 21.75, 2) = 13000.00
	if vm.CostLow == nil || *vm.CostLow != 7000.00 {
		t.Fatalf("want costLow 7000.00, got %v", *vm.CostLow)
	}
	if vm.CostHigh == nil || *vm.CostHigh != 13000.00 {
		t.Fatalf("want costHigh 13000.00, got %v", *vm.CostHigh)
	}
}

// 5. holdout 方式费用区间：工作台不自造倍率，明确标记不可用。
func TestPopulateValueModel_Holdout_Unavailable(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	repo := NewDemandDetailRepo(gormDB)
	svc := NewDetailService(repo)

	mock.ExpectQuery(`SELECT[\s\S]*FROM.*zt_config.*WHERE owner = \? AND module = \? AND section = \?`).
		WithArgs("system", "demand", "demandvalue").
		WillReturnRows(sqlmock.NewRows([]string{"key", "value"}).
			AddRow("enabled", "1").
			AddRow("costPerMonth", "21750").
			AddRow("intervalMethod", "holdout"))
	mock.ExpectQuery(`SELECT[\s\S]*FROM.*zt_config.*WHERE owner = \? AND module = \? AND section = \? AND.*key.*= \?`).
		WithArgs("system", "custom", "clarifyCategoryAIConfig", "noAiCategory").
		WillReturnRows(sqlmock.NewRows([]string{"value"}).AddRow(`["datachange"]`))
	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM information_schema\.COLUMNS`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	val := 10.0000
	mock.ExpectQuery(`SELECT.*demandValue.*FROM.*zt_demand.*WHERE id = \? AND deleted = '0'`).
		WithArgs(uint(104), 1).
		WillReturnRows(sqlmock.NewRows([]string{"demandValue", "demandValueDate"}).
			AddRow(val, time.Date(2026, 9, 30, 14, 0, 0, 0, time.UTC)))

	row := &DemandDetailRow{ID: 104, Category: "feature", Parent: 0}
	vm, err := svc.populateValueModel(context.Background(), row)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if vm == nil || vm.DemandValue == nil {
		t.Fatalf("expected non-nil demandValue, got %+v", vm)
	}
	if vm.CostAvailable {
		t.Fatal("holdout should have CostAvailable=false")
	}
	if vm.CostUnavailableReason != "区间需在禅道查看" {
		t.Fatalf("want reason '区间需在禅道查看', got %q", vm.CostUnavailableReason)
	}
	if vm.CostLow != nil || vm.CostHigh != nil {
		t.Fatalf("costLow/costHigh should be nil for holdout, got %v, %v", vm.CostLow, vm.CostHigh)
	}
}

// 6. 未配置每人月费用：费用区间标记不可用。
func TestPopulateValueModel_CostPerMonthNotConfigured(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	repo := NewDemandDetailRepo(gormDB)
	svc := NewDetailService(repo)

	mock.ExpectQuery(`SELECT[\s\S]*FROM.*zt_config.*WHERE owner = \? AND module = \? AND section = \?`).
		WithArgs("system", "demand", "demandvalue").
		WillReturnRows(sqlmock.NewRows([]string{"key", "value"}).
			AddRow("enabled", "1").
			AddRow("costPerMonth", ""). // 未配置
			AddRow("intervalMethod", "fixed").
			AddRow("intervalFixedPercent", "30"))
	mock.ExpectQuery(`SELECT[\s\S]*FROM.*zt_config.*WHERE owner = \? AND module = \? AND section = \? AND.*key.*= \?`).
		WithArgs("system", "custom", "clarifyCategoryAIConfig", "noAiCategory").
		WillReturnRows(sqlmock.NewRows([]string{"value"}).AddRow(`["datachange"]`))
	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM information_schema\.COLUMNS`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	val := 5.0000
	mock.ExpectQuery(`SELECT.*demandValue.*FROM.*zt_demand.*WHERE id = \? AND deleted = '0'`).
		WithArgs(uint(105), 1).
		WillReturnRows(sqlmock.NewRows([]string{"demandValue", "demandValueDate"}).
			AddRow(val, time.Date(2026, 9, 30, 14, 0, 0, 0, time.UTC)))

	row := &DemandDetailRow{ID: 105, Category: "feature", Parent: 0}
	vm, err := svc.populateValueModel(context.Background(), row)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if vm == nil {
		t.Fatal("expected non-nil ValueModel")
	}
	if vm.CostPerMonthConfigured {
		t.Fatal("expected CostPerMonthConfigured=false")
	}
	if vm.CostAvailable {
		t.Fatal("expected CostAvailable=false")
	}
	if vm.CostUnavailableReason != "未配置每人月费用（后台 › 功能配置 › 需求池 › 每人月费用配置）" {
		t.Fatalf("unexpected reason: %q", vm.CostUnavailableReason)
	}
}

// 7. 父需求合计（含子需求类别豁免与部分估算）。
func TestPopulateValueModel_ParentDemand_Aggregation(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	repo := NewDemandDetailRepo(gormDB)
	svc := NewDetailService(repo)

	mock.ExpectQuery(`SELECT[\s\S]*FROM.*zt_config.*WHERE owner = \? AND module = \? AND section = \?`).
		WithArgs("system", "demand", "demandvalue").
		WillReturnRows(sqlmock.NewRows([]string{"key", "value"}).
			AddRow("enabled", "1").
			AddRow("costPerMonth", "21750").
			AddRow("intervalMethod", "fixed").
			AddRow("intervalFixedPercent", "20"))
	mock.ExpectQuery(`SELECT[\s\S]*FROM.*zt_config.*WHERE owner = \? AND module = \? AND section = \? AND.*key.*= \?`).
		WithArgs("system", "custom", "clarifyCategoryAIConfig", "noAiCategory").
		WillReturnRows(sqlmock.NewRows([]string{"value"}).AddRow(`["datachange"]`))
	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM information_schema\.COLUMNS`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	// 父需求 ID=200，Parent=-1
	// 子需求 3 条：
	// 子1: category=feature, demandValue=3.2000 (参与，已估算)
	// 子2: category=feature, demandValue=0 (参与，未估算)
	// 子3: category=datachange, demandValue=10.0 (豁免，不参与)
	v1 := 3.2000
	v2 := 0.0
	v3 := 10.0
	mock.ExpectQuery(`SELECT.*FROM.*zt_demand.*WHERE parent = \? AND deleted = '0'`).
		WithArgs(uint(200)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "category", "demandValue"}).
			AddRow(201, "feature", &v1).
			AddRow(202, "feature", &v2).
			AddRow(203, "datachange", &v3))

	row := &DemandDetailRow{ID: 200, Category: "datachange", Parent: -1} // 父自身类别不影响汇总
	vm, err := svc.populateValueModel(context.Background(), row)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if vm == nil {
		t.Fatal("expected non-nil ValueModel")
	}
	if !vm.IsParent || !vm.ChildSum {
		t.Fatalf("expected IsParent=true, ChildSum=true, got %+v", vm)
	}
	if vm.ChildEligible != 2 {
		t.Fatalf("want ChildEligible=2, got %d", vm.ChildEligible)
	}
	if vm.ChildEstimated != 1 {
		t.Fatalf("want ChildEstimated=1, got %d", vm.ChildEstimated)
	}
	if vm.DemandValue == nil || *vm.DemandValue != 3.2000 {
		t.Fatalf("want sumValue 3.2000, got %v", vm.DemandValue)
	}
	if !vm.CostAvailable {
		t.Fatalf("expected CostAvailable=true, got reason=%q", vm.CostUnavailableReason)
	}
	// 验证金额：
	// valLow = round(3.2000 * 0.8, 4) = 2.5600
	// valHigh = round(3.2000 * 1.2, 4) = 3.8400
	// costLow = round(2.5600 * 21750 / 21.75, 2) = 2560.00
	// costHigh = round(3.8400 * 21750 / 21.75, 2) = 3840.00
	if vm.CostLow == nil || *vm.CostLow != 2560.00 {
		t.Fatalf("want costLow 2560.00, got %v", vm.CostLow)
	}
	if vm.CostHigh == nil || *vm.CostHigh != 3840.00 {
		t.Fatalf("want costHigh 3840.00, got %v", vm.CostHigh)
	}
}

// 8. 父需求所有子需求均未估算。
func TestPopulateValueModel_ParentDemand_NoneEstimated(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	repo := NewDemandDetailRepo(gormDB)
	svc := NewDetailService(repo)

	mock.ExpectQuery(`SELECT[\s\S]*FROM.*zt_config.*WHERE owner = \? AND module = \? AND section = \?`).
		WithArgs("system", "demand", "demandvalue").
		WillReturnRows(sqlmock.NewRows([]string{"key", "value"}).
			AddRow("enabled", "1").
			AddRow("costPerMonth", "21750").
			AddRow("intervalMethod", "fixed").
			AddRow("intervalFixedPercent", "20"))
	mock.ExpectQuery(`SELECT[\s\S]*FROM.*zt_config.*WHERE owner = \? AND module = \? AND section = \? AND.*key.*= \?`).
		WithArgs("system", "custom", "clarifyCategoryAIConfig", "noAiCategory").
		WillReturnRows(sqlmock.NewRows([]string{"value"}).AddRow(`["datachange"]`))
	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM information_schema\.COLUMNS`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	mock.ExpectQuery(`SELECT.*FROM.*zt_demand.*WHERE parent = \? AND deleted = '0'`).
		WithArgs(uint(210)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "category", "demandValue"}).
			AddRow(211, "feature", nil).
			AddRow(212, "feature", nil))

	row := &DemandDetailRow{ID: 210, Category: "feature", Parent: -1}
	vm, err := svc.populateValueModel(context.Background(), row)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if vm == nil {
		t.Fatal("expected non-nil ValueModel")
	}
	if !vm.IsParent || !vm.ChildSum {
		t.Fatalf("expected IsParent=true, ChildSum=true, got %+v", vm)
	}
	if vm.ChildEligible != 2 || vm.ChildEstimated != 0 {
		t.Fatalf("want eligible=2, estimated=0, got %d, %d", vm.ChildEligible, vm.ChildEstimated)
	}
	if vm.DemandValue != nil {
		t.Fatalf("expected nil DemandValue when 0 estimated, got %v", *vm.DemandValue)
	}
	if vm.CostAvailable {
		t.Fatal("expected CostAvailable=false")
	}
	if vm.CostUnavailableReason != "子需求暂未估算" {
		t.Fatalf("want reason '子需求暂未估算', got %q", vm.CostUnavailableReason)
	}
}
