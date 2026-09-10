/*
 * @module service/authbridge/sync_service
 * @description 用户拉取落库：全量/增量分页 → upsert_synced_user
 */
package authbridge

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// SyncMode 同步模式
type SyncMode string

const (
	SyncModeFull        SyncMode = "full"
	SyncModeIncremental SyncMode = "incremental"
)

// SyncResult 同步结果摘要
type SyncResult struct {
	Mode         SyncMode  `json:"mode"`
	StartedAt    time.Time `json:"started_at"`
	FinishedAt   time.Time `json:"finished_at"`
	Pages        int       `json:"pages"`
	Fetched      int       `json:"fetched"`
	Created      int       `json:"created"`
	Updated      int       `json:"updated"`
	Deactivated  int       `json:"deactivated"`
	Failed       int       `json:"failed"`
	UpdateTime   string    `json:"update_time,omitempty"`
	ErrorMessage string    `json:"error_message,omitempty"`
}

// UserSyncService 用户同步服务
type UserSyncService struct {
	cfg    Config
	client *UserSyncClient
	local  *LocalUserBridge

	mu         sync.Mutex
	running    bool
	lastResult *SyncResult
	lastOKAt   time.Time
}

// NewUserSyncService 创建服务
func NewUserSyncService(cfg Config) *UserSyncService {
	return &UserSyncService{
		cfg:    cfg,
		client: NewUserSyncClient(cfg),
		local:  NewLocalUserBridge(cfg),
	}
}

// LastResult 最近一次同步结果
func (s *UserSyncService) LastResult() *SyncResult {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.lastResult == nil {
		return nil
	}
	cp := *s.lastResult
	return &cp
}

// IsEnabled 是否启用
func (s *UserSyncService) IsEnabled() bool {
	return s.cfg.UserSyncEnabled
}

// Run 执行同步（互斥）
func (s *UserSyncService) Run(ctx context.Context, mode SyncMode) (*SyncResult, error) {
	if !s.cfg.UserSyncEnabled {
		return nil, fmt.Errorf("用户同步未启用（USER_SYNC_ENABLED=false）")
	}
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return nil, fmt.Errorf("同步正在进行中")
	}
	s.running = true
	s.mu.Unlock()
	defer func() {
		s.mu.Lock()
		s.running = false
		s.mu.Unlock()
	}()

	result := &SyncResult{
		Mode:      mode,
		StartedAt: time.Now(),
	}
	dataType := SyncDataTypeFull
	updateTime := ""
	if mode == SyncModeIncremental {
		dataType = SyncDataTypeIncremental
		s.mu.Lock()
		last := s.lastOKAt
		s.mu.Unlock()
		if last.IsZero() {
			last = time.Now().Add(-24 * time.Hour)
		}
		updateTime = last.Format("2006-01-02 15:04:05")
		result.UpdateTime = updateTime
	}

	pageSize := s.cfg.UserSyncPageSize
	if pageSize <= 0 {
		pageSize = 10
	}

	for page := 1; ; page++ {
		users, err := s.client.FetchPage(ctx, dataType, page, pageSize, updateTime)
		if err != nil {
			result.ErrorMessage = err.Error()
			result.FinishedAt = time.Now()
			s.storeResult(result, false)
			return result, err
		}
		result.Pages = page
		if len(users) == 0 {
			break
		}
		for _, u := range users {
			result.Fetched++
			created, deactivated, err := s.applyUser(ctx, u)
			if err != nil {
				result.Failed++
				slog.Warn("同步用户失败", "username", u.Username, "error", err)
				continue
			}
			if deactivated {
				result.Deactivated++
			} else if created {
				result.Created++
			} else {
				result.Updated++
			}
		}
		if len(users) < pageSize {
			break
		}
		// 防止异常死循环
		if page >= 500 {
			result.ErrorMessage = "超过最大页数限制"
			break
		}
	}

	result.FinishedAt = time.Now()
	ok := result.ErrorMessage == ""
	s.storeResult(result, ok)
	slog.Info("用户同步完成",
		"mode", mode,
		"fetched", result.Fetched,
		"created", result.Created,
		"updated", result.Updated,
		"deactivated", result.Deactivated,
		"failed", result.Failed,
	)
	return result, nil
}

func (s *UserSyncService) storeResult(result *SyncResult, ok bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	cp := *result
	s.lastResult = &cp
	if ok {
		s.lastOKAt = result.FinishedAt
	}
}

func (s *UserSyncService) applyUser(ctx context.Context, u RemoteUser) (created bool, deactivated bool, err error) {
	if u.Username == "" {
		return false, false, fmt.Errorf("用户名为空")
	}
	active := u.IsDel == 1
	extID := u.UserID.String()
	resp, err := s.local.UpsertSyncedUser(ctx, SyncedUser{
		Username:       u.Username,
		DisplayName:    u.TrueName,
		Email:          u.Email,
		ExternalUserID: extID,
		Active:         active,
	})
	if err != nil {
		return false, false, err
	}
	return resp.Created, !active, nil
}
