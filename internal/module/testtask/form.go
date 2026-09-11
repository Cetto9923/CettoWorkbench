// =============================================================================
// 文件: internal/module/testtask/form.go
// 模块: 提测办理
// 类型: action
// 职责: 提测上下文与创建版本请求/响应结构。
// 依赖: 无
// =============================================================================

package testtask

import (
	"strconv"
	"strings"
	"time"
)

// DemandContextRow 业需上下文查询行（zt_demand + 主系统名；人员为账号，展示名由 user map 解析）。
type DemandContextRow struct {
	ID             uint   `gorm:"column:id"`
	Name           string `gorm:"column:name"`
	Status         string `gorm:"column:status"`
	Stage          string `gorm:"column:stage"`
	BRA            string `gorm:"column:BRA"`
	RD             string `gorm:"column:RD"`
	QD             string `gorm:"column:QD"`
	MainSystemID   uint   `gorm:"column:main_system_id"`
	MainSystemName string `gorm:"column:product_name"`
	EstimateLaunch string `gorm:"column:estimate_launch"`
}

// StoryItem 业务需求实际转出的研发需求简要信息。
type StoryItem struct {
	ID        uint   `json:"id"`
	Title     string `json:"title"`
	Pri       int    `json:"pri"`
	Status    string `json:"status"`
	Stage     string `json:"stage"`
	ProductID uint   `json:"productId"`
}

// SystemItem 需求涉及产品/系统（选择系统列表项）。
type SystemItem struct {
	ID      uint        `json:"id"`
	Name    string      `json:"name"`
	IsMain  bool        `json:"isMain"`
	Stories []StoryItem `json:"stories"`
}

// ExecutionOption 所属执行下拉项（创建新版本）。
// Value 为「项目id-执行id」，Label 为「项目名称/执行名称」。
type ExecutionOption struct {
	Value       string `json:"value"`
	Label       string `json:"label"`
	ProjectID   uint   `json:"projectId"`
	ExecutionID uint   `json:"executionId"`
}

// BuildOption 是已有版本下拉项。
type BuildOption struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

// UserOption 测试负责人检索下拉项（内部用户）。
type UserOption struct {
	Account  string `json:"account"`
	Realname string `json:"realname"`
}

// ContextResp 提测弹窗「当前需求上下文」JSON。
type ContextResp struct {
	DemandID       uint         `json:"demandId"`
	Title          string       `json:"title"`
	Stage          string       `json:"stage"`
	RawStatus      string       `json:"rawStatus"`
	MainSystemName string       `json:"mainSystemName"`
	EstimateLaunch string       `json:"estimateLaunch"`
	BRAName        string       `json:"braName"`
	RDName         string       `json:"rdName"`
	QD             string       `json:"qd"`
	QDName         string       `json:"qdName"`
	HandlerName    string       `json:"handlerName"`
	Systems        []SystemItem `json:"systems"`
	Users          []UserOption `json:"users"`
}

// FieldError 表单字段级错误。
type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// CreateBuildItem 单个「创建新版本」提交项。
type CreateBuildItem struct {
	ProductID   uint   `json:"productId"`
	ProjectID   uint   `json:"projectId"`
	ExecutionID uint   `json:"executionId"`
	Name        string `json:"name"`
	Date        string `json:"date"`
	Desc        string `json:"desc"`
}

// CreateBuildsReq 第 2 步保存新版本（仅创建新版本项）。
type CreateBuildsReq struct {
	Builds []CreateBuildItem `json:"builds"`
}

// CreateBuildResult 单个版本创建结果。
type CreateBuildResult struct {
	ProductID uint   `json:"productId"`
	BuildID   uint   `json:"buildId"`
	Name      string `json:"name"`
}

// CreateBuildsResp 批量创建版本响应。
type CreateBuildsResp struct {
	Builds []CreateBuildResult `json:"builds"`
}

// Validate 校验创建版本请求。
func (r *CreateBuildsReq) Validate() []FieldError {
	var errs []FieldError
	if r == nil || len(r.Builds) == 0 {
		errs = append(errs, FieldError{Field: "_form", Message: "请至少创建一个系统版本"})
		return errs
	}
	for i, item := range r.Builds {
		prefix := "builds." + strconv.Itoa(i)
		if item.ProductID == 0 {
			errs = append(errs, FieldError{Field: prefix + ".productId", Message: "产品无效"})
		}
		if item.ProjectID == 0 {
			errs = append(errs, FieldError{Field: prefix + ".projectId", Message: "所属执行的项目无效"})
		}
		if item.ExecutionID == 0 {
			errs = append(errs, FieldError{Field: prefix + ".executionId", Message: "所属执行不能为空"})
		}
		if strings.TrimSpace(item.Name) == "" {
			errs = append(errs, FieldError{Field: prefix + ".name", Message: "版本名称不能为空"})
		}
		date := strings.TrimSpace(item.Date)
		if date == "" {
			errs = append(errs, FieldError{Field: prefix + ".date", Message: "计划上线日期不能为空"})
		} else if !isYMD(date) {
			errs = append(errs, FieldError{Field: prefix + ".date", Message: "计划上线日期格式应为 YYYY-MM-DD"})
		}
	}
	return errs
}

func isYMD(s string) bool {
	_, err := time.ParseInLocation("2006-01-02", s, time.Local)
	return err == nil
}

// =============================================================================
// 非联调测试单提交与响应结构
// =============================================================================

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

// CreateTesttasksReq 第 4 步保存测试单（当前仅支持非联调）。
type CreateTesttasksReq struct {
	Joint int                  `json:"joint"`
	Tasks []CreateTesttaskItem `json:"tasks"`
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

// Validate 校验创建测试单请求（联调暂不支持）。
func (r *CreateTesttasksReq) Validate() []FieldError {
	var errs []FieldError
	if r == nil {
		errs = append(errs, FieldError{Field: "_form", Message: "请至少配置一个测试单"})
		return errs
	}
	if r.Joint == 1 {
		errs = append(errs, FieldError{Field: "joint", Message: "联调测试单暂不支持，请选择否"})
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
