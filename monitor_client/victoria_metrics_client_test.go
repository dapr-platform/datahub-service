package monitor_client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestQuery(t *testing.T) {
	// 创建测试服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/query" {
			t.Errorf("期望路径 /api/v1/query, 实际 %s", r.URL.Path)
		}

		query := r.URL.Query().Get("query")
		if query == "" {
			t.Error("query 参数不能为空")
		}

		resp := QueryResultResp{
			Status: "success",
			Data: QueryResult{
				Type: "vector",
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// 设置测试URL
	SetVictoriaMetricsUrl(server.URL)

	tests := []struct {
		name      string
		query     string
		queryTime time.Time
		wantErr   bool
	}{
		{
			name:      "正常查询",
			query:     "up",
			queryTime: time.Now(),
			wantErr:   false,
		},
		{
			name:      "空查询字符串",
			query:     "",
			queryTime: time.Now(),
			wantErr:   true,
		},
		{
			name:      "零时间使用当前时间",
			query:     "up",
			queryTime: time.Time{},
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			result, err := Query(ctx, tt.query, tt.queryTime)

			if (err != nil) != tt.wantErr {
				t.Errorf("Query() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && result == nil {
				t.Error("期望返回结果，但得到 nil")
			}
		})
	}
}

func TestQueryRange(t *testing.T) {
	// 创建测试服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/query_range" {
			t.Errorf("期望路径 /api/v1/query_range, 实际 %s", r.URL.Path)
		}

		resp := QueryResultResp{
			Status: "success",
			Data: QueryResult{
				Type: "matrix",
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// 设置测试URL
	SetVictoriaMetricsUrl(server.URL)

	now := time.Now()
	start := now.Add(-1 * time.Hour)
	end := now

	tests := []struct {
		name    string
		query   string
		start   time.Time
		end     time.Time
		step    time.Duration
		wantErr bool
	}{
		{
			name:    "正常查询",
			query:   "up",
			start:   start,
			end:     end,
			step:    15 * time.Second,
			wantErr: false,
		},
		{
			name:    "空查询字符串",
			query:   "",
			start:   start,
			end:     end,
			step:    15 * time.Second,
			wantErr: true,
		},
		{
			name:    "开始时间为零",
			query:   "up",
			start:   time.Time{},
			end:     end,
			step:    15 * time.Second,
			wantErr: true,
		},
		{
			name:    "结束时间为零",
			query:   "up",
			start:   start,
			end:     time.Time{},
			step:    15 * time.Second,
			wantErr: true,
		},
		{
			name:    "开始时间晚于结束时间",
			query:   "up",
			start:   end,
			end:     start,
			step:    15 * time.Second,
			wantErr: true,
		},
		{
			name:    "步长为0使用默认值",
			query:   "up",
			start:   start,
			end:     end,
			step:    0,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			result, err := QueryRange(ctx, tt.query, tt.start, tt.end, tt.step)

			if (err != nil) != tt.wantErr {
				t.Errorf("QueryRange() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && result == nil {
				t.Error("期望返回结果，但得到 nil")
			}
		})
	}
}

func TestFormatTime(t *testing.T) {
	tests := []struct {
		name string
		time time.Time
		want string
	}{
		{
			name: "Unix纪元时间",
			time: time.Unix(0, 0),
			want: "0",
		},
		{
			name: "特定时间",
			time: time.Unix(1640000000, 0),
			want: "1640000000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatTime(tt.time)
			if got != tt.want {
				t.Errorf("formatTime() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSetAndGetVictoriaMetricsUrl(t *testing.T) {
	originalUrl := GetVictoriaMetricsUrl()
	defer SetVictoriaMetricsUrl(originalUrl) // 恢复原始URL

	testUrl := "http://test.example.com:8428"
	SetVictoriaMetricsUrl(testUrl)

	if got := GetVictoriaMetricsUrl(); got != testUrl {
		t.Errorf("GetVictoriaMetricsUrl() = %v, want %v", got, testUrl)
	}
}

func TestQueryWithTimeout(t *testing.T) {
	// 创建一个慢响应的服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		resp := QueryResultResp{
			Status: "success",
			Data:   QueryResult{Type: "vector"},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	SetVictoriaMetricsUrl(server.URL)

	// 创建一个带超时的上下文
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	_, err := Query(ctx, "up", time.Now())
	if err == nil {
		t.Error("期望超时错误，但没有收到错误")
	}
}
