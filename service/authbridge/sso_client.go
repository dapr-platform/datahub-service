/*
 * @module service/authbridge/sso_client
 * @description 萌海 Sa-Token SSO 客户端（pushS: checkTicket / userinfo / signout）
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

// SSOClient SSO HTTP 客户端
type SSOClient struct {
	cfg        Config
	httpClient *http.Client
}

// NewSSOClient 创建 SSO 客户端
func NewSSOClient(cfg Config) *SSOClient {
	return &SSOClient{
		cfg: cfg,
		httpClient: &http.Client{
			Timeout: 20 * time.Second,
		},
	}
}

// BuildAuthURL 构建跳转统一账号 SSO 登录页的 URL
// redirect 不含 #；由 url.Values 自动做 query escape
func (c *SSOClient) BuildAuthURL(redirectURI string) string {
	if redirectURI == "" {
		redirectURI = c.cfg.SSORedirectURI
	}
	redirectURI = NormalizeRedirectURI(redirectURI)
	u, err := url.Parse(c.cfg.SSOAuthURL)
	if err != nil {
		return c.cfg.SSOAuthURL + "?client=" + url.QueryEscape(c.cfg.SSOClient) + "&redirect=" + url.QueryEscape(redirectURI)
	}
	q := u.Query()
	q.Set("client", c.cfg.SSOClient)
	q.Set("redirect", redirectURI)
	u.RawQuery = q.Encode()
	return u.String()
}

type pushSResponse struct {
	Code       int             `json:"code"`
	Msg        string          `json:"msg"`
	Data       json.RawMessage `json:"data"`
	LoginID    interface{}     `json:"loginId"`
	TokenValue string          `json:"tokenValue"`
}

// CheckTicket 通过 /sso/pushS msgType=checkTicket 校验 ticket
func (c *SSOClient) CheckTicket(ctx context.Context, ticket string) (loginID string, tokenValue string, err error) {
	params := map[string]string{
		"msgType": "checkTicket",
		"ticket":  ticket,
		"client":  c.cfg.SSOClient,
	}
	if c.cfg.SSOLogoutCallURL != "" {
		params["ssoLogoutCall"] = c.cfg.SSOLogoutCallURL
	}
	var resp pushSResponse
	if err := c.pushS(ctx, params, &resp); err != nil {
		return "", "", err
	}
	if resp.Code != 200 {
		return "", "", fmt.Errorf("checkTicket 失败: %s", resp.Msg)
	}
	loginID = stringifyID(resp.LoginID)
	if loginID == "" && len(resp.Data) > 0 {
		var asStr string
		if json.Unmarshal(resp.Data, &asStr) == nil {
			loginID = asStr
		}
	}
	if loginID == "" {
		return "", "", fmt.Errorf("checkTicket 未返回 loginId")
	}
	return loginID, resp.TokenValue, nil
}

// FetchUserInfo 通过 /sso/pushS msgType=userinfo 拉取用户资料
func (c *SSOClient) FetchUserInfo(ctx context.Context, loginID string) (*ExternalUser, error) {
	params := map[string]string{
		"msgType": "userinfo",
		"loginId": loginID,
		"client":  c.cfg.SSOClient,
	}
	var resp pushSResponse
	if err := c.pushS(ctx, params, &resp); err != nil {
		return nil, err
	}
	if resp.Code != 200 {
		return nil, fmt.Errorf("userinfo 失败: %s", resp.Msg)
	}
	var data map[string]interface{}
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		return nil, fmt.Errorf("解析 userinfo 失败: %w", err)
	}
	username := firstString(data, "username", "loginName", "userName", "account")
	if username == "" {
		username = loginID
	}
	display := firstString(data, "trueName", "nickname", "name", "displayName")
	email := firstString(data, "email")
	return &ExternalUser{
		Username:    username,
		DisplayName: display,
		Email:       email,
		Source:      "sso",
		LoginID:     loginID,
	}, nil
}

// SignOut 通知认证中心注销（尽力而为）
func (c *SSOClient) SignOut(ctx context.Context, loginID string) error {
	params := map[string]string{
		"msgType": "signout",
		"loginId": loginID,
		"client":  c.cfg.SSOClient,
	}
	var resp pushSResponse
	if err := c.pushS(ctx, params, &resp); err != nil {
		return err
	}
	if resp.Code != 200 {
		return fmt.Errorf("signout 失败: %s", resp.Msg)
	}
	return nil
}

func (c *SSOClient) pushS(ctx context.Context, params map[string]string, out *pushSResponse) error {
	params["timestamp"] = strconv.FormatInt(time.Now().UnixMilli(), 10)
	params["nonce"] = strings.ReplaceAll(uuid.NewString(), "-", "")
	params["sign"] = BuildSaSign(params, c.cfg.SSOSecretKey)

	form := url.Values{}
	for k, v := range params {
		form.Set(k, v)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.cfg.SSOServerURL+"/sso/pushS", strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("请求 pushS 失败: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("解析 pushS 响应失败: %w, body=%s", err, string(body))
	}
	return nil
}

func stringifyID(v interface{}) string {
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		return t
	case float64:
		return strconv.FormatInt(int64(t), 10)
	case json.Number:
		return t.String()
	default:
		return fmt.Sprintf("%v", t)
	}
}

func firstString(m map[string]interface{}, keys ...string) string {
	for _, k := range keys {
		if v, ok := m[k]; ok && v != nil {
			s := strings.TrimSpace(fmt.Sprintf("%v", v))
			if s != "" && s != "<nil>" {
				return s
			}
		}
	}
	return ""
}
