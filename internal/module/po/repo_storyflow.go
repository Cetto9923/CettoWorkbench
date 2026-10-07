// =============================================================================
// 文件: internal/module/po/repo_storyflow.go
// 模块: PO 工作台
// 类型: repo
// 职责: 独立研需的交付与窗口分格条件。
// =============================================================================
package po

import "strconv"

const storyDeliveryStartedSQL = `(TRIM(IFNULL(veriFier, '')) <> '' OR (TRIM(IFNULL(verifyDate, '')) REGEXP '^[1-9][0-9]{3}-[0-9]{2}-[0-9]{2}' AND STR_TO_DATE(LEFT(verifyDate, 10), '%Y-%m-%d') IS NOT NULL))`
const storyDeliveredSQL = `(status = 'closed' OR stage IN ('released', 'closed', 'delivering', 'delivered') OR ` + storyDeliveryStartedSQL + `)`
const storyWindowBoundSQL = `EXISTS (SELECT 1 FROM zt_planstory ps JOIN zt_versionwindowproduct vwp ON vwp.plan = ps.plan AND vwp.deletedAt IS NULL WHERE ps.story = zt_story.id)`

func homeFocusStoryStageSQL() string {
	return "CASE WHEN " + storyDeliveredSQL + " THEN 99 WHEN " + storyWindowBoundSQL + " THEN " + strconv.Itoa(homeFocusStageIndex("acceptanced")) + " ELSE " + strconv.Itoa(homeFocusStageIndex("schedule")) + " END"
}
