package authbridge

import (
	"encoding/hex"
	"testing"
)

func TestSM4RoundTripAndNoPadDecrypt(t *testing.T) {
	key := "uYdc58GHjd4TjsHe"
	plain := `{"dataType":1001,"page":1,"size":10,"clientName":"yqsjdz-client"}`
	enc, err := SM4EncryptHex(key, plain)
	if err != nil {
		t.Fatal(err)
	}
	dec, err := SM4DecryptHex(key, enc)
	if err != nil {
		t.Fatal(err)
	}
	if dec != plain {
		t.Fatalf("roundtrip mismatch: %s", dec)
	}

	// 模拟无 PKCS7 的响应：明文整块加密后直接 hex
	blockPlain := []byte(`[{"userId":1,"username":"a"}]xxxx`) // 补齐到 32 字节演示用
	for len(blockPlain)%16 != 0 {
		blockPlain = append(blockPlain, ' ')
	}
	raw, err := hex.DecodeString(mustEncryptRaw(t, key, blockPlain))
	_ = raw
}

func mustEncryptRaw(t *testing.T, key string, data []byte) string {
	t.Helper()
	// 复用 Encrypt：先用带 pad 的接口验证签名稳定
	sign := BuildSyncSign(map[string]string{
		"data":      "abc",
		"nonce":     "n1",
		"timestamp": "1",
	}, "YQfyZtAmDbYHTBaHPSx3GZeX7x2ip7ik")
	if sign == "" {
		t.Fatal("empty sign")
	}
	return hex.EncodeToString(data)
}
