// Package revocation 提供证书吊销服务（CRL/OCSP/CAIssuer）。
package revocation

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/globaltrusts/server-card/internal/ca"
	"github.com/globaltrusts/server-card/internal/storage"
)

// ServiceConfig 是吊销服务配置模型。
type ServiceConfig struct {
	UUID        string `json:"uuid"`
	CAUUID      string `json:"ca_uuid"`
	ServiceType string `json:"service_type"` // crl/ocsp/caissuer
	Path        string `json:"path"`         // 服务路径
	Enabled     bool   `json:"enabled"`
	CRLInterval int    `json:"crl_interval"` // CRL 更新间隔（分钟）
}

// OCSPStatus 是 OCSP 查询结果。
type OCSPStatus struct {
	SerialNumber string     `json:"serial_number"`
	Status       string     `json:"status"` // good/revoked/unknown
	RevokedAt    *time.Time `json:"revoked_at,omitempty"`
	Reason       int        `json:"reason,omitempty"`
}

// Service 是吊销服务管理。
type Service struct {
	db         *storage.DB
	caSvc      *ca.Service
	crlCache   map[string][]byte // caUUID -> CRL DER
	crlCacheMu sync.RWMutex
}

// NewService 创建吊销服务。
func NewService(db *storage.DB, caSvc *ca.Service) *Service {
	return &Service{
		db:       db,
		caSvc:    caSvc,
		crlCache: make(map[string][]byte),
	}
}

// CreateServiceConfig 创建吊销服务配置。
func (s *Service) CreateServiceConfig(ctx context.Context, cfg *ServiceConfig) error {
	if cfg.Path == "" {
		return fmt.Errorf("服务路径不能为空")
	}
	if cfg.CRLInterval <= 0 {
		cfg.CRLInterval = 60 // 默认 60 分钟
	}
	cfg.UUID = uuid.New().String()
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO revocation_services (uuid, ca_uuid, service_type, path, enabled, crl_interval)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		cfg.UUID, cfg.CAUUID, cfg.ServiceType, cfg.Path,
		boolToInt(cfg.Enabled), cfg.CRLInterval,
	)
	return err
}

// ListServiceConfigs 查询吊销服务配置列表。
func (s *Service) ListServiceConfigs(ctx context.Context, caUUID string) ([]*ServiceConfig, error) {
	query := `SELECT uuid, ca_uuid, service_type, path, enabled, crl_interval
		 FROM revocation_services`
	var args []interface{}
	if caUUID != "" {
		query += ` WHERE ca_uuid = ?`
		args = append(args, caUUID)
	}
	query += ` ORDER BY service_type`

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var configs []*ServiceConfig
	for rows.Next() {
		c := &ServiceConfig{}
		var enabled int
		if err := rows.Scan(&c.UUID, &c.CAUUID, &c.ServiceType,
			&c.Path, &enabled, &c.CRLInterval); err != nil {
			return nil, err
		}
		c.Enabled = enabled == 1
		configs = append(configs, c)
	}
	return configs, rows.Err()
}

// DeleteServiceConfig 删除吊销服务配置。
func (s *Service) DeleteServiceConfig(ctx context.Context, cfgUUID string) error {
	_, err := s.db.ExecContext(ctx,
		`DELETE FROM revocation_services WHERE uuid = ?`, cfgUUID,
	)
	return err
}

// GetCRL 获取 CA 的 CRL（优先从缓存获取）。
func (s *Service) GetCRL(ctx context.Context, caUUID string) ([]byte, error) {
	s.crlCacheMu.RLock()
	if crl, ok := s.crlCache[caUUID]; ok {
		s.crlCacheMu.RUnlock()
		return crl, nil
	}
	s.crlCacheMu.RUnlock()

	return s.RefreshCRL(ctx, caUUID)
}

// RefreshCRL 刷新 CA 的 CRL 缓存。
func (s *Service) RefreshCRL(ctx context.Context, caUUID string) ([]byte, error) {
	crl, err := s.caSvc.GenerateCRL(ctx, caUUID)
	if err != nil {
		return nil, err
	}

	s.crlCacheMu.Lock()
	s.crlCache[caUUID] = crl
	s.crlCacheMu.Unlock()

	return crl, nil
}

// QueryOCSPStatus 查询证书的 OCSP 状态。
func (s *Service) QueryOCSPStatus(ctx context.Context, caUUID, serialNumber string) (*OCSPStatus, error) {
	status := &OCSPStatus{
		SerialNumber: serialNumber,
		Status:       "good",
	}

	var revokedAt sql.NullTime
	var reason int
	err := s.db.QueryRowContext(ctx,
		`SELECT revoked_at, reason FROM revoked_certs
		 WHERE ca_uuid = ? AND serial_number = ?`,
		caUUID, serialNumber,
	).Scan(&revokedAt, &reason)

	if err == sql.ErrNoRows {
		return status, nil // 未吊销
	}
	if err != nil {
		return nil, fmt.Errorf("查询吊销状态失败: %w", err)
	}

	status.Status = "revoked"
	if revokedAt.Valid {
		status.RevokedAt = &revokedAt.Time
	}
	status.Reason = reason
	return status, nil
}

// GetCAIssuerCert 获取 CA 证书 PEM（用于 AIA CAIssuer）。
func (s *Service) GetCAIssuerCert(ctx context.Context, caUUID string) (string, error) {
	caObj, err := s.caSvc.GetByUUID(ctx, caUUID)
	if err != nil {
		return "", err
	}
	return caObj.CertPEM, nil
}

// StartCRLRefreshLoop 启动 CRL 定时刷新循环。
func (s *Service) StartCRLRefreshLoop(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				configs, err := s.ListServiceConfigs(ctx, "")
				if err != nil {
					slog.Error("获取吊销服务配置失败", "error", err)
					continue
				}
				for _, cfg := range configs {
					if cfg.ServiceType == "crl" && cfg.Enabled {
						if _, err := s.RefreshCRL(ctx, cfg.CAUUID); err != nil {
							slog.Error("刷新 CRL 失败",
								"ca_uuid", cfg.CAUUID, "error", err)
						} else {
							slog.Debug("CRL 已刷新", "ca_uuid", cfg.CAUUID)
						}
					}
				}
			}
		}
	}()
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}