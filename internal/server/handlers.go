package server

import (
	loginmodule "workbench/internal/module/login"
	"workbench/internal/module/user"
)

// Handlers 聚合各业务模块 Handler。
type Handlers struct {
	Auth *loginmodule.Handler
	User *user.Handler
}
