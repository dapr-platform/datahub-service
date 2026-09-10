/*
 * @module service/authbridge/sm4util
 * @description 萌海同步接口 SM4(ECB/PKCS7) + 签名工具
 */
package authbridge

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/tjfoc/gmsm/sm4"
)

// SM4EncryptHex SM4-ECB PKCS7，输出小写 hex
func SM4EncryptHex(key, plain string) (string, error) {
	k := []byte(key)
	if len(k) != 16 {
		return "", fmt.Errorf("sm4 key 长度必须为 16，当前 %d", len(k))
	}
	data := pkcs7Pad([]byte(plain), 16)
	out, err := sm4.Sm4Ecb(k, data, true)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(out), nil
}

// SM4DecryptHex 解密 hex 密文。
// 请求侧多用 PKCS7；响应侧常见无填充（明文刚好整块），故 PKCS7 失败时按无填充/去零处理。
func SM4DecryptHex(key, cipherHex string) (string, error) {
	k := []byte(key)
	if len(k) != 16 {
		return "", fmt.Errorf("sm4 key 长度必须为 16，当前 %d", len(k))
	}
	raw, err := hex.DecodeString(strings.TrimSpace(cipherHex))
	if err != nil {
		return "", err
	}
	out, err := sm4.Sm4Ecb(k, raw, false)
	if err != nil {
		return "", err
	}
	if unpadded, uerr := pkcs7Unpad(out, 16); uerr == nil {
		return string(unpadded), nil
	}
	return string(bytes.TrimRight(out, "\x00")), nil
}

// BuildSyncSign 同步接口签名：md5(k=v&...&key=signKey)
func BuildSyncSign(params map[string]string, signKey string) string {
	return BuildSaSign(params, signKey)
}

func pkcs7Pad(data []byte, block int) []byte {
	pad := block - len(data)%block
	out := make([]byte, len(data)+pad)
	copy(out, data)
	for i := len(data); i < len(out); i++ {
		out[i] = byte(pad)
	}
	return out
}

func pkcs7Unpad(data []byte, block int) ([]byte, error) {
	if len(data) == 0 || len(data)%block != 0 {
		return nil, fmt.Errorf("非法 PKCS7 数据长度")
	}
	pad := int(data[len(data)-1])
	if pad < 1 || pad > block {
		return nil, fmt.Errorf("非法 PKCS7 padding")
	}
	for i := 0; i < pad; i++ {
		if data[len(data)-1-i] != byte(pad) {
			return nil, fmt.Errorf("PKCS7 padding 校验失败")
		}
	}
	return data[:len(data)-pad], nil
}
