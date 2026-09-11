// =============================================================================
// 文件: internal/module/build/searchfields.go
// 模块: 版本管理
// 类型: action
// 职责: 关联需求搜索字段定义与静态选项（对齐禅道 product/search + story 语言包）。
// 依赖: 无
// =============================================================================

package build

// StorySearchFieldDefs 返回 linkStory 可用搜索字段（不含 product）。
func StorySearchFieldDefs(includeBranch bool) []SearchFieldDef {
	defs := []SearchFieldDef{
		{Key: "title", Label: "需求名称", Control: "input", Operator: "include"},
		{Key: "id", Label: "需求编号", Control: "input", Operator: "="},
		{Key: "keywords", Label: "关键词", Control: "input", Operator: "include"},
		{Key: "status", Label: "当前状态", Control: "select", Operator: "=", OptionsKey: "status"},
		{Key: "pri", Label: "优先级", Control: "select", Operator: "=", OptionsKey: "pri"},
		{Key: "module", Label: "所属模块", Control: "select", Operator: "belong", OptionsKey: "modules"},
		{Key: "stage", Label: "阶段", Control: "select", Operator: "=", OptionsKey: "stage"},
	}
	if includeBranch {
		defs = append(defs, SearchFieldDef{Key: "branch", Label: "分支/平台", Control: "select", Operator: "=", OptionsKey: "branches"})
	}
	defs = append(defs,
		SearchFieldDef{Key: "grade", Label: "层级", Control: "input", Operator: "="},
		SearchFieldDef{Key: "plan", Label: "所属计划", Control: "select", Operator: "=", OptionsKey: "plans"},
		SearchFieldDef{Key: "estimate", Label: "预计工时", Control: "input", Operator: "="},
		SearchFieldDef{Key: "source", Label: "来源", Control: "select", Operator: "=", OptionsKey: "source"},
		SearchFieldDef{Key: "sourceNote", Label: "来源备注", Control: "input", Operator: "include"},
		SearchFieldDef{Key: "fromBug", Label: "来源Bug", Control: "input", Operator: "="},
		SearchFieldDef{Key: "category", Label: "类别", Control: "select", Operator: "=", OptionsKey: "category"},
		SearchFieldDef{Key: "openedBy", Label: "由谁创建", Control: "input", Operator: "="},
		SearchFieldDef{Key: "reviewedBy", Label: "由谁评审", Control: "input", Operator: "include"},
		SearchFieldDef{Key: "result", Label: "评审结果", Control: "select", Operator: "=", OptionsKey: "result"},
		SearchFieldDef{Key: "assignedTo", Label: "指派给", Control: "input", Operator: "="},
		SearchFieldDef{Key: "closedBy", Label: "由谁关闭", Control: "input", Operator: "="},
		SearchFieldDef{Key: "lastEditedBy", Label: "最后修改", Control: "input", Operator: "="},
		SearchFieldDef{Key: "mailto", Label: "抄送给", Control: "input", Operator: "include"},
		SearchFieldDef{Key: "closedReason", Label: "关闭原因", Control: "select", Operator: "=", OptionsKey: "closedReason"},
		SearchFieldDef{Key: "version", Label: "版本号", Control: "input", Operator: ">="},
		SearchFieldDef{Key: "openedDate", Label: "创建日期", Control: "date", Operator: "="},
		SearchFieldDef{Key: "reviewedDate", Label: "评审时间", Control: "date", Operator: "="},
		SearchFieldDef{Key: "assignedDate", Label: "指派日期", Control: "date", Operator: "="},
		SearchFieldDef{Key: "closedDate", Label: "关闭日期", Control: "date", Operator: "="},
		SearchFieldDef{Key: "lastEditedDate", Label: "最后修改日期", Control: "date", Operator: "="},
		SearchFieldDef{Key: "activatedDate", Label: "激活日期", Control: "date", Operator: "="},
		SearchFieldDef{Key: "estimateLaunch", Label: "预计上线时间", Control: "date", Operator: "="},
		SearchFieldDef{Key: "deliverDate", Label: "交付日期", Control: "date", Operator: "="},
		SearchFieldDef{Key: "isMainSystemAssociation", Label: "是否是主系统研发需求", Control: "select", Operator: "=", OptionsKey: "yesNo"},
		SearchFieldDef{Key: "isCarReview", Label: "快速评审", Control: "select", Operator: "=", OptionsKey: "yesNo"},
	)
	return defs
}

// StaticSearchOptions 静态枚举选项。
func StaticSearchOptions() map[string][]SearchOption {
	return map[string][]SearchOption{
		"status": {
			{Value: "", Label: ""},
			{Value: "draft", Label: "草稿"},
			{Value: "reviewing", Label: "评审中"},
			{Value: "active", Label: "激活"},
			{Value: "changing", Label: "变更中"},
			{Value: "closed", Label: "已关闭"},
			{Value: "launched", Label: "已投产"},
			{Value: "developing", Label: "研发中"},
		},
		"stage": {
			{Value: "", Label: ""},
			{Value: "wait", Label: "未开始"},
			{Value: "planned", Label: "已计划"},
			{Value: "projected", Label: "研发立项"},
			{Value: "designing", Label: "设计中"},
			{Value: "designed", Label: "设计完毕"},
			{Value: "developing", Label: "研发中"},
			{Value: "developed", Label: "研发完毕"},
			{Value: "testing", Label: "测试中"},
			{Value: "tested", Label: "测试完毕"},
			{Value: "verified", Label: "已验收"},
			{Value: "rejected", Label: "验收失败"},
			{Value: "delivering", Label: "交付中"},
			{Value: "delivered", Label: "已交付"},
			{Value: "released", Label: "已发布"},
			{Value: "closed", Label: "已关闭"},
		},
		"pri": {
			{Value: "", Label: ""},
			{Value: "1", Label: "1"},
			{Value: "2", Label: "2"},
			{Value: "3", Label: "3"},
			{Value: "4", Label: "4"},
		},
		"source": {
			{Value: "", Label: ""},
			{Value: "customer", Label: "客户"},
			{Value: "user", Label: "用户"},
			{Value: "po", Label: "产品经理"},
			{Value: "market", Label: "市场"},
			{Value: "service", Label: "客服"},
			{Value: "operation", Label: "运营"},
			{Value: "support", Label: "技术支持"},
			{Value: "competitor", Label: "竞争对手"},
			{Value: "partner", Label: "合作伙伴"},
			{Value: "dev", Label: "开发人员"},
			{Value: "tester", Label: "测试人员"},
			{Value: "bug", Label: "Bug"},
			{Value: "forum", Label: "论坛"},
			{Value: "other", Label: "其他"},
		},
		"category": {
			{Value: "", Label: ""},
			{Value: "feature", Label: "功能"},
			{Value: "interface", Label: "接口"},
			{Value: "performance", Label: "性能"},
			{Value: "safe", Label: "安全"},
			{Value: "experience", Label: "体验"},
			{Value: "improve", Label: "改进"},
			{Value: "other", Label: "其他"},
		},
		"closedReason": {
			{Value: "", Label: ""},
			{Value: "done", Label: "已完成"},
			{Value: "subdivided", Label: "已拆分"},
			{Value: "duplicate", Label: "重复"},
			{Value: "postponed", Label: "延期"},
			{Value: "willnotdo", Label: "不做"},
			{Value: "cancel", Label: "已取消"},
			{Value: "bydesign", Label: "设计如此"},
		},
		"result": {
			{Value: "", Label: ""},
			{Value: "pass", Label: "确认通过"},
			{Value: "revert", Label: "撤销变更"},
			{Value: "clarify", Label: "有待明确"},
			{Value: "reject", Label: "拒绝"},
		},
		"yesNo": {
			{Value: "", Label: ""},
			{Value: "1", Label: "是"},
			{Value: "0", Label: "否"},
		},
		"operators": {
			{Value: "=", Label: "="},
			{Value: "!=", Label: "!="},
			{Value: ">", Label: ">"},
			{Value: ">=", Label: ">="},
			{Value: "<", Label: "<"},
			{Value: "<=", Label: "<="},
			{Value: "include", Label: "包含"},
			{Value: "notinclude", Label: "不包含"},
			{Value: "belong", Label: "从属于"},
			{Value: "between", Label: "介于"},
		},
	}
}

// FieldControl 查字段控件类型。
func FieldControl(field string, defs []SearchFieldDef) string {
	for _, d := range defs {
		if d.Key == field {
			return d.Control
		}
	}
	return "input"
}

// FieldOptionsKey 查字段选项集 key。
func FieldOptionsKey(field string, defs []SearchFieldDef) string {
	for _, d := range defs {
		if d.Key == field {
			return d.OptionsKey
		}
	}
	return ""
}
