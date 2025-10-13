package monitor_client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestLokiQuery(t *testing.T) {
	// 创建测试服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/loki/api/v1/query" {
			t.Errorf("期望路径 /loki/api/v1/query, 实际 %s", r.URL.Path)
		}

		query := r.URL.Query().Get("query")
		if query == "" {
			t.Error("query 参数不能为空")
		}

		resp := QueryResultResp{
			Status: "success",
			Data: QueryResult{
				Type: "streams",
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// 设置测试URL
	SetLokiUrl(server.URL)

	tests := []struct {
		name    string
		query   string
		limit   int
		wantErr bool
	}{
		{
			name:    "正常查询",
			query:   "{job=\"test\"}",
			limit:   100,
			wantErr: false,
		},
		{
			name:    "空查询字符串",
			query:   "",
			limit:   100,
			wantErr: true,
		},
		{
			name:    "零限制使用默认值",
			query:   "{job=\"test\"}",
			limit:   0,
			wantErr: false,
		},
		{
			name:    "负数限制使用默认值",
			query:   "{job=\"test\"}",
			limit:   -1,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			result, err := LokiQuery(ctx, tt.query, tt.limit)

			if (err != nil) != tt.wantErr {
				t.Errorf("LokiQuery() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && result == nil {
				t.Error("期望返回结果，但得到 nil")
			}
		})
	}
}

func TestLokiStreamQuery(t *testing.T) {
	// 创建测试服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/loki/api/v1/query_range" {
			t.Errorf("期望路径 /loki/api/v1/query_range, 实际 %s", r.URL.Path)
		}

		resp := LokiQueryResultResp{
			Status: "success",
			Data: LokiQueryResult{
				ResultType: "streams",
				Result:     []LokiResult{},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// 设置测试URL
	SetLokiUrl(server.URL)

	tests := []struct {
		name     string
		query    string
		limit    int
		preHours int
		wantErr  bool
	}{
		{
			name:     "正常查询",
			query:    "{job=\"test\"}",
			limit:    1000,
			preHours: 1,
			wantErr:  false,
		},
		{
			name:     "空查询字符串",
			query:    "",
			limit:    1000,
			preHours: 1,
			wantErr:  true,
		},
		{
			name:     "零限制使用默认值",
			query:    "{job=\"test\"}",
			limit:    0,
			preHours: 1,
			wantErr:  false,
		},
		{
			name:     "零小时使用默认值",
			query:    "{job=\"test\"}",
			limit:    1000,
			preHours: 0,
			wantErr:  false,
		},
		{
			name:     "负数小时使用默认值",
			query:    "{job=\"test\"}",
			limit:    1000,
			preHours: -1,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			result, err := LokiStreamQuery(ctx, tt.query, tt.limit, tt.preHours)

			if (err != nil) != tt.wantErr {
				t.Errorf("LokiStreamQuery() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && result == nil {
				t.Error("期望返回结果，但得到 nil")
			}
		})
	}
}

func TestLokiLabelValues(t *testing.T) {
	// 创建测试服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 验证路径格式
		if r.URL.Path != "/loki/api/v1/label/job/values" && r.URL.Path != "/loki/api/v1/label//values" {
			t.Logf("接收到路径: %s", r.URL.Path)
		}

		resp := LokiLabelValueResp{
			Status: "success",
			Data:   []string{"value1", "value2", "value3"},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// 设置测试URL
	SetLokiUrl(server.URL)

	tests := []struct {
		name    string
		label   string
		wantErr bool
		wantLen int
	}{
		{
			name:    "正常查询",
			label:   "job",
			wantErr: false,
			wantLen: 3,
		},
		{
			name:    "空标签",
			label:   "",
			wantErr: true,
			wantLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			result, err := LokiLabelValues(ctx, tt.label)

			if (err != nil) != tt.wantErr {
				t.Errorf("LokiLabelValues() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if len(result) != tt.wantLen {
					t.Errorf("LokiLabelValues() 返回长度 = %v, want %v", len(result), tt.wantLen)
				}
			}
		})
	}
}

func TestSetAndGetLokiUrl(t *testing.T) {
	originalUrl := GetLokiUrl()
	defer SetLokiUrl(originalUrl) // 恢复原始URL

	testUrl := "http://test.example.com:3100"
	SetLokiUrl(testUrl)

	if got := GetLokiUrl(); got != testUrl {
		t.Errorf("GetLokiUrl() = %v, want %v", got, testUrl)
	}
}

func TestLokiQueryWithTimeout(t *testing.T) {
	// 创建一个慢响应的服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		resp := QueryResultResp{
			Status: "success",
			Data:   QueryResult{Type: "streams"},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	SetLokiUrl(server.URL)

	// 创建一个带超时的上下文
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	_, err := LokiQuery(ctx, "{job=\"test\"}", 100)
	if err == nil {
		t.Error("期望超时错误，但没有收到错误")
	}
}

func TestLokiQueryErrorResponse(t *testing.T) {
	// 创建一个返回错误的服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := QueryResultResp{
			Status: "error",
			Data:   QueryResult{},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	SetLokiUrl(server.URL)

	ctx := context.Background()
	_, err := LokiQuery(ctx, "{job=\"test\"}", 100)
	if err == nil {
		t.Error("期望错误响应，但没有收到错误")
	}
}
