// =============================================================================
// 文件: internal/module/po/todo_query_extra.go
// 模块: PO 工作台
// 类型: repo
// 职责: 我的待办附加对象（审批汇总、研发需求、风险、问题、个人待办、测试单）SQL 构建。
// =============================================================================

package po

func buildTodoApprovalSQL(account string) (string, []interface{}) {
	sql := `SELECT 'approval' AS kind, ao.approval AS id, CAST(ao.approval AS CHAR) AS display_id,
		CASE
			WHEN ao.objectType = 'charter' THEN COALESCE(NULLIF(p1.name, ''), '项目章程')
			WHEN ao.objectType = 'planchange' THEN COALESCE(NULLIF(pc.title, ''), NULLIF(p2.name, ''), '计划变更')
			WHEN ao.objectType = 'buildguideline' THEN COALESCE(NULLIF(p3.name, ''), '项目建设指引')
			WHEN ao.objectType = 'review' THEN COALESCE(NULLIF(rv.title, ''), NULLIF(p4.name, ''), '项目评审')
			ELSE '审批'
		END AS title,
		'doing' AS status,
		'2' AS pri_str, 2 AS priority_rank,
		CASE WHEN ao.objectType = 'review' AND rv.deadline IS NOT NULL AND rv.deadline <> '0000-00-00' THEN DATE_FORMAT(rv.deadline, '%Y-%m-%d') ELSE '9999-12-31' END AS deadline_str,
		n.account AS owner_account, '我负责' AS relation, '待我处理' AS responsibility, 0 AS blocked, 4 AS type_order
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
	GROUP BY ao.approval, ao.objectType, ao.objectID, c.project, pc.project, bg.projectID,
	         rv.project, p1.name, pc.title, p2.name, p3.name, rv.title, p4.name, rv.deadline, n.account`
	return sql, []interface{}{account}
}

func buildTodoStorySQL(account string) (string, []interface{}) {
	sql := `SELECT 'story' AS kind, s.id, CAST(s.id AS CHAR) AS display_id, s.title AS title, s.status,
		CASE WHEN s.pri = 1 THEN '1' WHEN s.pri = 2 THEN '2' WHEN s.pri = 3 THEN '3' WHEN s.pri = 4 THEN '4' ELSE '0' END AS pri_str,
		CASE WHEN s.pri = 1 THEN 1 WHEN s.pri = 2 THEN 2 WHEN s.pri = 3 THEN 3 ELSE 4 END AS priority_rank,
		CASE WHEN s.deliverDate IS NULL OR s.deliverDate = '0000-00-00' THEN '9999-12-31' ELSE DATE_FORMAT(s.deliverDate, '%Y-%m-%d') END AS deadline_str,
		s.assignedTo AS owner_account, '我负责' AS relation, '待我处理' AS responsibility, 0 AS blocked, 5 AS type_order
	FROM zt_story AS s
	WHERE s.deleted = '0' AND s.assignedTo = ? AND s.status NOT IN ('closed', 'released')
	  AND IFNULL(s.sourceType, '') <> 'demandpool'`
	return sql, []interface{}{account}
}

func buildTodoRiskSQL(account string) (string, []interface{}) {
	sql := `SELECT 'risk' AS kind, rk.id, CAST(rk.id AS CHAR) AS display_id, rk.name AS title, rk.status,
		CASE WHEN rk.pri = '1' THEN '1' WHEN rk.pri = '2' THEN '2' WHEN rk.pri = '3' THEN '3' WHEN rk.pri = '4' THEN '4' ELSE '0' END AS pri_str,
		CASE WHEN rk.pri = '1' THEN 1 WHEN rk.pri = '2' THEN 2 WHEN rk.pri = '3' THEN 3 ELSE 4 END AS priority_rank,
		'9999-12-31' AS deadline_str,
		rk.assignedTo AS owner_account, '我负责' AS relation, '待我处理' AS responsibility, 0 AS blocked, 6 AS type_order
	FROM zt_risk AS rk
	WHERE rk.deleted = '0' AND (rk.assignedTo = ? OR rk.createdBy = ?) AND rk.status NOT IN ('closed', 'cancel')`
	return sql, []interface{}{account, account}
}

func buildTodoIssueSQL(account string) (string, []interface{}) {
	sql := `SELECT 'issue' AS kind, iss.id, CAST(iss.id AS CHAR) AS display_id, iss.title AS title, iss.status,
		CASE WHEN iss.pri = '1' THEN '1' WHEN iss.pri = '2' THEN '2' WHEN iss.pri = '3' THEN '3' WHEN iss.pri = '4' THEN '4' ELSE '0' END AS pri_str,
		CASE WHEN iss.pri = '1' THEN 1 WHEN iss.pri = '2' THEN 2 WHEN iss.pri = '3' THEN 3 ELSE 4 END AS priority_rank,
		CASE WHEN iss.deadline IS NULL OR iss.deadline = '0000-00-00' THEN '9999-12-31' ELSE DATE_FORMAT(iss.deadline, '%Y-%m-%d') END AS deadline_str,
		iss.assignedTo AS owner_account, '我负责' AS relation, '待我处理' AS responsibility, 0 AS blocked, 7 AS type_order
	FROM zt_issue AS iss
	WHERE iss.deleted = '0' AND (iss.assignedTo = ? OR iss.createdBy = ?) AND iss.status NOT IN ('closed', 'cancel')`
	return sql, []interface{}{account, account}
}

func buildTodoPersonalSQL(account string) (string, []interface{}) {
	sql := `SELECT 'todo' AS kind, td.id, CAST(td.id AS CHAR) AS display_id, td.name AS title, td.status,
		CASE WHEN td.pri = 1 THEN '1' WHEN td.pri = 2 THEN '2' WHEN td.pri = 3 THEN '3' WHEN td.pri = 4 THEN '4' ELSE '0' END AS pri_str,
		CASE WHEN td.pri = 1 THEN 1 WHEN td.pri = 2 THEN 2 WHEN td.pri = 3 THEN 3 ELSE 4 END AS priority_rank,
		CASE WHEN td.date IS NULL OR td.date = '0000-00-00' THEN '9999-12-31' ELSE DATE_FORMAT(td.date, '%Y-%m-%d') END AS deadline_str,
		CASE WHEN TRIM(td.assignedTo) <> '' THEN td.assignedTo ELSE td.account END AS owner_account,
		'我负责' AS relation, '待我处理' AS responsibility, 0 AS blocked, 8 AS type_order
	FROM zt_todo AS td
	WHERE td.deleted = '0' AND (td.account = ? OR td.assignedTo = ?) AND td.status NOT IN ('done', 'closed')`
	return sql, []interface{}{account, account}
}

func buildTodoTesttaskSQL(account string) (string, []interface{}) {
	sql := `SELECT 'testtask' AS kind, tt.id, CAST(tt.id AS CHAR) AS display_id, tt.name AS title, tt.status,
		CASE WHEN tt.pri = 1 THEN '1' WHEN tt.pri = 2 THEN '2' WHEN tt.pri = 3 THEN '3' WHEN tt.pri = 4 THEN '4' ELSE '0' END AS pri_str,
		CASE WHEN tt.pri = 1 THEN 1 WHEN tt.pri = 2 THEN 2 WHEN tt.pri = 3 THEN 3 ELSE 4 END AS priority_rank,
		CASE WHEN tt.end IS NULL OR tt.end = '0000-00-00' THEN '9999-12-31' ELSE DATE_FORMAT(tt.end, '%Y-%m-%d') END AS deadline_str,
		tt.owner AS owner_account, '我负责' AS relation, '待我处理' AS responsibility,
		CASE WHEN tt.status = 'blocked' THEN 1 ELSE 0 END AS blocked, 9 AS type_order
	FROM zt_testtask AS tt
	WHERE tt.deleted = '0' AND tt.owner = ? AND tt.status <> 'done'`
	return sql, []interface{}{account}
}
