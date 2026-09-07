// =============================================================================
// 文件: internal/module/po/repotodoapproval.go
// 模块: PO 工作台
// 类型: action
// 职责: 查询禅道通用审批流中当前轮到本人的真实审批待办。
// 依赖: internal/pkg/zentao
// =============================================================================

package po

import (
	"context"
	"fmt"
	"strings"
	"time"

	"workbench/internal/pkg/zentao"
)

// FindCurrentApprovalTodos 返回 doing/review 当前节点，不把未来 wait 节点或抄送节点算作待办。
func (r *Repo) FindCurrentApprovalTodos(ctx context.Context, account string) ([]TodoItem, error) {
	type row struct {
		NodeID     int64      `gorm:"column:node_id"`
		ApprovalID int64      `gorm:"column:approval_id"`
		ObjectType string     `gorm:"column:object_type"`
		ObjectID   int64      `gorm:"column:object_id"`
		ProjectID  int64      `gorm:"column:project_id"`
		Title      string     `gorm:"column:title"`
		Deadline   *time.Time `gorm:"column:deadline"`
	}
	var rows []row
	err := r.db.WithContext(ctx).Raw(`
		SELECT MIN(n.id) AS node_id,
		       n.approval AS approval_id,
		       ao.objectType AS object_type,
		       ao.objectID AS object_id,
		       CASE
		         WHEN ao.objectType = 'charter' THEN c.project
		         WHEN ao.objectType = 'planchange' THEN pc.project
		         WHEN ao.objectType = 'buildguideline' THEN bg.projectID
		         WHEN ao.objectType = 'review' THEN rv.project
		         ELSE 0
		       END AS project_id,
		       CASE
		         WHEN ao.objectType = 'charter' THEN COALESCE(NULLIF(p1.name, ''), '项目章程')
		         WHEN ao.objectType = 'planchange' THEN COALESCE(NULLIF(pc.title, ''), NULLIF(p2.name, ''), '计划变更')
		         WHEN ao.objectType = 'buildguideline' THEN COALESCE(NULLIF(p3.name, ''), '项目建设指引')
		         WHEN ao.objectType = 'review' THEN COALESCE(NULLIF(rv.title, ''), NULLIF(p4.name, ''), '项目评审')
		         ELSE ''
		       END AS title,
		       CASE WHEN ao.objectType = 'review' THEN rv.deadline ELSE NULL END AS deadline
		FROM zt_approvalnode AS n
		INNER JOIN zt_approvalobject AS ao ON ao.approval = n.approval
		LEFT JOIN zt_charter AS c ON ao.objectType = 'charter' AND c.id = ao.objectID AND c.deleted = '0'
		LEFT JOIN zt_project AS p1 ON p1.id = c.project AND p1.deleted = '0'
		LEFT JOIN zt_planchange AS pc ON ao.objectType = 'planchange' AND pc.id = ao.objectID
		LEFT JOIN zt_project AS p2 ON p2.id = pc.project AND p2.deleted = '0'
		LEFT JOIN zt_projectbuildguide AS bg ON ao.objectType = 'buildguideline' AND bg.id = ao.objectID AND bg.deleted = '0'
		LEFT JOIN zt_project AS p3 ON p3.id = bg.projectID AND p3.deleted = '0'
		LEFT JOIN zt_review AS rv ON ao.objectType = 'review' AND rv.id = ao.objectID AND rv.deleted = '0'
		LEFT JOIN zt_project AS p4 ON p4.id = rv.project AND p4.deleted = '0'
		WHERE n.account = ?
		  AND n.status = 'doing'
		  AND n.type = 'review'
		  AND ao.objectType IN ('charter', 'planchange', 'buildguideline', 'review')
		  AND ao.objectID > 0
		GROUP BY n.approval, ao.objectType, ao.objectID, c.project, pc.project, bg.projectID,
		         rv.project, p1.name, pc.title, p2.name, p3.name, rv.title, p4.name, rv.deadline`, account).
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	displayMap, _ := r.loadAccountDisplayMap(ctx)
	owner := displayMap[account]
	if owner == "" {
		owner = account
	}
	items := make([]TodoItem, 0, len(rows))
	for _, row := range rows {
		if strings.TrimSpace(row.Title) == "" {
			continue
		}
		items = append(items, TodoItem{
			Kind: "approval", ID: row.ApprovalID, DisplayID: fmt.Sprintf("%d", row.ApprovalID),
			Title: row.Title, Type: approvalObjectLabel(row.ObjectType), Stage: "doing",
			Relation: "我负责", Responsibility: "待我处理", Reason: approvalObjectReason(row.ObjectType),
			Deadline: formatTodoDeadline(row.Deadline), Owner: owner,
			URL: approvalObjectURL(row.ObjectType, row.ObjectID, row.ProjectID), Action: "审批",
		})
	}
	return items, nil
}

func approvalObjectLabel(objectType string) string {
	if objectType == "review" {
		return "项目评审"
	}
	return "审批"
}

func approvalObjectReason(objectType string) string {
	switch objectType {
	case "charter":
		return "项目章程审批"
	case "planchange":
		return "计划变更审批"
	case "buildguideline":
		return "项目建设指引审批"
	case "review":
		return "项目评审"
	}
	return "待审批"
}

func approvalObjectURL(objectType string, objectID, projectID int64) string {
	switch objectType {
	case "charter":
		return zentao.URL("charter", "view", fmt.Sprintf("projectID=%d", projectID))
	case "planchange":
		return zentao.URL("planchange", "view", fmt.Sprintf("ID=%d", objectID))
	case "buildguideline":
		return zentao.URL("buildguideline", "view", fmt.Sprintf("projectID=%d", projectID))
	case "review":
		return zentao.URL("review", "view", fmt.Sprintf("reviewID=%d", objectID))
	}
	return ""
}
