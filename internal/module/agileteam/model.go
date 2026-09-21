// =============================================================================
// 文件: internal/module/agileteam/model.go
// 模块: 敏捷小组治理
// 类型: model
// 职责: Workbench 自有调整单 / 明细 / 历史 ORM；禅道 zt_teamgroup/zt_team 为正式真源。
// 依赖: 无
// =============================================================================

package agileteam

import (
	"time"

	"gorm.io/gorm"
)

const (
	StatusPending    = "pending"
	StatusConfirmed  = "confirmed"
	StatusRejected   = "rejected"
	StatusSuperseded = "superseded"

	ActionAdd        = "add"
	ActionRemove     = "remove"
	ActionRoleChange = "roleChange"

	EventSubmit  = "submit"
	EventConfirm = "confirm"
	EventReject  = "reject"
	// EventUpdate 历史口径模糊（PO/Manager 与 PMO 编辑共用同一事件类型）。
	// 2026-08-27 B 决策（AGENTS.md §4.7）：拆分两个细粒度事件，便于审计 / 报表 /
	// 任何"待确认" UI 过滤时能直接按事件类型排除 PMO 直接编辑。
	// 字段 length=32（model.History.EventType gorm size），新值仍符合。
	EventUpdate        = "update"         // 兼容历史：未识别主体时回退到此值
	EventUpdateByPMO   = "updatedByPmo"   // PMO（allowGlobal=true）直接编辑，立即生效，不进待确认
	EventUpdateByOwner = "updatedByOwner" // PO / 敏捷教练在自身权限范围内编辑，立即生效，不进待确认
)

// Adjustment 成员调整单头。
type Adjustment struct {
	ID            int64          `gorm:"column:id;primaryKey;autoIncrement"`
	TeamgroupID   uint           `gorm:"column:teamgroupId;not null"`
	AdjustNo      string         `gorm:"column:adjustNo;size:32;not null"`
	Status        string         `gorm:"column:status;size:20;not null;default:pending"`
	Reason        string         `gorm:"column:reason;size:500;not null;default:''"`
	SubmittedBy   string         `gorm:"column:submittedBy;size:30;not null"`
	ConfirmedBy   string         `gorm:"column:confirmedBy;size:30;not null;default:''"`
	ConfirmedDate *time.Time     `gorm:"column:confirmedDate"`
	RejectedBy    string         `gorm:"column:rejectedBy;size:30;not null;default:''"`
	RejectedDate  *time.Time     `gorm:"column:rejectedDate"`
	RejectReason  string         `gorm:"column:rejectReason;size:500;not null;default:''"`
	CreatedBy     string         `gorm:"column:createdBy;size:30;not null;default:''"`
	UpdatedBy     string         `gorm:"column:updatedBy;size:30;not null;default:''"`
	CreatedDate   time.Time      `gorm:"column:createdDate;not null"`
	UpdatedDate   time.Time      `gorm:"column:updatedDate;not null"`
	DeletedAt     gorm.DeletedAt `gorm:"column:deletedAt"`
}

func (Adjustment) TableName() string { return "zt_wb_agileteam_adjustment" }

// AdjustmentItem 成员调整明细。
type AdjustmentItem struct {
	ID             int64          `gorm:"column:id;primaryKey;autoIncrement"`
	AdjustmentID   int64          `gorm:"column:adjustmentId;not null"`
	Account        string         `gorm:"column:account;size:30;not null"`
	ActionType     string         `gorm:"column:actionType;size:20;not null"`
	Role           string         `gorm:"column:role;size:64;not null;default:''"`
	PrevRole       string         `gorm:"column:prevRole;size:64;not null;default:''"`
	AvailableHours float64        `gorm:"column:availableHours;not null;default:0"`
	PrevHours      float64        `gorm:"column:prevHours;not null;default:0"`
	CreatedBy      string         `gorm:"column:createdBy;size:30;not null;default:''"`
	UpdatedBy      string         `gorm:"column:updatedBy;size:30;not null;default:''"`
	CreatedDate    time.Time      `gorm:"column:createdDate;not null"`
	UpdatedDate    time.Time      `gorm:"column:updatedDate;not null"`
	DeletedAt      gorm.DeletedAt `gorm:"column:deletedAt"`
}

func (AdjustmentItem) TableName() string { return "zt_wb_agileteam_adjustment_item" }

// History 敏捷小组操作历史。
type History struct {
	ID           int64     `gorm:"column:id;primaryKey;autoIncrement"`
	TeamgroupID  uint      `gorm:"column:teamgroupId;not null"`
	EventType    string    `gorm:"column:eventType;size:32;not null"`
	AdjustmentID *int64    `gorm:"column:adjustmentId"`
	Summary      string    `gorm:"column:summary;size:500;not null;default:''"`
	Actor        string    `gorm:"column:actor;size:30;not null;default:''"`
	CreatedDate  time.Time `gorm:"column:createdDate;not null"`
}

func (History) TableName() string { return "zt_wb_agileteam_history" }
