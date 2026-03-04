// Package cloud 提供 server-card 的 HTTP 客户端。
// Cloud Slot 通过此客户端与 server-card 通信。
package cloud

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client 是 server-card 的 HTTP 客户端。
type Client struct {
	baseURL    string
	token      string
	httpClient *http.Client
}

// NewClient 创建 server-card 客户端。
func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// SetToken 设置认证 Token。
func (c *Client) SetToken(token string) {
	c.token = token
}

// Token 返回当前 Token。
func (c *Client) Token() string {
	return c.token
}

// ---- 认证 API ----

// LoginResponse 是登录响应。
type LoginResponse struct {
	Token    string `json:"token"`
	UserUUID string `json:"user_uuid"`
	Username string `json:"username"`
}

// Login 登录 server-card，获取 JWT Token。
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

// Sign 请求 server-card 使用云端私钥签名。
func (c *Client) Sign(ctx context.Context, cardUUID, certUUID, mechanism string, data []byte) ([]byte, error) {
	body := map[string]interface{}{
		"cert_uuid":  certUUID,
		"mechanism":  mechanism,
		"data":       data,
	}
	var resp struct {
		Signature []byte `json:"signature"`
	}
	if err := c.post(ctx, fmt.Sprintf("/api/cards/%s/sign", cardUUID), body, &resp); err != nil {
		return nil, fmt.Errorf("云端签名失败: %w", err)
	}
	return resp.Signature, nil
}

// Decrypt 请求 server-card 使用云端私钥解密。
func (c *Client) Decrypt(ctx context.Context, cardUUID, certUUID, mechanism string, ciphertext []byte) ([]byte, error) {
	body := map[string]interface{}{
		"cert_uuid":   certUUID,
		"mechanism":   mechanism,
		"ciphertext":  ciphertext,
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

func (c *Client) do(req *http.Request, out interface{}) error {
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("HTTP 请求失败: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("读取响应失败: %w", err)
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
