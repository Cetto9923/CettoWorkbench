// =============================================================================
// 文件: internal/model/zentao/lang.go
// 模块: 数据模型
// 类型: model
// 职责: 定义禅道语言定义表 zt_lang 字段与表映射。
// 依赖: 无
// =============================================================================

package model

// ZtLang 禅道语言定义表（zt_lang），用于自定义下拉（如任务类型 typeList）。
type ZtLang struct {
	ID      uint   `gorm:"column:id;primaryKey;autoIncrement;type:mediumint unsigned" json:"id"`
	Lang    string `gorm:"column:lang;type:varchar(30);not null;default:''" json:"lang"`
	Module  string `gorm:"column:module;type:varchar(30);not null;default:''" json:"module"`
	Section string `gorm:"column:section;type:varchar(30);not null;default:''" json:"section"`
	Key     string `gorm:"column:key;type:varchar(60);not null;default:''" json:"key"`
	Value   string `gorm:"column:value;type:text;not null" json:"value"`
}

// TableName 指定表名。
func (ZtLang) TableName() string {
	return "zt_lang"
}
