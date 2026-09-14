// =============================================================================
// 文件: internal/model/zentao/testtask.go
// 模块: 数据模型
// 类型: model
// 职责: 定义禅道测试单表 zt_testtask 字段与表映射。
// 依赖: 无
// =============================================================================

package model

import "time"

// ZtTesttask 测试单表模型（zt_testtask）。
type ZtTesttask struct {
	ID               uint       `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Project          uint       `gorm:"column:project;type:mediumint unsigned;not null;default:0" json:"project"`
	Product          uint       `gorm:"column:product;type:mediumint unsigned;not null;default:0" json:"product"`
	Name             string     `gorm:"column:name;type:char(90);not null;default:''" json:"name"`
	Execution        uint       `gorm:"column:execution;type:mediumint unsigned;not null;default:0" json:"execution"`
	Build            string     `gorm:"column:build;type:char(30);not null;default:''" json:"build"`
	Type             string     `gorm:"column:type;type:varchar(255);not null;default:''" json:"type"`
	Owner            string     `gorm:"column:owner;type:varchar(30);not null;default:''" json:"owner"`
	Pri              uint8      `gorm:"column:pri;type:tinyint unsigned;not null;default:0" json:"pri"`
	Begin            *time.Time `gorm:"column:begin;type:date" json:"begin"`
	End              *time.Time `gorm:"column:end;type:date" json:"end"`
	RealBegan        *time.Time `gorm:"column:realBegan;type:date" json:"realBegan"`
	RealFinishedDate *time.Time `gorm:"column:realFinishedDate;type:datetime" json:"realFinishedDate"`
	Mailto           string     `gorm:"column:mailto;type:text" json:"mailto"`
	Desc             string     `gorm:"column:desc;type:mediumtext" json:"desc"`
	Report           string     `gorm:"column:report;type:text" json:"report"`
	Status           string     `gorm:"column:status;type:enum('blocked','doing','wait','done');not null;default:'wait'" json:"status"`
	Testreport       uint       `gorm:"column:testreport;type:mediumint unsigned;not null;default:0" json:"testreport"`
	Auto             string     `gorm:"column:auto;type:varchar(10);not null;default:'no'" json:"auto"`
	SubStatus        string     `gorm:"column:subStatus;type:varchar(30);not null;default:''" json:"subStatus"`
	CreatedBy        string     `gorm:"column:createdBy;type:varchar(30);not null;default:''" json:"createdBy"`
	CreatedDate      *time.Time `gorm:"column:createdDate;type:datetime" json:"createdDate"`
	Deleted          string     `gorm:"column:deleted;type:enum('0','1');not null;default:'0'" json:"deleted"`
	Members          string     `gorm:"column:members;type:text" json:"members"`
}

// TableName 指定表名。
func (ZtTesttask) TableName() string {
	return "zt_testtask"
}
