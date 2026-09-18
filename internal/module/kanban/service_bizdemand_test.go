// =============================================================================
// 文件: internal/module/kanban/service_bizdemand_test.go
// 模块: 工作看板
// 类型: test
// 职责: 验证独立研发需求价值流阶段派生与 ID 归一化逻辑。
// 依赖: internal/module/kanban
// =============================================================================

package kanban

import (
	"testing"
	"time"
)

func TestDeriveStoryStage(t *testing.T) {
	now := time.Now()
	past := now.AddDate(0, 0, -1)
	future := now.AddDate(0, 0, 5)
	zeroTime := time.Time{}

	tests := []struct {
		name          string
		status        string
		stage         string
		developFinish *time.Time
		testFinish    *time.Time
		verifyFinish  *time.Time
		deliverDate   *time.Time
		want          string
	}{
		{
			name:        "过去交付日期应落入交付/评价",
			deliverDate: &past,
			want:        "交付/评价",
		},
		{
			name:        "今天交付日期应落入交付/评价",
			deliverDate: &now,
			want:        "交付/评价",
		},
		{
			name:        "零日期交付不应误判为已交付",
			deliverDate: &zeroTime,
			status:      "active",
			stage:       "wait",
			want:        "排期",
		},
		{
			name:   "launched 状态应落入交付/评价",
			status: "launched",
			want:   "交付/评价",
		},
		{
			name:  "released stage 应落入交付/评价",
			stage: "released",
			want:  "交付/评价",
		},
		{
			name:  "testing stage 应落入联调/验收",
			stage: "testing",
			want:  "联调/验收",
		},
		{
			name:          "三段完成时间均有效填写应落入联调/验收",
			status:        "active",
			stage:         "wait",
			developFinish: &past,
			testFinish:    &past,
			verifyFinish:  &past,
			want:          "联调/验收",
		},
		{
			name:          "部分完成时间有效仍应落入研发或排期",
			status:        "active",
			stage:         "wait",
			developFinish: &past,
			testFinish:    &past,
			verifyFinish:  nil,
			want:          "排期",
		},
		{
			name:   "developing 状态应落入研发/提测",
			status: "developing",
			stage:  "wait",
			want:   "研发/提测",
		},
		{
			name:  "developing stage 应落入研发/提测",
			stage: "developing",
			want:  "研发/提测",
		},
		{
			name:   "默认无关键日期且未启动应落入排期",
			status: "active",
			stage:  "wait",
			want:   "排期",
		},
		{
			name:        "未来交付日期未启动应落入排期",
			status:      "active",
			stage:       "wait",
			deliverDate: &future,
			want:        "排期",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := deriveStoryStage(tt.status, tt.stage, tt.developFinish, tt.testFinish, tt.verifyFinish, tt.deliverDate)
			if got != tt.want {
				t.Errorf("deriveStoryStage() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDemandItemKey(t *testing.T) {
	tests := []struct {
		kind string
		id   string
		want string
	}{
		{kind: "story", id: "U123", want: "story:123"},
		{kind: "story", id: "123", want: "story:123"},
		{kind: "demand", id: "US63407", want: "demand:63407"},
		{kind: "demand", id: "63407", want: "demand:63407"},
		{kind: "demand", id: "REQ-888", want: "demand:888"},
	}

	for _, tt := range tests {
		got := demandItemKey(tt.kind, tt.id)
		if got != tt.want {
			t.Errorf("demandItemKey(%s, %s) = %s, want %s", tt.kind, tt.id, got, tt.want)
		}
	}
}

func TestRawWorkItemID(t *testing.T) {
	tests := []struct {
		id   string
		want string
	}{
		{id: "US63407", want: "63407"},
		{id: "U68485", want: "68485"},
		{id: "#US123", want: "123"},
		{id: "REQ-999", want: "999"},
		{id: "SUB-555", want: "555"},
		{id: "RD-666", want: "666"},
		{id: "777", want: "777"},
		{id: "  US888  ", want: "888"},
		{id: "us123", want: "123"},
		{id: "USUS123", want: "US123"},
		{id: "abc", want: "abc"},
		{id: "#", want: ""},
		{id: "", want: ""},
	}

	for _, tt := range tests {
		got := rawWorkItemID(tt.id)
		if got != tt.want {
			t.Errorf("rawWorkItemID(%s) = %s, want %s", tt.id, got, tt.want)
		}
	}
}

func TestComputeDemandSummary(t *testing.T) {
	items := []BizDemandItem{
		{ValueStream: "受理 / 澄清", ZentaoStatus: "wait"},
		{ValueStream: "排期", ZentaoStatus: "wait"},
		{ValueStream: "研发 / 提测", ZentaoStatus: "suspended"},
		{ValueStream: "交付 / 评价", ZentaoStatus: "refuse"},
	}
	sum := computeDemandSummary(items)
	if sum.Clarify != 1 {
		t.Errorf("Clarify = %d, want 1", sum.Clarify)
	}
	if sum.Schedule != 1 {
		t.Errorf("Schedule = %d, want 1", sum.Schedule)
	}
	if sum.Blocked != 2 {
		t.Errorf("Blocked = %d, want 2", sum.Blocked)
	}
}

func TestComputeMemberCounts(t *testing.T) {
	items := []BizDemandItem{
		{OwnerAccount: "user1"},
		{OwnerAccount: "user1"},
		{OwnerAccount: "user2"},
		{OwnerAccount: ""},
	}
	counts := computeMemberCounts(items)
	if counts["user1"] != 2 {
		t.Errorf("user1 = %d, want 2", counts["user1"])
	}
	if counts["user2"] != 1 {
		t.Errorf("user2 = %d, want 1", counts["user2"])
	}
	if _, ok := counts[""]; ok {
		t.Errorf("empty account should not be in map")
	}
}
