/*
 * @module service/monitoring/notification
 * @description 通知渠道接口和实现，为告警管理器提供多种通知发送能力
 * @architecture 分层架构 - 业务服务层
 * @documentReference ai_docs/basic_library_process_impl.md
 * @stateFlow 通知配置 -> 通知发送 -> 状态跟踪
 * @rules 确保通知发送的可靠性和及时性
 * @dependencies datahub-service/service/models
 * @refs ai_docs/patch_basic_library_process.md
 */

package monitoring

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// NotificationSender 通知发送器接口
type NotificationSender interface {
	Send(alert *Alert) error
	GetChannelType() string
	IsEnabled() bool
	Configure(config map[string]interface{}) error
}

// EmailNotificationChannel 邮件通知渠道
type EmailNotificationChannel struct {
	SMTPServer  string   `json:"smtp_server"`
	SMTPPort    int      `json:"smtp_port"`
	Username    string   `json:"username"`
	Password    string   `json:"password"`
	FromAddress string   `json:"from_address"`
	ToAddresses []string `json:"to_addresses"`
	Enabled     bool     `json:"is_enabled"`
}

// Send 发送邮件通知
func (e *EmailNotificationChannel) Send(alert *Alert) error {
	if !e.Enabled {
		return fmt.Errorf("邮件通知渠道未启用")
	}

	subject := fmt.Sprintf("[%s] %s", alert.Severity, alert.Message)
	body := e.buildEmailBody(alert)

	// 简化实现 - 实际应该使用SMTP发送邮件
	fmt.Printf("发送邮件: To=%v, Subject=%s\n", e.ToAddresses, subject)
	fmt.Printf("Body:\n%s\n", body)

	return nil
}

// GetChannelType 获取渠道类型
func (e *EmailNotificationChannel) GetChannelType() string {
	return "email"
}

// IsEnabled 检查是否启用
func (e *EmailNotificationChannel) IsEnabled() bool {
	return e.Enabled
}

// Configure 配置邮件渠道
func (e *EmailNotificationChannel) Configure(config map[string]interface{}) error {
	if server, ok := config["smtp_server"].(string); ok {
		e.SMTPServer = server
	}
	if port, ok := config["smtp_port"].(float64); ok {
		e.SMTPPort = int(port)
	}
	if username, ok := config["username"].(string); ok {
		e.Username = username
	}
	if password, ok := config["password"].(string); ok {
		e.Password = password
	}
	if from, ok := config["from_address"].(string); ok {
		e.FromAddress = from
	}
	if to, ok := config["to_addresses"].([]interface{}); ok {
		e.ToAddresses = make([]string, len(to))
		for i, addr := range to {
			if str, ok := addr.(string); ok {
				e.ToAddresses[i] = str
			}
		}
	}
	if enabled, ok := config["is_enabled"].(bool); ok {
		e.Enabled = enabled
	}

	return nil
}

// 构建邮件正文
func (e *EmailNotificationChannel) buildEmailBody(alert *Alert) string {
	body := fmt.Sprintf(`
告警详情：
- 告警ID: %s
- 告警规则: %s
- 严重性: %s
- 状态: %s
- 描述: %s
- 触发时间: %s
- 指标值: %v
- 阈值: %v

来源信息：
- 源: %s
- 对象ID: %s
- 对象类型: %s

`, alert.ID, alert.RuleName, alert.Severity, alert.Status, alert.Description,
		alert.TriggeredAt.Format(time.RFC3339), alert.MetricValue, alert.Threshold,
		alert.Source, alert.ObjectID, alert.ObjectType)

	if len(alert.Labels) > 0 {
		body += "\n标签:\n"
		for k, v := range alert.Labels {
			body += fmt.Sprintf("- %s: %s\n", k, v)
		}
	}

	if len(alert.Annotations) > 0 {
		body += "\n注释:\n"
		for k, v := range alert.Annotations {
			body += fmt.Sprintf("- %s: %v\n", k, v)
		}
	}

	return body
}

// WebhookNotificationChannel Webhook通知渠道
type WebhookNotificationChannel struct {
	URL     string            `json:"url"`
	Method  string            `json:"method"`
	Headers map[string]string `json:"headers"`
	Timeout time.Duration     `json:"timeout"`
	Enabled bool              `json:"is_enabled"`
}

// Send 发送Webhook通知
func (w *WebhookNotificationChannel) Send(alert *Alert) error {
	if !w.Enabled {
		return fmt.Errorf("Webhook通知渠道未启用")
	}

	payload, err := json.Marshal(alert)
	if err != nil {
		return fmt.Errorf("序列化告警数据失败: %v", err)
	}

	req, err := http.NewRequest(w.Method, w.URL, bytes.NewBuffer(payload))
	if err != nil {
		return fmt.Errorf("创建HTTP请求失败: %v", err)
	}

	// 设置头部
	req.Header.Set("Content-Type", "application/json")
	for k, v := range w.Headers {
		req.Header.Set(k, v)
	}

	client := &http.Client{Timeout: w.Timeout}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("发送Webhook通知失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("Webhook通知响应错误: %d", resp.StatusCode)
	}

	return nil
}

// GetChannelType 获取渠道类型
func (w *WebhookNotificationChannel) GetChannelType() string {
	return "webhook"
}

// IsEnabled 检查是否启用
func (w *WebhookNotificationChannel) IsEnabled() bool {
	return w.Enabled
}

// Configure 配置Webhook渠道
func (w *WebhookNotificationChannel) Configure(config map[string]interface{}) error {
	if url, ok := config["url"].(string); ok {
		w.URL = url
	}
	if method, ok := config["method"].(string); ok {
		w.Method = method
	}
	if headers, ok := config["headers"].(map[string]interface{}); ok {
		w.Headers = make(map[string]string)
		for k, v := range headers {
			if str, ok := v.(string); ok {
				w.Headers[k] = str
			}
		}
	}
	if timeout, ok := config["timeout"].(float64); ok {
		w.Timeout = time.Duration(timeout) * time.Second
	}
	if enabled, ok := config["is_enabled"].(bool); ok {
		w.Enabled = enabled
	}

	return nil
}
