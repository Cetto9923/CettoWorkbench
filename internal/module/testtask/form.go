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
