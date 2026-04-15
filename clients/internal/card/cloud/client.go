// Package cloud 提供 servers 的 HTTP 客户端。
// Cloud Slot 通过此客户端与 servers 通信。
package cloud

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"
)

// Client 是 servers（OpenCert Platform）的 HTTP 客户端。
type Client struct {
	mu             sync.RWMutex
	baseURL        string
	token          string
	tokenExpiresAt time.Time // Token 过期时间
	httpClient     *http.Client
	allowInsecure  bool // 是否允许非 HTTPS（仅开发环境）
}

// NewClient 创建 servers 客户端。
// baseURL 生产环境必须为 https://，开发环境可通过 allowInsecure=true 允许 http://。
func NewClient(baseURL string, allowInsecure bool) (*Client, error) {
	// 校验 URL scheme
	if !strings.HasPrefix(baseURL, "https://") {
		if !allowInsecure {
			return nil, fmt.Errorf("Cloud Slot 要求 HTTPS 连接，当前 URL: %s（如需开发环境使用 HTTP，请设置 allow_insecure_cloud=true）", baseURL)
		}
		slog.Warn("⚠️ Cloud Slot 使用不安全的 HTTP 连接", "url", baseURL)
	}

	// 配置 TLS：不跳过证书验证
	tlsConfig := &tls.Config{
		InsecureSkipVerify: false,
		MinVersion:         tls.VersionTLS12,
	}

	transport := &http.Transport{
		TLSClientConfig: tlsConfig,
	}

	return &Client{
		baseURL:       strings.TrimRight(baseURL, "/"),
		allowInsecure: allowInsecure,
		httpClient: &http.Client{
			Timeout:   30 * time.Second,
			Transport: transport,
		},
	}, nil
}

// SetToken 设置认证 Token 和过期时间。
func (c *Client) SetToken(token string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.token = token
	// 默认设置 Token 有效期为 1 小时
	c.tokenExpiresAt = time.Now().Add(1 * time.Hour)
}

// SetTokenWithExpiry 设置认证 Token 和自定义过期时间。
func (c *Client) SetTokenWithExpiry(token string, expiresAt time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.token = token
	c.tokenExpiresAt = expiresAt
}

// Token 返回当前 Token。
func (c *Client) Token() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.token
}

// IsTokenExpiringSoon 检查 Token 是否即将过期（剩余时间 < 总时间的 20%）。
func (c *Client) IsTokenExpiringSoon() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.token == "" {
		return false
	}
	remaining := time.Until(c.tokenExpiresAt)
	return remaining < 12*time.Minute // 1小时的20%
}

// IsTokenExpired 检查 Token 是否已过期。
func (c *Client) IsTokenExpired() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.token == "" {
		return true
	}
	return time.Now().After(c.tokenExpiresAt)
}

// EnsureToken 确保 Token 有效，过期或即将过期时自动 refresh。
func (c *Client) EnsureToken(ctx context.Context) error {
	if c.IsTokenExpired() || c.IsTokenExpiringSoon() {
		_, err := c.Refresh(ctx)
		if err != nil {
			slog.Warn("JWT Token 刷新失败", "error", err)
			return fmt.Errorf("token refresh 失败: %w", err)
		}
		slog.Debug("JWT Token 已自动刷新")
	}
	return nil
}

// ---- 认证 API ----

// LoginResponse 是登录响应。
type LoginResponse struct {
	Token    string `json:"token"`
	UserUUID string `json:"user_uuid"`
	Username string `json:"username"`
}

// Login 登录 servers，获取 JWT Token。
func (c *Client) Login(ctx context.Context, username, password string) (*LoginResponse, error) {
	body := map[string]string{"username": username, "password": password}
	var resp LoginResponse
	if err := c.post(ctx, "/api/auth/login", body, &resp); err != nil {
		return nil, fmt.Errorf("登录失败: %w", err)
	}
	c.token = resp.Token
	return &resp, nil
}

// Refresh 刷新 Token。
func (c *Client) Refresh(ctx context.Context) (*LoginResponse, error) {
	var resp LoginResponse
	if err := c.post(ctx, "/api/auth/refresh", nil, &resp); err != nil {
		return nil, fmt.Errorf("刷新 Token 失败: %w", err)
	}
	c.token = resp.Token
	return &resp, nil
}

// ---- 卡片 API ----

// Card 是云端卡片信息。
type Card struct {
	UUID      string    `json:"uuid"`
	UserUUID  string    `json:"user_uuid"`
	CardName  string    `json:"card_name"`
	Remark    string    `json:"remark"`
	CreatedAt time.Time `json:"created_at"`
}

// ListCards 列出用户的所有云端卡片。
func (c *Client) ListCards(ctx context.Context) ([]*Card, error) {
	var resp struct {
		Cards []*Card `json:"cards"`
	}
	if err := c.get(ctx, "/api/cards", &resp); err != nil {
		return nil, fmt.Errorf("获取卡片列表失败: %w", err)
	}
	return resp.Cards, nil
}

// ---- 证书 API ----

// Cert 是云端证书信息（不含私钥）。
type Cert struct {
	UUID        string    `json:"uuid"`
	CardUUID    string    `json:"card_uuid"`
	CertType    string    `json:"cert_type"`
	KeyType     string    `json:"key_type"`
	CertContent []byte    `json:"cert_content"`
	Remark      string    `json:"remark"`
	CreatedAt   time.Time `json:"created_at"`
}

// ListCerts 列出卡片的所有证书。
func (c *Client) ListCerts(ctx context.Context, cardUUID string) ([]*Cert, error) {
	var resp struct {
		Certs []*Cert `json:"certs"`
	}
	if err := c.get(ctx, fmt.Sprintf("/api/cards/%s/certs", cardUUID), &resp); err != nil {
		return nil, fmt.Errorf("获取证书列表失败: %w", err)
	}
	return resp.Certs, nil
}

// ---- 签名/解密 API ----

// Sign 请求 servers 使用云端私钥签名。
func (c *Client) Sign(ctx context.Context, cardUUID, certUUID, mechanism string, data []byte) ([]byte, error) {
	body := map[string]interface{}{
		"cert_uuid": certUUID,
		"mechanism": mechanism,
		"data":      data,
	}
	var resp struct {
		Signature []byte `json:"signature"`
	}
	if err := c.post(ctx, fmt.Sprintf("/api/cards/%s/sign", cardUUID), body, &resp); err != nil {
		return nil, fmt.Errorf("云端签名失败: %w", err)
	}
	return resp.Signature, nil
}

// Decrypt 请求 servers 使用云端私钥解密。
func (c *Client) Decrypt(ctx context.Context, cardUUID, certUUID, mechanism string, ciphertext []byte) ([]byte, error) {
	body := map[string]interface{}{
		"cert_uuid":  certUUID,
		"mechanism":  mechanism,
		"ciphertext": ciphertext,
	}
	var resp struct {
		Plaintext []byte `json:"plaintext"`
	}
	if err := c.post(ctx, fmt.Sprintf("/api/cards/%s/decrypt", cardUUID), body, &resp); err != nil {
		return nil, fmt.Errorf("云端解密失败: %w", err)
	}
	return resp.Plaintext, nil
}

// ---- HTTP 工具方法 ----

func (c *Client) get(ctx context.Context, path string, out interface{}) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return err
	}
	return c.do(req, out)
}

func (c *Client) post(ctx context.Context, path string, body interface{}, out interface{}) error {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("序列化请求体失败: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bodyReader)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	return c.do(req, out)
}

// ErrNetworkUnavailable 表示网络不可用。
var ErrNetworkUnavailable = fmt.Errorf("网络不可用")

func (c *Client) do(req *http.Request, out interface{}) error {
	c.mu.RLock()
	token := c.token
	c.mu.RUnlock()

	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		// 判断是否为网络不可用
		return fmt.Errorf("%w: %v", ErrNetworkUnavailable, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("读取响应失败: %w", err)
	}

	// 401 表示 Token 无效/过期
	if resp.StatusCode == http.StatusUnauthorized {
		return fmt.Errorf("认证失败: Token 无效或已过期")
	}

	if resp.StatusCode >= 400 {
		var errResp struct {
			Error string `json:"error"`
		}
		json.Unmarshal(respBody, &errResp)
		if errResp.Error != "" {
			return fmt.Errorf("服务端错误 [%d]: %s", resp.StatusCode, errResp.Error)
		}
		return fmt.Errorf("HTTP 错误: %d", resp.StatusCode)
	}

	if out != nil {
		if err := json.Unmarshal(respBody, out); err != nil {
			return fmt.Errorf("解析响应失败: %w", err)
		}
	}
	return nil
}

// IsNetworkError 判断错误是否为网络不可用。
func IsNetworkError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "网络不可用") ||
		strings.Contains(err.Error(), "connection refused") ||
		strings.Contains(err.Error(), "no such host") ||
		strings.Contains(err.Error(), "i/o timeout")
}
