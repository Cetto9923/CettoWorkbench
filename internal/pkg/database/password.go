// =============================================================================
// 文件: internal/pkg/database/password.go
// 模块: 基础设施
// 类型: infra
// 职责: 解密 OceanBase 连接密码，对齐禅道 helper::decryptPassword 的 crcb 分支。
// 依赖: 无
// =============================================================================

package database

import (
	"encoding/base64"
	"math/big"
	"strings"
)

const crcbPasswordPrefix = "crcb:"

// decryptPassword 解密数据库密码。
// 禅道连 OceanBase 时配置为 "crcb:<密文> <私钥>" 或 "crcb:<密文>|<私钥>"；
// 无法解密时退回原文，避免明文密码被改写。
func decryptPassword(password string) string {
	if password == "" {
		return ""
	}
	if strings.HasPrefix(password, crcbPasswordPrefix) {
		value := strings.TrimSpace(password[len(crcbPasswordPrefix):])
		parts := splitCrcbParts(value)
		if len(parts) == 2 {
			decrypted := strings.TrimSpace(decryptCrcbPassword(strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])))
			if decrypted != "" {
				return decrypted
			}
		}
	}
	return strings.TrimSpace(password)
}

// splitCrcbParts 按空白或 | 切成至多两段，对齐 preg_split('/[\s|]+/', $value, 2)。
func splitCrcbParts(value string) []string {
	i := indexCrcbSep(value)
	if i < 0 {
		return []string{value}
	}
	j := i
	for j < len(value) && isCrcbSep(value[j]) {
		j++
	}
	if j >= len(value) {
		return []string{value[:i]}
	}
	return []string{value[:i], value[j:]}
}

func indexCrcbSep(value string) int {
	for i := 0; i < len(value); i++ {
		if isCrcbSep(value[i]) {
			return i
		}
	}
	return -1
}

func isCrcbSep(b byte) bool {
	switch b {
	case '|', ' ', '\t', '\n', '\v', '\f', '\r':
		return true
	default:
		return false
	}
}

// decryptCrcbPassword 对齐禅道 helper::decryptCrcbPassword。
// 密文是按 6 bit 打包的自定义 Base64；私钥是 Base64("模数,指数")，逐字节做模幂。
func decryptCrcbPassword(cipher, privateKey string) string {
	if cipher == "" || privateKey == "" {
		return ""
	}
	encoded, ok := unpackCrcbCipher(cipher)
	if !ok {
		return ""
	}
	keyRaw, err := decodeCrcbKey(privateKey)
	if err != nil {
		return ""
	}
	keyParts := strings.Split(string(keyRaw), ",")
	if len(keyParts) != 2 {
		return ""
	}
	divisor := new(big.Int)
	pow := new(big.Int)
	if _, ok := divisor.SetString(strings.TrimSpace(keyParts[0]), 10); !ok || divisor.Sign() <= 0 {
		return ""
	}
	if _, ok := pow.SetString(strings.TrimSpace(keyParts[1]), 10); !ok || pow.Sign() < 0 {
		return ""
	}

	base := new(big.Int)
	modResult := new(big.Int)
	mask := big.NewInt(255)
	low := new(big.Int)
	var clear strings.Builder
	clear.Grow(len(encoded))
	for _, value := range encoded {
		base.SetInt64(int64(value))
		modResult.Exp(base, pow, divisor)
		low.And(modResult, mask)
		clear.WriteByte(byte(low.Uint64()))
	}
	return clear.String()
}

const crcbAlphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"

func unpackCrcbCipher(cipher string) ([]byte, bool) {
	index := make(map[byte]int, len(crcbAlphabet))
	for i := 0; i < len(crcbAlphabet); i++ {
		index[crcbAlphabet[i]] = i
	}
	var bits strings.Builder
	bits.Grow(len(cipher) * 6)
	for i := 0; i < len(cipher); i++ {
		v, ok := index[cipher[i]]
		if !ok {
			return nil, false
		}
		bits.WriteString(sixBits(v))
	}
	bin := bits.String()
	count := len(bin) / 8
	out := make([]byte, count)
	for i := 0; i < count; i++ {
		chunk := bin[i*8 : (i+1)*8]
		var b byte
		for _, c := range chunk {
			b <<= 1
			if c == '1' {
				b |= 1
			}
		}
		out[i] = b
	}
	return out, true
}

func sixBits(v int) string {
	buf := []byte{'0', '0', '0', '0', '0', '0'}
	for i := 5; i >= 0; i-- {
		if v&1 == 1 {
			buf[i] = '1'
		}
		v >>= 1
	}
	return string(buf)
}

func decodeCrcbKey(privateKey string) ([]byte, error) {
	if raw, err := base64.StdEncoding.DecodeString(privateKey); err == nil {
		return raw, nil
	}
	return base64.RawStdEncoding.DecodeString(privateKey)
}
