/*
 * @module service/basic_library/datasource/mock_lvyun_server
 * @description 模拟绿云酒店接口服务器，用于单元测试
 * @architecture 测试辅助工具 - HTTP服务器模拟
 * @documentReference test1.txt - 绿云通用查询接口说明
 * @stateFlow 启动服务器 -> 处理登录 -> 处理刷新 -> 处理查询 -> 处理退出
 * @rules 模拟真实的绿云接口行为，包括签名验证和sessionId管理
 * @dependencies net/http, encoding/json, crypto/sha1
 * @refs http_auth_test.go
 */

package datasource

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"sync"
	"time"
)

// MockLvyunServer 模拟绿云服务器
type MockLvyunServer struct {
	server      *httptest.Server
	sessions    map[string]*SessionInfo
	mu          sync.RWMutex
	appSecret   string
	validUsers  map[string]string // username -> password
	hotelGroups map[string]bool   // valid hotel group codes
}

// SessionInfo 会话信息
type SessionInfo struct {
	SessionID   string    `json:"sessionId"`
	Username    string    `json:"username"`
	GroupCode   string    `json:"groupCode"`
	CreatedAt   time.Time `json:"createdAt"`
	LastRefresh time.Time `json:"lastRefresh"`
}

// LoginRequest 登录请求
type LoginRequest struct {
	V              string `json:"v"`
	HotelGroupCode string `json:"hotelGroupCode"`
	Usercode       string `json:"usercode"`
	Password       string `json:"password"`
	Method         string `json:"method"`
	Local          string `json:"local"`
	Format         string `json:"format"`
	AppKey         string `json:"appKey"`
	Sign           string `json:"sign"`
}

// QueryRequest 查询请求
type QueryRequest struct {
	Method         string `json:"method"`
	V              string `json:"v"`
	Format         string `json:"format"`
	Local          string `json:"local"`
	AppKey         string `json:"appKey"`
	SessionID      string `json:"sessionId"`
	HotelGroupCode string `json:"hotelGroupCode"`
	HotelCode      string `json:"hotelCode"`
	Exec           string `json:"exec"`
	Params         string `json:"params"`
	Sign           string `json:"sign"`
}

// StandardResponse 标准响应格式
type StandardResponse struct {
	ResultCode string      `json:"resultCode"`
	ResultMsg  string      `json:"resultMsg"`
	ResultInfo interface{} `json:"resultInfo,omitempty"`
}

// ErrorResponse 错误响应格式
type ErrorResponse struct {
	ErrorToken string `json:"errorToken"`
	Code       string `json:"code"`
	Message    string `json:"message"`
	Solution   string `json:"solution"`
}

// QueryResult 查询结果
type QueryResult struct {
	ResultCode int                      `json:"resultCode"`
	ResultMsg  string                   `json:"resultMsg"`
	Result     []map[string]interface{} `json:"result"`
}

// NewMockLvyunServer 创建模拟绿云服务器
func NewMockLvyunServer(appSecret string) *MockLvyunServer {
	mock := &MockLvyunServer{
		sessions:    make(map[string]*SessionInfo),
		appSecret:   appSecret,
		validUsers:  make(map[string]string),
		hotelGroups: make(map[string]bool),
	}

	// 添加默认的测试用户和酒店集团
	mock.validUsers["test2"] = "123456"
	mock.validUsers["admin"] = "password"
	mock.hotelGroups["LYG"] = true
	mock.hotelGroups["DEFAULT"] = true

	// 创建HTTP服务器
	mux := http.NewServeMux()
	mux.HandleFunc("/ipmsgroup/router", mock.handleRouter)
	mock.server = httptest.NewServer(mux)

	return mock
}

// URL 获取服务器URL
func (m *MockLvyunServer) URL() string {
	return m.server.URL + "/ipmsgroup/router"
}

// Close 关闭服务器
func (m *MockLvyunServer) Close() {
	m.server.Close()
}

// AddUser 添加有效用户
func (m *MockLvyunServer) AddUser(username, password string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.validUsers[username] = password
}

// AddHotelGroup 添加有效酒店集团
func (m *MockLvyunServer) AddHotelGroup(groupCode string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.hotelGroups[groupCode] = true
}

// handleRouter 处理路由请求
func (m *MockLvyunServer) handleRouter(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 解析表单数据
	if err := r.ParseForm(); err != nil {
		m.sendErrorResponse(w, "PARSE_ERROR", "400", "参数解析失败", "请检查请求格式")
		return
	}

	method := r.FormValue("method")
	switch method {
	case "user.login":
		m.handleLogin(w, r)
	case "user.refresh":
		m.handleRefresh(w, r)
	case "user.logout":
		m.handleLogout(w, r)
	case "crs.kpi":
		m.handleQuery(w, r)
	default:
		m.sendErrorResponse(w, "UNKNOWN_METHOD", "400", "未知的方法: "+method, "请检查method参数")
	}
}

// handleLogin 处理登录请求
func (m *MockLvyunServer) handleLogin(w http.ResponseWriter, r *http.Request) {
	// 提取参数
	params := map[string]string{
		"v":              r.FormValue("v"),
		"hotelGroupCode": r.FormValue("hotelGroupCode"),
		"usercode":       r.FormValue("usercode"),
		"password":       r.FormValue("password"),
		"method":         r.FormValue("method"),
		"local":          r.FormValue("local"),
		"format":         r.FormValue("format"),
		"appKey":         r.FormValue("appKey"),
		"sign":           r.FormValue("sign"),
	}

	// 验证签名
	if !m.verifySign(params, params["sign"]) {
		m.sendErrorResponse(w, "SIGN_ERROR", "401", "签名验证失败", "请检查签名算法")
		return
	}

	// 验证用户凭据
	m.mu.RLock()
	validPassword, userExists := m.validUsers[params["usercode"]]
	groupValid := m.hotelGroups[params["hotelGroupCode"]]
	m.mu.RUnlock()

	if !userExists || validPassword != params["password"] {
		m.sendErrorResponse(w, "AUTH_ERROR", "401", "用户名或密码错误", "请检查用户凭据")
		return
	}

	if !groupValid {
		m.sendErrorResponse(w, "GROUP_ERROR", "400", "无效的酒店集团代码", "请检查hotelGroupCode参数")
		return
	}

	// 生成sessionId
	sessionID := m.generateSessionID(params["usercode"])
	sessionInfo := &SessionInfo{
		SessionID:   sessionID,
		Username:    params["usercode"],
		GroupCode:   params["hotelGroupCode"],
		CreatedAt:   time.Now(),
		LastRefresh: time.Now(),
	}

	m.mu.Lock()
	m.sessions[sessionID] = sessionInfo
	m.mu.Unlock()

	// 返回成功响应
	response := StandardResponse{
		ResultCode: "0",
		ResultMsg:  "成功",
		ResultInfo: sessionID,
	}

	m.sendJSONResponse(w, response)
}

// handleRefresh 处理刷新请求
func (m *MockLvyunServer) handleRefresh(w http.ResponseWriter, r *http.Request) {
	params := map[string]string{
		"v":         r.FormValue("v"),
		"sessionId": r.FormValue("sessionId"),
		"method":    r.FormValue("method"),
		"local":     r.FormValue("local"),
		"format":    r.FormValue("format"),
		"appKey":    r.FormValue("appKey"),
		"sign":      r.FormValue("sign"),
	}

	// 验证签名
	if !m.verifySign(params, params["sign"]) {
		m.sendErrorResponse(w, "SIGN_ERROR", "401", "签名验证失败", "请检查签名算法")
		return
	}

	// 验证sessionId
	m.mu.Lock()
	sessionInfo, exists := m.sessions[params["sessionId"]]
	if exists {
		// 更新刷新时间
		sessionInfo.LastRefresh = time.Now()
		// 生成新的sessionId
		newSessionID := m.generateSessionID(sessionInfo.Username)
		sessionInfo.SessionID = newSessionID
		// 更新映射
		delete(m.sessions, params["sessionId"])
		m.sessions[newSessionID] = sessionInfo
	}
	m.mu.Unlock()

	if !exists {
		m.sendErrorResponse(w, "SESSION_ERROR", "401", "无效的sessionId", "请重新登录")
		return
	}

	// 返回新的sessionId
	response := StandardResponse{
		ResultCode: "0",
		ResultMsg:  "刷新成功",
		ResultInfo: sessionInfo.SessionID,
	}

	m.sendJSONResponse(w, response)
}

// handleLogout 处理退出请求
func (m *MockLvyunServer) handleLogout(w http.ResponseWriter, r *http.Request) {
	params := map[string]string{
		"v":         r.FormValue("v"),
		"sessionId": r.FormValue("sessionId"),
		"method":    r.FormValue("method"),
		"local":     r.FormValue("local"),
		"format":    r.FormValue("format"),
		"appKey":    r.FormValue("appKey"),
		"sign":      r.FormValue("sign"),
	}

	// 验证签名
	if !m.verifySign(params, params["sign"]) {
		m.sendErrorResponse(w, "SIGN_ERROR", "401", "签名验证失败", "请检查签名算法")
		return
	}

	// 删除session
	m.mu.Lock()
	delete(m.sessions, params["sessionId"])
	m.mu.Unlock()

	response := StandardResponse{
		ResultCode: "0",
		ResultMsg:  "退出成功",
	}

	m.sendJSONResponse(w, response)
}

// handleQuery 处理查询请求
func (m *MockLvyunServer) handleQuery(w http.ResponseWriter, r *http.Request) {
	params := map[string]string{
		"method":         r.FormValue("method"),
		"v":              r.FormValue("v"),
		"format":         r.FormValue("format"),
		"local":          r.FormValue("local"),
		"appKey":         r.FormValue("appKey"),
		"sessionId":      r.FormValue("sessionId"),
		"hotelGroupCode": r.FormValue("hotelGroupCode"),
		"hotelCode":      r.FormValue("hotelCode"),
		"exec":           r.FormValue("exec"),
		"params":         r.FormValue("params"),
		"sign":           r.FormValue("sign"),
	}

	// 验证签名
	if !m.verifySign(params, params["sign"]) {
		m.sendErrorResponse(w, "SIGN_ERROR", "401", "签名验证失败", "请检查签名算法")
		return
	}

	// 验证sessionId
	m.mu.RLock()
	_, exists := m.sessions[params["sessionId"]]
	m.mu.RUnlock()

	if !exists {
		response := QueryResult{
			ResultCode: 1,
			ResultMsg:  "无效的sessionId，请重新登录",
			Result:     nil,
		}
		m.sendJSONResponse(w, response)
		return
	}

	// 根据exec参数返回模拟数据
	mockData := m.generateMockData(params["exec"], params["params"])

	response := QueryResult{
		ResultCode: 0,
		ResultMsg:  "查询成功",
		Result:     mockData,
	}

	m.sendJSONResponse(w, response)
}

// verifySign 验证签名
func (m *MockLvyunServer) verifySign(params map[string]string, providedSign string) bool {
	// 获取所有参数的key并排序
	var keys []string
	for key := range params {
		if key != "sign" { // 排除sign参数本身
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)

	// 构建签名字符串
	var signStr strings.Builder
	signStr.WriteString(m.appSecret)

	for _, key := range keys {
		signStr.WriteString(key)
		signStr.WriteString(params[key])
	}

	signStr.WriteString(m.appSecret)

	// SHA1加密并转换为大写
	hash := sha1.Sum([]byte(signStr.String()))
	expectedSign := strings.ToUpper(hex.EncodeToString(hash[:]))

	return expectedSign == providedSign
}

// generateSessionID 生成sessionId
func (m *MockLvyunServer) generateSessionID(username string) string {
	timestamp := time.Now().UnixNano()
	data := fmt.Sprintf("%s_%d_%s", username, timestamp, m.appSecret)
	hash := sha1.Sum([]byte(data))
	return hex.EncodeToString(hash[:])[:32] // 取前32位
}

// generateMockData 生成模拟数据
func (m *MockLvyunServer) generateMockData(exec, params string) []map[string]interface{} {
	switch exec {
	case "Kpi_Ihotel_Room_Total":
		return []map[string]interface{}{
			{
				"HotelCode":        "001",
				"HotelDesc":        "测试酒店",
				"RoomsTotalAmount": 100.0,
				"RoomsAvlAmount":   80.0,
				"RoomsSalesAmount": 20.0,
				"RoomsOccAmount":   0.2,
			},
		}
	case "Kpi_Ihotel_Room_Rank":
		return []map[string]interface{}{
			{
				"HotelCode":      "001",
				"HotelDesc":      "测试酒店",
				"RoomsRevAmount": 50000.0,
				"RoomsOccAmount": 0.85,
				"RoomsAdvAmount": 250.0,
			},
		}
	case "Kpi_Ihotel_Room_Adr_List":
		return []map[string]interface{}{
			{
				"HotelCode":      "001",
				"HotelDesc":      "测试酒店",
				"BizDate":        "2024-01-15T00:00:00Z",
				"RoomsAdrAmount": 280.0,
			},
			{
				"HotelCode":      "001",
				"HotelDesc":      "测试酒店",
				"BizDate":        "2024-01-16T00:00:00Z",
				"RoomsAdrAmount": 300.0,
			},
		}
	default:
		return []map[string]interface{}{
			{
				"message": "默认测试数据",
				"exec":    exec,
				"params":  params,
				"time":    time.Now().Format(time.RFC3339),
			},
		}
	}
}

// sendJSONResponse 发送JSON响应
func (m *MockLvyunServer) sendJSONResponse(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(data)
}

// sendErrorResponse 发送错误响应
func (m *MockLvyunServer) sendErrorResponse(w http.ResponseWriter, errorToken, code, message, solution string) {
	response := ErrorResponse{
		ErrorToken: errorToken,
		Code:       code,
		Message:    message,
		Solution:   solution,
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK) // 绿云接口即使错误也返回200
	json.NewEncoder(w).Encode(response)
}

// GetSessionCount 获取当前会话数量（用于测试）
func (m *MockLvyunServer) GetSessionCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.sessions)
}

// GetSession 获取指定sessionId的会话信息（用于测试）
func (m *MockLvyunServer) GetSession(sessionID string) *SessionInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if session, exists := m.sessions[sessionID]; exists {
		// 返回副本
		sessionCopy := *session
		return &sessionCopy
	}
	return nil
}
