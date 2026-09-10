/*
 * @module service/authbridge/local_user
 * @description 本地用户桥接：外部身份 → ensure_external_user → issue_token_for_user
 */
package authbridge

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// TokenResult 与 PostgREST get_token/issue_token_for_user 对齐
type TokenResult struct {
	Success          bool                   `json:"success"`
	Message          string                 `json:"message"`
	AccessToken      string                 `json:"access_token"`
	RefreshToken     string                 `json:"refresh_token"`
	AccessExpiresIn  int                    `json:"access_expires_in"`
	ExpiresIn        int                    `json:"expires_in"`
	RefreshExpiresIn int                    `json:"refresh_expires_in"`
	Username         string                 `json:"username"`
	Roles            []string               `json:"roles"`
	Permissions      []string               `json:"permissions"`
	IsSuperuser      bool                   `json:"is_superuser"`
	UserInfo         map[string]interface{} `json:"user_info"`
}

// ExternalUser 外部用户信息
type ExternalUser struct {
	Username    string
	DisplayName string
	Email       string
	Source      string // oauth2 / sso / password
	LoginID     string
}

// LocalUserBridge 本地用户与 JWT 签发
type LocalUserBridge struct {
	cfg        Config
	httpClient *http.Client
}

// NewLocalUserBridge 创建桥接器
func NewLocalUserBridge(cfg Config) *LocalUserBridge {
	return &LocalUserBridge{
		cfg: cfg,
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

func (b *LocalUserBridge) callRPC(ctx context.Context, rpcName string, body interface{}, out interface{}) error {
	payload, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, b.cfg.PostgrestURL+"/rpc/"+rpcName, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept-Profile", "postgrest")
	req.Header.Set("Content-Profile", "postgrest")

	resp, err := b.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("调用 PostgREST %s 失败: %w", rpcName, err)
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("PostgREST %s HTTP %d: %s", rpcName, resp.StatusCode, string(data))
	}
	if err := json.Unmarshal(data, out); err != nil {
		return fmt.Errorf("解析 %s 响应失败: %w, body=%s", rpcName, err, string(data))
	}
	return nil
}

// EnsureAndIssueToken 确保本地用户存在并签发 JWT
func (b *LocalUserBridge) EnsureAndIssueToken(ctx context.Context, user ExternalUser) (*TokenResult, error) {
	if user.Username == "" {
		return nil, fmt.Errorf("外部用户名为空")
	}
	ensureResp := map[string]interface{}{}
	err := b.callRPC(ctx, "ensure_external_user", map[string]interface{}{
		"p_username":        user.Username,
		"p_display_name":    user.DisplayName,
		"p_email":           user.Email,
		"p_auth_source":     user.Source,
		"p_target_schemas":  b.cfg.DefaultSchemas,
	}, &ensureResp)
	if err != nil {
		return nil, err
	}
	if ok, _ := ensureResp["success"].(bool); !ok {
		msg, _ := ensureResp["message"].(string)
		return nil, fmt.Errorf("确保用户失败: %s", msg)
	}

	var token TokenResult
	if err := b.callRPC(ctx, "issue_token_for_user", map[string]interface{}{
		"p_username": user.Username,
	}, &token); err != nil {
		return nil, err
	}
	if !token.Success {
		return nil, fmt.Errorf("签发 token 失败: %s", token.Message)
	}
	if token.ExpiresIn == 0 && token.AccessExpiresIn > 0 {
		token.ExpiresIn = token.AccessExpiresIn
	}
	return &token, nil
}
