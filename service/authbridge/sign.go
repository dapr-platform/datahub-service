/*
 * @module service/authbridge/sign
 * @description Sa-Token SSO 消息签名（md5 字典序 + key=secret）
 */
package authbridge

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
)

// BuildSaSign 按 Sa-Token 规则生成签名
// 算法: md5( k1=v1&k2=v2&...&key={secret} )，参数按 key 字典序，排除 sign，空值不参与
func BuildSaSign(params map[string]string, secret string) string {
	keys := make([]string, 0, len(params))
	for k, v := range params {
		if k == "sign" || v == "" {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys)+1)
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s=%s", k, params[k]))
	}
	parts = append(parts, "key="+secret)
	raw := strings.Join(parts, "&")
	sum := md5.Sum([]byte(raw))
	return hex.EncodeToString(sum[:])
}
