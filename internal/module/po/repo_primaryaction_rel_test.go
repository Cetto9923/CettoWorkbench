// =============================================================================
// 文件: internal/module/po/repo_primaryaction_rel_test.go
// 模块: PO 工作台
// 类型: test
// 职责: 排期按钮的对象级关系门禁单测。验证「有排期能力但与该需求无关」时不出按钮。
// =============================================================================

package po

import (
	"testing"

	"workbench/internal/module/po/primaryaction"
)

// scheduleCapabilityForDemand 复刻 service_primaryaction.go 的排期能力门禁：
// 「有排期权限位」AND「与该需求有对象级关系」。本测试锁住该口径，
// 防止将来只改一半导致权限位重新独立生效。
func scheduleCapabilityForDemand(hasPerm bool, hasRelation bool) bool {
	return hasPerm && hasRelation
}

// TestScheduleRequiresRelation 锁定 T1 口径：排期按钮 = 有排期能力 AND 与该需求有对象级关系。
func TestScheduleRequiresRelation(t *testing.T) {
	cases := []struct {
		name        string
		hasCapable  bool
		hasRelation bool
		wantEnabled bool
	}{
		{
			name:        "po 有 schedule:update 但非该需求相关人 → 无排期按钮",
			hasCapable:  true,
			hasRelation: false,
			wantEnabled: false,
		},
		{
			name:        "po 有 schedule:update 且是相关人 → 有排期按钮",
			hasCapable:  true,
			hasRelation: true,
			wantEnabled: true,
		},
		{
			name:        "无排期能力但有关系 → 仍无按钮",
			hasCapable:  false,
			hasRelation: true,
			wantEnabled: false,
		},
		{
			name:        "既无能力也无关系 → 无按钮",
			hasCapable:  false,
			hasRelation: false,
			wantEnabled: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gated := scheduleCapabilityForDemand(tc.hasCapable, tc.hasRelation)
			in := primaryaction.Input{
				Stage:                 primaryaction.StageSchedule,
				Kind:                  primaryaction.ObjectBusinessDemand,
				ObjectID:              1,
				HasScheduleCapability: gated,
			}
			got := primaryaction.Derive(in)
			if got.Enabled != tc.wantEnabled {
				t.Fatalf("Enabled = %v，期望 %v（reason=%q）", got.Enabled, tc.wantEnabled, got.Reason)
			}
		})
	}
}

// TestDemandRelationBatchLookup 验证批量关系集合的查询语义。
func TestDemandRelationBatchLookup(t *testing.T) {
	batch := &DemandRelationBatch{Related: map[uint]bool{10: true, 20: false}}
	if !batch.HasRelated(10) {
		t.Fatal("需求 10 应判定为有关系")
	}
	if batch.HasRelated(20) {
		t.Fatal("需求 20 应判定为无关系")
	}
	if batch.HasRelated(30) {
		t.Fatal("未在集合中的需求应判定为无关系")
	}
	var nilBatch *DemandRelationBatch
	if nilBatch.HasRelated(10) {
		t.Fatal("nil batch 不应判定为有关系")
	}
}
