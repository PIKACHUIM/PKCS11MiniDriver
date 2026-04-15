package storage_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	config "github.com/globaltrusts/client-card/configs"
	"github.com/globaltrusts/client-card/internal/api"
	"github.com/globaltrusts/client-card/internal/card"
	"github.com/globaltrusts/client-card/internal/storage"
)

// setupAPIServer 创建测试用 API 服务器（内存 SQLite + 空 Manager）。
func setupAPIServer(t *testing.T) (*httptest.Server, func()) {
	t.Helper()

	f, err := os.CreateTemp("", "api-test-*.db")
	if err != nil {
		t.Fatal(err)
	}
	f.Close()

	db, err := storage.Open(f.Name())
	if err != nil {
		os.Remove(f.Name())
		t.Fatal(err)
	}

	mgr := card.NewManager()
	cfg := &config.APIConfig{Host: "127.0.0.1", Port: 0}
	srv := api.NewServer(cfg, mgr, db)

	ts := httptest.NewServer(srv.Handler())
	return ts, func() {
		ts.Close()
		db.Close()
		os.Remove(f.Name())
	}
}

// apiReq 发送 JSON 请求并返回响应。
func apiReq(t *testing.T, ts *httptest.Server, method, path string, body interface{}) *http.Response {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			t.Fatalf("序列化请求体失败: %v", err)
		}
	}
	req, err := http.NewRequestWithContext(context.Background(), method, ts.URL+path, &buf)
	if err != nil {
		t.Fatalf("创建请求失败: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("发送请求失败: %v", err)
	}
	return resp
}

// decodeResp 解码 JSON 响应体。
func decodeResp(t *testing.T, resp *http.Response, out interface{}) {
	t.Helper()
	defer resp.Body.Close()
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		t.Fatalf("解码响应失败: %v", err)
	}
}

// ---- 健康检查 ----

func TestAPIHealth(t *testing.T) {
	ts, cleanup := setupAPIServer(t)
	defer cleanup()

	resp := apiReq(t, ts, http.MethodGet, "/api/health", nil)
	if resp.StatusCode != http.StatusOK {
		t.Errorf("健康检查状态码错误: got %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var result map[string]interface{}
	decodeResp(t, resp, &result)
	if result["data"] == nil {
		t.Error("健康检查响应缺少 data 字段")
	}
}

// ---- 用户 CRUD ----

func TestAPIUserCRUD(t *testing.T) {
	ts, cleanup := setupAPIServer(t)
	defer cleanup()

	// 创建用户
	createResp := apiReq(t, ts, http.MethodPost, "/api/users", map[string]interface{}{
		"user_type":    "local",
		"display_name": "API 测试用户",
		"email":        "api@test.com",
		"password":     "testpass123",
	})
	if createResp.StatusCode != http.StatusCreated {
		t.Fatalf("创建用户状态码错误: got %d, want %d", createResp.StatusCode, http.StatusCreated)
	}

	var createResult struct {
		Code int                    `json:"code"`
		Data map[string]interface{} `json:"data"`
	}
	decodeResp(t, createResp, &createResult)
	userUUID, _ := createResult.Data["uuid"].(string)
	if userUUID == "" {
		t.Fatal("创建用户响应缺少 uuid")
	}

	// 查询用户
	getResp := apiReq(t, ts, http.MethodGet, "/api/users/"+userUUID, nil)
	if getResp.StatusCode != http.StatusOK {
		t.Errorf("查询用户状态码错误: got %d, want %d", getResp.StatusCode, http.StatusOK)
	}
	getResp.Body.Close()

	// 列出用户
	listResp := apiReq(t, ts, http.MethodGet, "/api/users", nil)
	if listResp.StatusCode != http.StatusOK {
		t.Errorf("列出用户状态码错误: got %d, want %d", listResp.StatusCode, http.StatusOK)
	}
	var listResult struct {
		Data []interface{} `json:"data"`
	}
	decodeResp(t, listResp, &listResult)
	if len(listResult.Data) != 1 {
		t.Errorf("用户数量错误: got %d, want 1", len(listResult.Data))
	}

	// 更新用户
	updateResp := apiReq(t, ts, http.MethodPut, "/api/users/"+userUUID, map[string]interface{}{
		"display_name": "更新后的名称",
	})
	if updateResp.StatusCode != http.StatusOK {
		t.Errorf("更新用户状态码错误: got %d, want %d", updateResp.StatusCode, http.StatusOK)
	}
	updateResp.Body.Close()

	// 删除用户
	deleteResp := apiReq(t, ts, http.MethodDelete, "/api/users/"+userUUID, nil)
	if deleteResp.StatusCode != http.StatusOK {
		t.Errorf("删除用户状态码错误: got %d, want %d", deleteResp.StatusCode, http.StatusOK)
	}
	deleteResp.Body.Close()

	// 查询已删除用户应返回 404
	notFoundResp := apiReq(t, ts, http.MethodGet, "/api/users/"+userUUID, nil)
	if notFoundResp.StatusCode != http.StatusNotFound {
		t.Errorf("查询已删除用户状态码错误: got %d, want %d", notFoundResp.StatusCode, http.StatusNotFound)
	}
	notFoundResp.Body.Close()
}

// ---- 卡片 CRUD ----

func TestAPICardCRUD(t *testing.T) {
	ts, cleanup := setupAPIServer(t)
	defer cleanup()

	// 先创建用户
	createUserResp := apiReq(t, ts, http.MethodPost, "/api/users", map[string]interface{}{
		"user_type":    "local",
		"display_name": "卡片测试用户",
		"email":        "card-api@test.com",
		"password":     "cardpass123",
	})
	var userResult struct {
		Data map[string]interface{} `json:"data"`
	}
	decodeResp(t, createUserResp, &userResult)
	userUUID, _ := userResult.Data["uuid"].(string)

	// 创建卡片
	createCardResp := apiReq(t, ts, http.MethodPost, "/api/cards", map[string]interface{}{
		"slot_type":     "local",
		"card_name":     "API 测试卡片",
		"user_uuid":     userUUID,
		"user_password": "cardpass123",
		"remark":        "测试",
	})
	if createCardResp.StatusCode != http.StatusCreated {
		t.Fatalf("创建卡片状态码错误: got %d, want %d", createCardResp.StatusCode, http.StatusCreated)
	}

	var cardResult struct {
		Data map[string]interface{} `json:"data"`
	}
	decodeResp(t, createCardResp, &cardResult)
	cardUUID, _ := cardResult.Data["uuid"].(string)
	if cardUUID == "" {
		t.Fatal("创建卡片响应缺少 uuid")
	}

	// 查询卡片
	getCardResp := apiReq(t, ts, http.MethodGet, "/api/cards/"+cardUUID, nil)
	if getCardResp.StatusCode != http.StatusOK {
		t.Errorf("查询卡片状态码错误: got %d, want %d", getCardResp.StatusCode, http.StatusOK)
	}
	getCardResp.Body.Close()

	// 列出卡片（按用户过滤）
	listCardResp := apiReq(t, ts, http.MethodGet, "/api/cards?user_uuid="+userUUID, nil)
	if listCardResp.StatusCode != http.StatusOK {
		t.Errorf("列出卡片状态码错误: got %d, want %d", listCardResp.StatusCode, http.StatusOK)
	}
	var listCardResult struct {
		Data []interface{} `json:"data"`
	}
	decodeResp(t, listCardResp, &listCardResult)
	if len(listCardResult.Data) != 1 {
		t.Errorf("卡片数量错误: got %d, want 1", len(listCardResult.Data))
	}

	// 更新卡片
	updateCardResp := apiReq(t, ts, http.MethodPut, "/api/cards/"+cardUUID, map[string]interface{}{
		"card_name": "更新后的卡片名",
	})
	if updateCardResp.StatusCode != http.StatusOK {
		t.Errorf("更新卡片状态码错误: got %d, want %d", updateCardResp.StatusCode, http.StatusOK)
	}
	updateCardResp.Body.Close()

	// 删除卡片
	deleteCardResp := apiReq(t, ts, http.MethodDelete, "/api/cards/"+cardUUID, nil)
	if deleteCardResp.StatusCode != http.StatusOK {
		t.Errorf("删除卡片状态码错误: got %d, want %d", deleteCardResp.StatusCode, http.StatusOK)
	}
	deleteCardResp.Body.Close()
}

// ---- 日志查询 ----

func TestAPILogs(t *testing.T) {
	ts, cleanup := setupAPIServer(t)
	defer cleanup()

	resp := apiReq(t, ts, http.MethodGet, "/api/logs", nil)
	if resp.StatusCode != http.StatusOK {
		t.Errorf("日志查询状态码错误: got %d, want %d", resp.StatusCode, http.StatusOK)
	}
	resp.Body.Close()
}

// ---- Slot 状态 ----

func TestAPISlots(t *testing.T) {
	ts, cleanup := setupAPIServer(t)
	defer cleanup()

	resp := apiReq(t, ts, http.MethodGet, "/api/slots", nil)
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Slot 状态查询状态码错误: got %d, want %d", resp.StatusCode, http.StatusOK)
	}
	resp.Body.Close()
}

// ---- 参数校验 ----

func TestAPIValidation(t *testing.T) {
	ts, cleanup := setupAPIServer(t)
	defer cleanup()

	// 创建用户缺少必填字段
	resp := apiReq(t, ts, http.MethodPost, "/api/users", map[string]interface{}{
		"email": "no-name@test.com",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("缺少 display_name 应返回 400: got %d", resp.StatusCode)
	}
	resp.Body.Close()

	// 查询不存在的用户
	resp2 := apiReq(t, ts, http.MethodGet, "/api/users/nonexistent-uuid", nil)
	if resp2.StatusCode != http.StatusNotFound {
		t.Errorf("查询不存在用户应返回 404: got %d", resp2.StatusCode)
	}
	resp2.Body.Close()

	// 创建卡片密码错误
	createUserResp := apiReq(t, ts, http.MethodPost, "/api/users", map[string]interface{}{
		"user_type":    "local",
		"display_name": "验证测试用户",
		"email":        "valid@test.com",
		"password":     "validpass",
	})
	var userResult struct {
		Data map[string]interface{} `json:"data"`
	}
	decodeResp(t, createUserResp, &userResult)
	userUUID, _ := userResult.Data["uuid"].(string)

	resp3 := apiReq(t, ts, http.MethodPost, "/api/cards", map[string]interface{}{
		"card_name":     "测试卡",
		"user_uuid":     userUUID,
		"user_password": "wrongpassword",
	})
	if resp3.StatusCode != http.StatusUnauthorized {
		t.Errorf("密码错误应返回 401: got %d", resp3.StatusCode)
	}
	resp3.Body.Close()
}
