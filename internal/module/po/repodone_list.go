// =============================================================================
// 文件: internal/module/po/repodone_list.go
// 模块: PO 工作台
// 类型: repo
// 职责: 已办动作列表的查询条件拼装与列表项装配。
// 依赖: internal/module/po/repodone.go
// =============================================================================

package po

import (
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
)

// buildDoneListQuery 在主查询上追加已办列表的全部过滤条件。
func buildDoneListQuery(db *gorm.DB, req RepoFindDoneActionsReq) *gorm.DB {
	scopeSQL, scopeArgs := buildFormalDoneScopeSQL()
	q := db.Table("zt_action AS a").
		Where("a.actor = ?", req.Account).
		Where(scopeSQL, scopeArgs...)
	if req.Result != "" && req.Result != "all" {
		resSQL, resArgs := buildDoneResultFilterSQL(req.Result)
		q = q.Where(resSQL, resArgs...)
	}
	if objectScopeSQL, objectScopeArgs := buildDoneObjectScopeSQL(req.Tab, req.ObjectType); objectScopeSQL != "" {
		q = q.Where(objectScopeSQL, objectScopeArgs...)
	}
	if req.Action != "" && req.Action != "all" {
		sql, args := buildActionFilterSQL(req.Action)
		q = q.Where(sql, args...)
	}
	if req.Keyword != "" {
		kw := "%" + req.Keyword + "%"
		q = q.Where(buildDoneKeywordFilterSQL(), kw, kw, kw, kw, kw, kw)
	}
	return applyDoneTimeRange(q, req, time.Now())
}

// buildDoneListItems 把 DB 行与补数据上下文装配成已办列表项。
func buildDoneListItems(
	rows []doneActionDBRow,
	hists map[int64][2]string,
	objCtxs map[string]doneObjectContext,
	actor string,
) []DoneAction {
	items := make([]DoneAction, 0, len(rows))
	for _, row := range rows {
		meta := formalDoneActions[row.ObjectType+":"+row.Action]
		actionLabel := doneHistoryActionLabel(row.ObjectType, row.Action)
		chg := hists[row.ID]
		ctx := objCtxs[fmt.Sprintf("%s:%d", row.ObjectType, row.ObjectID)]
		title := ctx.Title
		if title == "" {
			title = strings.TrimSpace(doneObjectTypeLabel(row.ObjectType) + " " + doneObjectCode(row.ObjectType, row.ObjectID))
		}
		url := objectViewURLWithProject(row.ObjectType, uint(row.ObjectID), uint(ctx.ProjectID))
		resultCode, resultText := resolveDoneActionResult(row.Action, row.ObjectType, row.Extra, meta.Result)
		items = append(items, DoneAction{
			ID:              row.ID,
			SourceActionId:  row.ID,
			SourceSystem:    "zentao",
			Actor:           actor,
			ActorName:       actor,
			Action:          actionLabel,
			ActionKey:       row.Action,
			ActionName:      actionLabel,
			IsCoreAction:    true,
			ObjectType:      row.ObjectType,
			ObjectTypeLabel: doneObjectTypeLabel(row.ObjectType),
			ObjectID:        row.ObjectID,
			ObjectCode:      doneObjectCode(row.ObjectType, row.ObjectID),
			ObjectName:      title,
			ObjectTitle:     title,
			Date:            row.Date.Format("2006-01-02 15:04:05"),
			HandledAt:       row.Date.Format(time.RFC3339),
			Result:          resultCode,
			ResultCode:      resultCode,
			ResultText:      resultText,
			BeforeStatus:    chg[0],
			AfterStatus:     chg[1],
			// 当前状态只来自对象本身；已办动作结果不能冒充对象状态。
			CurrentStatus: ctx.Status,
			ProjectName:   ctx.ProjectName,
			ExecutionName: ctx.ExecutionName,
			ProductName:   ctx.ProductName,
			PoolName:      ctx.PoolName,
			NextOwnerName: ctx.CurrentOwner,
			CanOpenObject: true,
			URL:           url,
		})
	}
	return items
}
