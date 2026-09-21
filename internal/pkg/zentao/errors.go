package zentao

import "errors"

var (
	ErrZentaoUnreachable = errors.New("无法连接禅道服务器")
	ErrZentaoAuthFailed  = errors.New("禅道鉴权失败")
	ErrZentaoAPIError    = errors.New("禅道接口执行异常")
)
