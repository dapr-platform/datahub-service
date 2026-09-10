/*
 * @module service/authbridge/sync_client
 * @description 萌海用户主数据拉取：GET /sso/dp/syncUser（SM4 + 签名）
 */
package authbridge

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	SyncDataTypeFull        = 1001 // 全量
	SyncDataTypeIncremental = 1002 // 增量
)

// RemoteUser 统一认证用户
type RemoteUser struct {
	UserID   json.Number `json:"userId"`
	OrgID    string      `json:"orgId"`
	IsDel    int         `json:"isDel"` // 1 正常，其它视为删除/停用
	Username string      `json:"username"`
	TrueName string      `json:"trueName"`
	Phone    string      `json:"phone"`
	Email    string      `json:"email"`
}

// UserSyncClient 用户同步 HTTP 客户端
type UserSyncClient struct {
	cfg        Config
	httpClient *http.Client
}

// NewUserSyncClient 创建客户端
func NewUserSyncClient(cfg Config) *UserSyncClient {
	return &UserSyncClient{
		cfg: cfg,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

type syncInnerReq struct {
	DataType   int    `json:"dataType"`
	Page       int    `json:"page"`
	Size       int    `json:"size"`
	ClientName string `json:"clientName"`
	UpdateTime string `json:"updateTime,omitempty"`
}

type syncOuterResp struct {
	Code int             `json:"code"`
	Msg  string          `json:"msg"`
	Data json.RawMessage `json:"data"`
}

// FetchPage 拉取一页用户
func (c *UserSyncClient) FetchPage(ctx context.Context, dataType, page, size int, updateTime string) ([]RemoteUser, error) {
	if c.cfg.UserSyncSM4Key == "" || c.cfg.UserSyncSignKey == "" {
		return nil, fmt.Errorf("USER_SYNC_SM4_KEY / USER_SYNC_SIGN_KEY 未配置")
	}
	inner := syncInnerReq{
		DataType:   dataType,
		Page:       page,
		Size:       size,
		ClientName: c.cfg.UserSyncClient,
		UpdateTime: updateTime,
	}
	innerBytes, err := json.Marshal(inner)
	if err != nil {
		return nil, err
	}
	cipherHex, err := SM4EncryptHex(c.cfg.UserSyncSM4Key, string(innerBytes))
	if err != nil {
		return nil, err
	}
	params := map[string]string{
		"data":      cipherHex,
		"timestamp": strconv.FormatInt(time.Now().UnixMilli(), 10),
		"nonce":     strings.ReplaceAll(uuid.NewString(), "-", ""),
	}
	params["sign"] = BuildSyncSign(params, c.cfg.UserSyncSignKey)

	q := url.Values{}
	for k, v := range params {
		q.Set(k, v)
	}
	reqURL := c.cfg.SSOServerURL + "/sso/dp/syncUser?" + q.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求 syncUser 失败: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var outer syncOuterResp
	if err := json.Unmarshal(body, &outer); err != nil {
		return nil, fmt.Errorf("解析 syncUser 响应失败: %w, body=%s", err, truncate(string(body), 300))
	}
	if outer.Code != 200 {
		return nil, fmt.Errorf("syncUser 失败: code=%d msg=%s", outer.Code, outer.Msg)
	}
	cipher := ""
	if len(outer.Data) > 0 && string(outer.Data) != "null" {
		if err := json.Unmarshal(outer.Data, &cipher); err != nil {
			// 兼容非字符串 data
			cipher = strings.Trim(string(outer.Data), `"`)
		}
	}
	if cipher == "" {
		return []RemoteUser{}, nil
	}
	plain, err := SM4DecryptHex(c.cfg.UserSyncSM4Key, cipher)
	if err != nil {
		return nil, fmt.Errorf("解密 syncUser data 失败: %w", err)
	}
	plain = strings.TrimSpace(plain)
	if plain == "" || plain == "null" || plain == "[]" {
		return []RemoteUser{}, nil
	}
	var users []RemoteUser
	if err := json.Unmarshal([]byte(plain), &users); err != nil {
		return nil, fmt.Errorf("解析用户列表失败: %w, plain=%s", err, truncate(plain, 300))
	}
	return users, nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
