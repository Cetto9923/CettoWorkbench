// =============================================================================
// 文件: internal/model/zentao/demandclarify.go
// 模块: 数据模型
// 类型: model
// 职责: 定义业务需求澄清模型字段与表映射。
// 依赖: 无
// =============================================================================

package model

import (
    "time"
)

// ZtDemandclarify 需求澄清表模型
type ZtDemandclarify struct {
    ID                    int        `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
    Demand                int        `gorm:"column:demand;type:mediumint;not null" json:"demand"`
    Product               string     `gorm:"column:product;type:varchar(255);not null" json:"product"`
    PM                    string     `gorm:"column:PM;type:longtext;not null" json:"pm"`
    SystemClarifyDesc     *string    `gorm:"column:systemClarifyDesc;type:longtext" json:"systemClarifyDesc"`
    StoryEnd              *time.Time `gorm:"column:storyEnd;type:date" json:"storyEnd"`
    DevEnd                *time.Time `gorm:"column:devEnd;type:date" json:"devEnd"`
    TestEnd               *time.Time `gorm:"column:testEnd;type:date" json:"testEnd"`
    UatEnd                *time.Time `gorm:"column:uatEnd;type:date" json:"uatEnd"`
    DemandCompletionDate  *time.Time `gorm:"column:demandCompletionDate;type:date;comment:需求完成日期" json:"demandCompletionDate"`
    AdditionalInfo        string     `gorm:"column:additionalInfo;type:mediumtext;not null" json:"additionalInfo"`
    IsAdditionalInfo      string     `gorm:"column:isAdditionalInfo;type:enum('0','1');default:'1'" json:"isAdditionalInfo"`
}

// TableName 指定表名
func (ZtDemandclarify) TableName() string {
    return "zt_demandclarify"
}