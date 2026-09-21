package po

// doneHistoryActionLabel also covers auxiliary events excluded from the core done list.
func doneHistoryActionLabel(objectType, action string) string {
	if label := formalDoneActions[objectType+":"+action].Label; label != "" {
		return label
	}
	labels := map[string]string{
		"created": "创建", "opened": "创建", "edited": "编辑", "submitted": "提交评审", "submit": "提交评审",
		"withdrawreview": "撤回评审", "reviewpassed": "评审通过", "reviewrejected": "评审驳回",
		"reviewed": "评审", "assigned": "指派", "commented": "添加备注", "closed": "关闭",
		"activated": "激活", "resolved": "解决", "deleted": "删除",
		"linkstory": "关联需求", "unlinkstory": "移除需求关联",
	}
	if label := labels[action]; label != "" {
		return label
	}
	return action
}
