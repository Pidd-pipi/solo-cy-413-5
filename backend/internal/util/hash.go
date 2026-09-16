package util

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

// HashStable 以稳定顺序拼接并计算 sha256，用作建议引擎的输入指纹。
// 相同的近 14 天数据必然得到相同哈希，用于判断是否需要真正重算。
func HashStable(parts ...string) string {
	sum := sha256.Sum256([]byte(strings.Join(parts, "|")))
	return hex.EncodeToString(sum[:])
}
