package testtask

import (
	"strconv"
	"strings"
)

// CreateTesttaskItem 单个非联调测试单提交项。
type CreateTesttaskItem struct {
	ProductID uint   `json:"productId"`
	BuildID   uint   `json:"buildId"`
	Name      string `json:"name"`
	Begin     string `json:"begin"`
	End       string `json:"end"`
	Owner     string `json:"owner"`
	Type      string `json:"type"`
	Pri       int    `json:"pri"`
	Desc      string `json:"desc"`
}

// CreateTesttasksReq 第 4 步保存测试单（独立或联调）。
type CreateTesttasksReq struct {
	Joint    int                  `json:"joint"`
	Name     string               `json:"name"`
	Begin    string               `json:"begin"`
	End      string               `json:"end"`
	Owner    string               `json:"owner"`
	Members  []string             `json:"members"`
	Products []uint               `json:"products"`
	Builds   [][]uint             `json:"builds"`
	Type     string               `json:"type"`
	Pri      int                  `json:"pri"`
	Desc     string               `json:"desc"`
	Tasks    []CreateTesttaskItem `json:"tasks"`
}

// CreateTesttaskResult 单个测试单创建结果。
type CreateTesttaskResult struct {
	ProductID  uint   `json:"productId"`
	BuildID    uint   `json:"buildId"`
	TesttaskID uint   `json:"testtaskId"`
	Name       string `json:"name"`
}

// CreateTesttasksResp 批量创建测试单响应。
type CreateTesttasksResp struct {
	Tasks []CreateTesttaskResult `json:"tasks"`
}

// IdempotencyKey 在同一请求内去重 (buildID, name) 用；与远程幂等无关，仅作内层去重与日志锚点。
func (it CreateTesttaskItem) IdempotencyKey() string {
	return strconv.FormatUint(uint64(it.BuildID), 10) + ":" + strings.TrimSpace(it.Name)
}

// Validate 校验创建测试单请求（独立或联调）。
func (r *CreateTesttasksReq) Validate() []FieldError {
	var errs []FieldError
	if r == nil {
		errs = append(errs, FieldError{Field: "_form", Message: "请至少配置一个测试单"})
		return errs
	}
	if r.Joint != 0 && r.Joint != 1 {
		return []FieldError{{Field: "joint", Message: "联调标识无效"}}
	}
	if r.Joint == 1 {
		return r.validateJoint()
	}
	if len(r.Tasks) == 0 {
		errs = append(errs, FieldError{Field: "_form", Message: "请至少配置一个测试单"})
		return errs
	}
	for i, item := range r.Tasks {
		prefix := "tasks." + strconv.Itoa(i)
		if item.ProductID == 0 {
			errs = append(errs, FieldError{Field: prefix + ".productId", Message: "产品无效"})
		}
		if item.BuildID == 0 {
			errs = append(errs, FieldError{Field: prefix + ".buildId", Message: "所属版本不能为空"})
		}
		if strings.TrimSpace(item.Name) == "" {
			errs = append(errs, FieldError{Field: prefix + ".name", Message: "测试单名称不能为空"})
		}
		begin := strings.TrimSpace(item.Begin)
		if begin == "" {
			errs = append(errs, FieldError{Field: prefix + ".begin", Message: "开始日期不能为空"})
		} else if !isYMD(begin) {
			errs = append(errs, FieldError{Field: prefix + ".begin", Message: "开始日期格式应为 YYYY-MM-DD"})
		}
		end := strings.TrimSpace(item.End)
		if end == "" {
			errs = append(errs, FieldError{Field: prefix + ".end", Message: "结束日期不能为空"})
		} else if !isYMD(end) {
			errs = append(errs, FieldError{Field: prefix + ".end", Message: "结束日期格式应为 YYYY-MM-DD"})
		}
		if begin != "" && end != "" && isYMD(begin) && isYMD(end) && end < begin {
			errs = append(errs, FieldError{Field: prefix + ".end", Message: "结束日期不能早于开始日期"})
		}
		if strings.TrimSpace(item.Owner) == "" {
			errs = append(errs, FieldError{Field: prefix + ".owner", Message: "测试负责人不能为空"})
		}
		if strings.TrimSpace(item.Type) == "" {
			errs = append(errs, FieldError{Field: prefix + ".type", Message: "测试类型不能为空"})
		}
		if item.Pri < 1 || item.Pri > 4 {
			errs = append(errs, FieldError{Field: prefix + ".pri", Message: "优先级无效"})
		}
	}
	return errs
}

func (r *CreateTesttasksReq) validateJoint() []FieldError {
	if len(r.Products) == 0 || len(r.Products) != len(r.Builds) {
		return []FieldError{{Field: "builds", Message: "各系统及版本必须一一对应"}}
	}
	seen := map[uint]bool{}
	for i, product := range r.Products {
		if product == 0 || seen[product] || len(r.Builds[i]) == 0 {
			return []FieldError{{Field: "products", Message: "系统不能重复或为空，且必须选择版本"}}
		}
		seen[product] = true
		for _, id := range r.Builds[i] {
			if id == 0 {
				return []FieldError{{Field: "builds", Message: "版本无效"}}
			}
		}
	}
	// 复用独立测试单的名称、日期、负责人等字段合同。
	check := CreateTesttasksReq{Tasks: []CreateTesttaskItem{{ProductID: r.Products[0], BuildID: r.Builds[0][0], Name: r.Name, Begin: r.Begin, End: r.End, Owner: r.Owner, Type: r.Type, Pri: r.Pri}}}
	errs := check.Validate()
	for i := range errs {
		errs[i].Field = strings.TrimPrefix(errs[i].Field, "tasks.0.")
	}
	return errs
}
