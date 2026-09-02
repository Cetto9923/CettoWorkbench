// =============================================================================
// 文件: internal/pkg/password/md5.go
// 模块: 基础设施
// 类型: infra
// 职责: 提供与禅道 zt_user 兼容的 MD5 密码摘要。
// 依赖: 无
// =============================================================================

package encode

import (
	"crypto/md5"
	"encoding/hex"
)

// MD5 计算明文密码的 MD5 十六进制摘要（与禅道 zt_user.password 存储规则一致）。
func MD5(plain string) string {
	sum := md5.Sum([]byte(plain))
	return hex.EncodeToString(sum[:])
}
