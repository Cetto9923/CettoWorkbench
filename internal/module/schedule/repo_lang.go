// =============================================================================
// 文件: internal/module/schedule/repo_lang.go
// 模块: 排期工作台
// 类型: action
// 职责: 从禅道 zt_lang 读取语言定义下拉（如任务类型 typeList）。
// 依赖: internal/module/schedule/form.go
// =============================================================================

package schedule

import (
	"context"
	"strings"
)

type ztLangTypeListRow struct {
	Key   string `gorm:"column:key"`
	Value string `gorm:"column:value"`
	Lang  string `gorm:"column:lang"`
}

// ListTaskTypeOptions 从 zt_lang 读取任务类型（module=task, section=typeList）。
// 对齐禅道：lang IN (zh-cn, all)，同 key 以 zh-cn 覆盖 all；跳过空 key。
func (r *Repo) ListTaskTypeOptions(ctx context.Context) ([]TaskTypeOption, error) {
	var rows []ztLangTypeListRow
	const query = `
SELECT ` + "`key`" + `, value, lang
FROM zt_lang
WHERE module = ?
  AND section = ?
  AND lang IN (?, ?)
ORDER BY CASE WHEN lang = 'all' THEN 0 ELSE 1 END, id
`
	if err := r.db.WithContext(ctx).Raw(query, "task", "typeList", "all", "zh-cn").Scan(&rows).Error; err != nil {
		return nil, err
	}

	out := make([]TaskTypeOption, 0, len(rows))
	indexByKey := make(map[string]int, len(rows))
	for _, row := range rows {
		key := strings.TrimSpace(row.Key)
		if key == "" {
			continue
		}
		value := strings.TrimSpace(row.Value)
		if value == "" {
			value = key
		}
		if idx, ok := indexByKey[key]; ok {
			out[idx].Value = value
			continue
		}
		indexByKey[key] = len(out)
		out = append(out, TaskTypeOption{Key: key, Value: value})
	}
	return out, nil
}
