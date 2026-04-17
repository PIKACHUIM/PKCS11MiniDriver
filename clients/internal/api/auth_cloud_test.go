// Package api 内的 cloud-login 功能单元测试（不依赖真实网络、不初始化完整 Server）。
package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/globaltrusts/client-card/internal/storage"
)

// newTestAuthServer 创建一个最小化 Server 实例：只需要 userRepo 即可验证 cloud-login 流程，
// 不初始化 API Token/中间件/路由等无关依赖，避免测试环境污染。
func newTestAuthServer(t *testing.T) *Server {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	db, err := storage.Open(dbPath)
	if err != nil {
		t.Fatalf("打开测试数据库失败: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	return &Server{
		db:       db,
		userRepo: storage.NewUserRepo(db),
	}
}

// TestHandleCloudLogin_Success 验证：云端登录成功后，本地 users 表应新增 UserType=cloud 的记录，
// 并返回有效的本地 session token。
func TestHandleCloudLogin_Success(t *testing.T) {
	srv := newTestAuthServer(t)

	restore := SetCloudLoginDoer(func(ctx context.Context, cloudURL, username, password string) (*cloudLoginResp, error) {
		if username != "alice" || password != "pass123" {
			t.Fatalf("mock 收到非预期凭据: %s/%s", username, password)
		}
		return &cloudLoginResp{Token: "CLOUDTOKEN", UserUUID: "u1", Username: "alice", Role: "user"}, nil
	})
	t.Cleanup(func() { SetCloudLoginDoer(restore) })

	body := `{"cloud_url":"http://cloud.example.com","username":"alice","password":"pass123"}`
	req := httptest.NewRequest(http.MethodPost, "/api/auth/cloud-login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.handleCloudLogin(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("期望 200, 得到 %d: %s", rec.Code, rec.Body.String())
	}
	var resp struct {
		Code int `json:"code"`
		Data struct {
			Token    string `json:"token"`
			UserUUID string `json:"user_uuid"`
			Username string `json:"username"`
			UserType string `json:"user_type"`
			CloudURL string `json:"cloud_url"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}
	if resp.Data.Token == "" {
		t.Fatalf("未返回本地 session token")
	}
	if resp.Data.UserType != "cloud" {
		t.Fatalf("user_type 应为 cloud, 实际 %s", resp.Data.UserType)
	}
	if !strings.Contains(resp.Data.Username, "alice") {
		t.Fatalf("本地 username 应包含 alice, 实际 %s", resp.Data.Username)
	}

	u, err := srv.userRepo.GetByUUID(context.Background(), resp.Data.UserUUID)
	if err != nil || u == nil {
		t.Fatalf("用户未写入数据库: err=%v", err)
	}
	if u.UserType != storage.UserTypeCloud {
		t.Fatalf("落库用户 UserType 错误: %s", u.UserType)
	}
	if string(u.AuthToken) != "CLOUDTOKEN" {
		t.Fatalf("未保存云端 token")
	}

	if s := getSession(resp.Data.Token); s == nil {
		t.Fatalf("本地 session token 无效")
	}
}

// TestHandleCloudLogin_Idempotent 验证：同一 cloud_url+username 重复登录时，不会重复创建用户。
func TestHandleCloudLogin_Idempotent(t *testing.T) {
	srv := newTestAuthServer(t)
	restore := SetCloudLoginDoer(func(ctx context.Context, cloudURL, username, password string) (*cloudLoginResp, error) {
		return &cloudLoginResp{Token: "T2", UserUUID: "u2", Username: "bob", Role: "user"}, nil
	})
	t.Cleanup(func() { SetCloudLoginDoer(restore) })

	body := `{"cloud_url":"https://c.example.com:8443","username":"bob","password":"x"}`
	for i := 0; i < 3; i++ {
		req := httptest.NewRequest(http.MethodPost, "/api/auth/cloud-login", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		srv.handleCloudLogin(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("第 %d 次登录失败: %d %s", i, rec.Code, rec.Body.String())
		}
	}

	users, err := srv.userRepo.List(context.Background())
	if err != nil {
		t.Fatalf("列用户失败: %v", err)
	}
	cloudCount := 0
	for _, u := range users {
		if u.UserType == storage.UserTypeCloud && strings.Contains(u.Username, "bob") {
			cloudCount++
		}
	}
	if cloudCount != 1 {
		t.Fatalf("bob 应只有 1 条云端记录, 实际 %d", cloudCount)
	}
}

// TestHandleCloudLogin_UpstreamFailure 验证：云端拒绝登录时返回 401。
func TestHandleCloudLogin_UpstreamFailure(t *testing.T) {
	srv := newTestAuthServer(t)
	restore := SetCloudLoginDoer(func(ctx context.Context, cloudURL, username, password string) (*cloudLoginResp, error) {
		return nil, errDummy
	})
	t.Cleanup(func() { SetCloudLoginDoer(restore) })

	body := `{"cloud_url":"http://c","username":"x","password":"x"}`
	req := httptest.NewRequest(http.MethodPost, "/api/auth/cloud-login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.handleCloudLogin(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("云端失败应返回 401, 实际 %d", rec.Code)
	}
}

// TestHandleCloudLogin_BadRequest 验证缺少必填参数时返回 400。
func TestHandleCloudLogin_BadRequest(t *testing.T) {
	srv := newTestAuthServer(t)

	cases := []struct {
		name string
		body string
	}{
		{"empty_all", `{}`},
		{"missing_password", `{"cloud_url":"http://c","username":"x"}`},
		{"bad_json", `not-a-json`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/auth/cloud-login", strings.NewReader(c.body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			srv.handleCloudLogin(rec, req)
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("期望 400, 实际 %d", rec.Code)
			}
		})
	}
}

// TestHandleLogoutAll 验证全量清空 session。
func TestHandleLogoutAll(t *testing.T) {
	srv := newTestAuthServer(t)

	u := &storage.User{UUID: "u-a", Username: "a", Role: "user"}
	_ = newSessionToken(u)
	_ = newSessionToken(u)

	req := httptest.NewRequest(http.MethodDelete, "/api/auth/logout-all", nil)
	rec := httptest.NewRecorder()
	srv.handleLogoutAll(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("期望 200, 实际 %d", rec.Code)
	}
	sessionMu.RLock()
	remaining := len(sessions)
	sessionMu.RUnlock()
	if remaining != 0 {
		t.Fatalf("session 未清空, 剩余 %d", remaining)
	}
}

// errString 是轻量的错误类型，用于测试场景伪造"云端拒绝"。
type errString string

func (e errString) Error() string { return string(e) }

var errDummy = errString("invalid credentials")
