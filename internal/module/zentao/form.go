// =============================================================================
// 文件: internal/module/zentao/form.go
// 模块: 禅道集成
// 类型: action
// 职责: 定义禅道接口请求/响应结构体。
// 依赖: 无
// =============================================================================
package zentao

// ProductsReq 获取禅道产品列表请求。
type ProductsReq struct{}

// ProductsResp 获取禅道产品列表响应。
type ProductsResp struct {
	Raw []byte
}

type tokenRequest struct {
	Account  string `json:"account"`
	Password string `json:"password"`
}

type tokenResponse struct {
	Token string `json:"token"`
}

// UserItem 禅道用户简要信息。
type UserItem struct {
	Account  string `json:"account"`
	Realname string `json:"realname"`
}

// UsersResp 获取禅道用户列表响应。
type UsersResp struct {
	Items []UserItem `json:"items"`
}
