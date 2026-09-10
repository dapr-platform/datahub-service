/*
 * @module service/authbridge/config
 * @description SSO 单点登录配置（本期只做 SSO，OAuth2 暂缓）
 */
package authbridge

import (
	"fmt"
	"os"
	"strings"
)

// Config SSO + 用户同步配置
type Config struct {
	SSOEnabled bool

	SSOServerURL     string // 如 https://tyzh.menghaikechuang.com/prod-api
	SSOAuthURL       string // 如 https://tyzh.menghaikechuang.com/sso/auth
	SSOClient        string
	SSOSecretKey     string
	SSORedirectURI   string // 不要带 #，例如 https://sjdz.menghaikechuang.com/datahub/
	SSOLogoutCallURL string

	// 用户主数据拉取（不做组织同步、不做中心推送）
	UserSyncEnabled  bool
	UserSyncSM4Key   string
	UserSyncSignKey  string
	UserSyncClient   string // clientName，默认与 SSO_CLIENT 相同
	UserSyncPageSize int
	UserSyncCron     string // 6 段 cron（含秒），默认每天 2:00

	PostgrestURL   string
	DefaultSchemas string
}

func envBool(key string, def bool) bool {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return def
	}
	return v == "true" || v == "1" || v == "yes"
}

func envOr(key, def string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return def
}

// NormalizeRedirectURI 去掉 fragment（#...），避免非法 redirect / 白名单匹配失败
func NormalizeRedirectURI(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return raw
	}
	if i := strings.Index(raw, "#"); i >= 0 {
		raw = raw[:i]
	}
	return raw
}

func envInt(key string, def int) int {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return def
	}
	var n int
	if _, err := fmt.Sscanf(v, "%d", &n); err != nil || n <= 0 {
		return def
	}
	return n
}

// LoadConfig 从环境变量加载配置
func LoadConfig() Config {
	client := envOr("SSO_CLIENT", "yqsjdz-client")
	return Config{
		SSOEnabled: envBool("SSO_ENABLED", false),

		SSOServerURL:     strings.TrimRight(envOr("SSO_SERVER_URL", "https://tyzh.menghaikechuang.com/prod-api"), "/"),
		SSOAuthURL:       envOr("SSO_AUTH_URL", "https://tyzh.menghaikechuang.com/sso/auth"),
		SSOClient:        client,
		SSOSecretKey:     envOr("SSO_SECRET_KEY", ""),
		SSORedirectURI:   NormalizeRedirectURI(envOr("SSO_REDIRECT_URI", "https://sjdz.menghaikechuang.com/datahub/")),
		SSOLogoutCallURL: envOr("SSO_LOGOUT_CALL_URL", ""),

		UserSyncEnabled:  envBool("USER_SYNC_ENABLED", false),
		UserSyncSM4Key:   envOr("USER_SYNC_SM4_KEY", ""),
		UserSyncSignKey:  envOr("USER_SYNC_SIGN_KEY", ""),
		UserSyncClient:   envOr("USER_SYNC_CLIENT_NAME", client),
		UserSyncPageSize: envInt("USER_SYNC_PAGE_SIZE", 10),
		UserSyncCron:     envOr("USER_SYNC_CRON", "0 0 2 * * *"),

		PostgrestURL:   strings.TrimRight(envOr("POSTGREST_URL", "http://postgrest:3000"), "/"),
		DefaultSchemas: envOr("AUTH_BRIDGE_DEFAULT_SCHEMAS", "public"),
	}
}
