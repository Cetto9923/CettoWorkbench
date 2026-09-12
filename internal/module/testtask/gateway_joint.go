package testtask

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"workbench/internal/pkg/zentao"
)

type createJointTesttaskReq struct {
	Account   string
	ProjectID uint
	Name      string
	Begin     string
	End       string
	Owner     string
	Members   []string
	Products  []uint
	Builds    [][]uint
	Type      string
	Pri       int
	Status    string
	Desc      string
}

// jointTesttaskPayload 组装禅道联调测试单 POST body。
func jointTesttaskPayload(req createJointTesttaskReq) map[string]any {
	status := strings.TrimSpace(req.Status)
	if status == "" {
		status = "wait"
	}
	members := normalizeJointMembers(req.Owner, req.Members)
	builds := make(map[string][]uint, len(req.Products))
	for i := range req.Products {
		key := fmt.Sprintf("%d", i)
		if i < len(req.Builds) {
			builds[key] = req.Builds[i]
		} else {
			builds[key] = []uint{}
		}
	}
	return map[string]any{
		"joint":    "1",
		"name":     strings.TrimSpace(req.Name),
		"begin":    strings.TrimSpace(req.Begin),
		"end":      strings.TrimSpace(req.End),
		"owner":    strings.TrimSpace(req.Owner),
		"members":  members,
		"products": req.Products,
		"builds":   builds,
		"type":     strings.TrimSpace(req.Type),
		"pri":      req.Pri,
		"status":   status,
		"desc":     req.Desc,
	}
}

func normalizeJointMembers(owner string, members []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(members)+1)
	add := func(raw string) {
		s := strings.TrimSpace(raw)
		if s == "" {
			return
		}
		if _, ok := seen[s]; ok {
			return
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	add(owner)
	for _, m := range members {
		add(m)
	}
	return out
}

// createJointTesttask 调用禅道创建联调测试单 POST /projects/:id/testtasks。
func createJointTesttask(ctx context.Context, client *zentao.Client, req createJointTesttaskReq) (*createdTesttask, error) {
	if client == nil {
		return nil, fmt.Errorf("禅道 API 未配置")
	}
	if req.ProjectID == 0 {
		return nil, fmt.Errorf("projectId 无效")
	}
	if len(req.Products) == 0 {
		return nil, fmt.Errorf("products 不能为空")
	}

	payload := jointTesttaskPayload(req)
	var out createdTesttask
	path := fmt.Sprintf("/projects/%d/testtasks", req.ProjectID)
	if err := client.DoAs(ctx, req.Account, http.MethodPost, path, payload, &out); err != nil {
		return nil, err
	}
	if out.ID == 0 {
		return nil, fmt.Errorf("禅道未返回测试单 ID")
	}
	return &out, nil
}
