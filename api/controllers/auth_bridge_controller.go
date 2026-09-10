/*
 * @module api/controllers/auth_bridge_controller
 * @description SSO 单点登录桥接 HTTP 入口（OAuth2 暂缓）
 */
package controllers

import (
	"encoding/json"
	"net/http"

	"datahub-service/service/authbridge"

	"github.com/go-chi/render"
)

// AuthBridgeController SSO 桥接控制器
type AuthBridgeController struct {
	cfg   authbridge.Config
	local *authbridge.LocalUserBridge
	sso   *authbridge.SSOClient
}

// NewAuthBridgeController 创建控制器
func NewAuthBridgeController() *AuthBridgeController {
	cfg := authbridge.LoadConfig()
	return &AuthBridgeController{
		cfg:   cfg,
		local: authbridge.NewLocalUserBridge(cfg),
		sso:   authbridge.NewSSOClient(cfg),
	}
}

// Status 返回 SSO 状态
func (c *AuthBridgeController) Status(w http.ResponseWriter, r *http.Request) {
	render.JSON(w, r, SuccessResponse("ok", map[string]interface{}{
		"sso_enabled":   c.cfg.SSOEnabled,
		"sso_auth_url":  c.cfg.SSOAuthURL,
		"sso_client":    c.cfg.SSOClient,
		"sso_redirect":  c.cfg.SSORedirectURI,
		"sso_server_url": c.cfg.SSOServerURL,
	}))
}

// SSOAuthURL 返回 SSO 登录跳转地址
func (c *AuthBridgeController) SSOAuthURL(w http.ResponseWriter, r *http.Request) {
	if !c.cfg.SSOEnabled {
		render.JSON(w, r, BadRequestResponse("SSO 未启用", nil))
		return
	}
	redirect := authbridge.NormalizeRedirectURI(r.URL.Query().Get("redirect"))
	render.JSON(w, r, SuccessResponse("ok", map[string]string{
		"url": c.sso.BuildAuthURL(redirect),
	}))
}

type ticketReq struct {
	Ticket string `json:"ticket"`
}

// SSOLoginByTicket ticket 换本地 JWT
func (c *AuthBridgeController) SSOLoginByTicket(w http.ResponseWriter, r *http.Request) {
	if !c.cfg.SSOEnabled {
		render.JSON(w, r, BadRequestResponse("SSO 未启用", nil))
		return
	}
	var req ticketReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Ticket == "" {
		render.JSON(w, r, BadRequestResponse("ticket 不能为空", err))
		return
	}
	loginID, _, err := c.sso.CheckTicket(r.Context(), req.Ticket)
	if err != nil {
		render.JSON(w, r, InternalErrorResponse("ticket 校验失败: "+err.Error(), err))
		return
	}
	extUser, err := c.sso.FetchUserInfo(r.Context(), loginID)
	if err != nil {
		render.JSON(w, r, InternalErrorResponse("获取用户信息失败: "+err.Error(), err))
		return
	}
	token, err := c.local.EnsureAndIssueToken(r.Context(), *extUser)
	if err != nil {
		render.JSON(w, r, InternalErrorResponse("签发本地 token 失败: "+err.Error(), err))
		return
	}
	render.JSON(w, r, SuccessResponse("SSO 登录成功", token))
}

// SSOLogoutCall 认证中心单点注销回调
func (c *AuthBridgeController) SSOLogoutCall(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	loginID := r.FormValue("loginId")
	if loginID == "" {
		render.JSON(w, r, BadRequestResponse("loginId 为空", nil))
		return
	}
	render.JSON(w, r, SuccessResponse("ok", map[string]string{"loginId": loginID}))
}
