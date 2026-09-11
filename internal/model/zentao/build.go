// =============================================================================
// 文件: internal/model/zentao/build.go
// 模块: 数据模型
// 类型: model
// 职责: 定义禅道版本表 zt_build 字段与表映射。
// 依赖: 无
// =============================================================================

package model

import "time"

// ZtBuild 版本表模型（zt_build）。
type ZtBuild struct {
	ID          uint       `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Project     uint       `gorm:"column:project;type:mediumint unsigned;not null;default:0" json:"project"`
	Product     uint       `gorm:"column:product;type:mediumint unsigned;not null;default:0" json:"product"`
	Branch      string     `gorm:"column:branch;type:varchar(255);not null;default:'0'" json:"branch"`
	Execution   uint       `gorm:"column:execution;type:mediumint unsigned;not null;default:0" json:"execution"`
	Builds      string     `gorm:"column:builds;type:varchar(255);not null;default:''" json:"builds"`
	Name        string     `gorm:"column:name;type:char(150);not null;default:''" json:"name"`
	SCMPath     string     `gorm:"column:scmPath;type:char(255);not null;default:''" json:"scmPath"`
	FilePath    string     `gorm:"column:filePath;type:char(255);not null;default:''" json:"filePath"`
	Date        *time.Time `gorm:"column:date;type:date" json:"date"`
	Stories     string     `gorm:"column:stories;type:text" json:"stories"`
	Bugs        string     `gorm:"column:bugs;type:text" json:"bugs"`
	Builder     string     `gorm:"column:builder;type:char(30);not null;default:''" json:"builder"`
	Desc        string     `gorm:"column:desc;type:mediumtext" json:"desc"`
	CreatedBy   string     `gorm:"column:createdBy;type:varchar(30);not null;default:''" json:"createdBy"`
	CreatedDate *time.Time `gorm:"column:createdDate;type:datetime" json:"createdDate"`
	Deleted     string     `gorm:"column:deleted;type:enum('0','1');not null;default:'0'" json:"deleted"`
}

// TableName 指定表名。
func (ZtBuild) TableName() string {
	return "zt_build"
}
