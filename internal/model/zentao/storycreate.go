// =============================================================================
// 文件: internal/model/zentao/storycreate.go
// 模块: 数据模型
// 类型: model
// 职责: 研发需求 zt_story 的插入列映射，只包含排期创建时写入的字段。
// 依赖: 无
// =============================================================================

package model

import "time"

// ZtStoryCreate 是 zt_story 的插入投影。
// 不使用完整 ZtStory，避免 GORM 把未赋值列写成零值。
type ZtStoryCreate struct {
	ID                      uint       `gorm:"column:id;primaryKey;autoIncrement"`
	Product                 uint       `gorm:"column:product"`
	Branch                  string     `gorm:"column:branch"`
	Module                  uint       `gorm:"column:module"`
	Plan                    string     `gorm:"column:plan"`
	Source                  string     `gorm:"column:source"`
	SourceNote              string     `gorm:"column:sourceNote"`
	Title                   string     `gorm:"column:title"`
	Type                    string     `gorm:"column:type"`
	Pri                     int        `gorm:"column:pri"`
	Grade                   int        `gorm:"column:grade"`
	Estimate                float64    `gorm:"column:estimate"`
	Status                  string     `gorm:"column:status"`
	Stage                   string     `gorm:"column:stage"`
	SourceType              string     `gorm:"column:sourceType"`
	FromDemand              uint       `gorm:"column:fromDemand"`
	Version                 int        `gorm:"column:version"`
	OpenedBy                string     `gorm:"column:openedBy"`
	OpenedDate              time.Time  `gorm:"column:openedDate"`
	AssignedTo              string     `gorm:"column:assignedTo"`
	IsMainSystemAssociation string     `gorm:"column:isMainSystemAssociation"`
	EstimateLaunch          *time.Time `gorm:"column:estimateLaunch"`
	DevelopFinish           *time.Time `gorm:"column:developFinish"`
	TestFinish              *time.Time `gorm:"column:testFinish"`
	VerifyPlan              string     `gorm:"column:verifyPlan"`
	Deleted                 string     `gorm:"column:deleted"`
}

// TableName 指定表名。
func (ZtStoryCreate) TableName() string { return "zt_story" }
