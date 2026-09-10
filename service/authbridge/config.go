/*
 * @module service/authbridge/config
 * @description SSO 单点登录配置（本期只做 SSO，OAuth2 暂缓）
 */
package authbridge

import (
	"os"
	"strings"
)

// Config SSO 配置
type Config struct {
	SSOEnabled bool

	SSOServerURL     string // 如 https://tyzh.menghaikechuang.com/prod-api
	SSOAuthURL       string // 如 https://tyzh.menghaikechuang.com/sso/auth
	SSOClient        string
	SSOSecretKey     string
	SSORedirectURI   string // 不要带 #，例如 https://sjdz.menghaikechuang.com/datahub/
	SSOLogoutCallURL string

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

// LoadConfig 从环境变量加载配置
func LoadConfig() Config {
	return Config{
		SSOEnabled: envBool("SSO_ENABLED", false),

		SSOServerURL:     strings.TrimRight(envOr("SSO_SERVER_URL", "https://tyzh.menghaikechuang.com/prod-api"), "/"),
		SSOAuthURL:       envOr("SSO_AUTH_URL", "https://tyzh.menghaikechuang.com/sso/auth"),
		SSOClient:        envOr("SSO_CLIENT", "yqsjdz-client"),
		SSOSecretKey:     envOr("SSO_SECRET_KEY", ""),
		SSORedirectURI:   NormalizeRedirectURI(envOr("SSO_REDIRECT_URI", "https://sjdz.menghaikechuang.com/datahub/")),
		SSOLogoutCallURL: envOr("SSO_LOGOUT_CALL_URL", ""),

		PostgrestURL:   strings.TrimRight(envOr("POSTGREST_URL", "http://postgrest:3000"), "/"),
		DefaultSchemas: envOr("AUTH_BRIDGE_DEFAULT_SCHEMAS", "public"),
	}
}
