package api

import (
	"net/http"
	"time"

	"github.com/globaltrusts/client-card/internal/card/cloud"
	"github.com/globaltrusts/client-card/internal/storage"
)

// cloudSyncStatus 记录最近一次同步状态。
var cloudSyncStatus = struct {
	LastSync    *time.Time `json:"last_sync,omitempty"`
	LastError   string     `json:"last_error,omitempty"`
	SyncedCards int        `json:"synced_cards"`
	SyncedCerts int        `json:"synced_certs"`
}{}

// handleCloudSync POST /api/cloud/sync
// 手动触发云端同步：从云端拉取最新卡片和证书信息。
func (s *Server) handleCloudSync(w http.ResponseWriter, r *http.Request) {
	// 查找所有 Cloud Slot 类型的卡片
	cards, err := s.cardRepo.ListAll(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "查询卡片失败: "+err.Error())
		return
	}

	syncedCards := 0
	syncedCerts := 0
	var lastErr string

	for _, c := range cards {
		if c.SlotType != storage.SlotTypeCloud {
			continue
		}
		if c.CloudURL == "" || c.CloudCardUUID == "" {
			continue
		}

		// 创建云端客户端
		client, err := cloud.NewClient(c.CloudURL, false)
		if err != nil {
			lastErr = "创建云端客户端失败: " + err.Error()
			continue
		}

		// 获取用户 token（从卡片关联的用户获取）
		user, err := s.userRepo.GetByUUID(r.Context(), c.UserUUID)
		if err != nil || user == nil {
			continue
		}
		if len(user.AuthToken) > 0 {
			client.SetToken(string(user.AuthToken))
		}

		// 拉取云端证书列表
		cloudCerts, err := client.ListCerts(r.Context(), c.CloudCardUUID)
		if err != nil {
			lastErr = "拉取云端证书失败: " + err.Error()
			continue
		}

		syncedCards++
		syncedCerts += len(cloudCerts)
	}

	now := time.Now()
	cloudSyncStatus.LastSync = &now
	cloudSyncStatus.LastError = lastErr
	cloudSyncStatus.SyncedCards = syncedCards
	cloudSyncStatus.SyncedCerts = syncedCerts

	writeOK(w, map[string]interface{}{
		"synced_cards": syncedCards,
		"synced_certs": syncedCerts,
		"last_error":   lastErr,
		"synced_at":    now,
	})
}

// handleCloudStatus GET /api/cloud/status
// 返回云端同步状态。
func (s *Server) handleCloudStatus(w http.ResponseWriter, r *http.Request) {
	writeOK(w, cloudSyncStatus)
}
