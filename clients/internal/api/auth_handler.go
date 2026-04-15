package api

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"sync"
	"time"

	"github.com/globaltrusts/client-card/internal/storage"
)

// ---- Session 管理（内存存储）----

// session 表示一个登录会话。
type session struct {
	UserUUID  string
	Username  string
	Role      string
	ExpiresAt time.Time
}

var (
	sessionMu sync.RWMutex
	sessions  = make(map[string]*session) // token -> session
)

// newSessionToken 生成随机 Token 并存储 session。
func newSessionToken(u *storage.User) string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	token := hex.EncodeToString(b)

	sessionMu.Lock()
	sessions[token] = &session{
		UserUUID:  u.UUID,
		Username:  u.Username,
		Role:      u.Role,
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	sessionMu.Unlock()
	return token
}

// getSession 根据 Token 获取 session（自动过期清理）。
func getSession(token string) *session {
	sessionMu.RLock()
	s, ok := sessions[token]
	sessionMu.RUnlock()
	if !ok || time.Now().After(s.ExpiresAt) {
		sessionMu.Lock()
		delete(sessions, token)
		sessionMu.Unlock()
		return nil
	}
	return s
}

// deleteSession 删除 session（登出）。
func deleteSession(token string) {
	sessionMu.Lock()
	delete(sessions, token)
	sessionMu.Unlock()
}

// authTokenResponse 是登录/注册成功的响应体。
type authTokenResponse struct {
	Token     string `json:"token"`
	UserUUID  string `json:"user_uuid"`
	Username  string `json:"username"`
	Role      string `json:"role"`
	ExpiresAt string `json:"expires_at"`
}

// ---- 登录 POST /api/auth/login ----

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "请求格式错误: "+err.Error())
		return
	}
	if req.Username == "" || req.Password == "" {
		writeError(w, http.StatusBadRequest, "用户名和密码不能为空")
		return
	}

	user, err := s.userRepo.GetByUsername(r.Context(), req.Username)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "查询用户失败")
		return
	}
	if user == nil || !verifyPassword(req.Password, user.PasswordHash) {
		writeError(w, http.StatusUnauthorized, "用户名或密码错误")
		return
	}
	if !user.Enabled {
		writeError(w, http.StatusForbidden, "账号已被禁用")
		return
	}

	token := newSessionToken(user)
	writeOK(w, authTokenResponse{
		Token:     token,
		UserUUID:  user.UUID,
		Username:  user.Username,
		Role:      user.Role,
		ExpiresAt: time.Now().Add(24 * time.Hour).Format(time.RFC3339),
	})
}

// ---- 注册 POST /api/auth/register ----

func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username    string `json:"username"`
		Password    string `json:"password"`
		Email       string `json:"email"`
		DisplayName string `json:"display_name"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "请求格式错误: "+err.Error())
		return
	}
	if req.Username == "" || req.Password == "" || req.DisplayName == "" {
		writeError(w, http.StatusBadRequest, "username、password、display_name 不能为空")
		return
	}
	if len(req.Username) < 3 {
		writeError(w, http.StatusBadRequest, "用户名至少3位")
		return
	}
	if len(req.Password) < 8 {
		writeError(w, http.StatusBadRequest, "密码至少8位")
		return
	}

	// 检查用户名是否已存在
	existing, err := s.userRepo.GetByUsername(r.Context(), req.Username)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "查询用户失败")
		return
	}
	if existing != nil {
		writeError(w, http.StatusConflict, "用户名已存在")
		return
	}

	hash, err := hashPassword(req.Password)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "密码处理失败")
		return
	}

	user := &storage.User{
		UserType:     storage.UserTypeLocal,
		Role:         "user",
		Username:     req.Username,
		DisplayName:  req.DisplayName,
		Email:        req.Email,
		Enabled:      true,
		PasswordHash: hash,
	}
	if err := s.userRepo.Create(r.Context(), user); err != nil {
		writeError(w, http.StatusInternalServerError, "创建用户失败: "+err.Error())
		return
	}

	token := newSessionToken(user)
	writeJSON(w, http.StatusCreated, Response{Code: 0, Message: "ok", Data: authTokenResponse{
		Token:     token,
		UserUUID:  user.UUID,
		Username:  user.Username,
		Role:      user.Role,
		ExpiresAt: time.Now().Add(24 * time.Hour).Format(time.RFC3339),
	}})
}

// ---- 刷新 Token POST /api/auth/refresh ----

func (s *Server) handleRefreshToken(w http.ResponseWriter, r *http.Request) {
	token := extractBearerToken(r)
	if token == "" {
		writeError(w, http.StatusUnauthorized, "缺少 Token")
		return
	}
	sess := getSession(token)
	if sess == nil {
		writeError(w, http.StatusUnauthorized, "Token 无效或已过期")
		return
	}

	user, err := s.userRepo.GetByUUID(r.Context(), sess.UserUUID)
	if err != nil || user == nil {
		writeError(w, http.StatusUnauthorized, "用户不存在")
		return
	}

	// 删除旧 Token，生成新 Token
	deleteSession(token)
	newToken := newSessionToken(user)
	writeOK(w, authTokenResponse{
		Token:     newToken,
		UserUUID:  user.UUID,
		Username:  user.Username,
		Role:      user.Role,
		ExpiresAt: time.Now().Add(24 * time.Hour).Format(time.RFC3339),
	})
}

// ---- 登出 DELETE /api/auth/logout ----

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	token := extractBearerToken(r)
	if token != "" {
		deleteSession(token)
	}
	writeOK(w, nil)
}

// ---- 修改密码 PUT /api/auth/password ----

func (s *Server) handleChangePassword(w http.ResponseWriter, r *http.Request) {
	token := extractBearerToken(r)
	sess := getSession(token)
	if sess == nil {
		writeError(w, http.StatusUnauthorized, "未登录")
		return
	}

	var req struct {
		OldPassword string `json:"old_password"`
		NewPassword string `json:"new_password"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "请求格式错误: "+err.Error())
		return
	}
	if len(req.NewPassword) < 8 {
		writeError(w, http.StatusBadRequest, "新密码至少8位")
		return
	}

	user, err := s.userRepo.GetByUUID(r.Context(), sess.UserUUID)
	if err != nil || user == nil {
		writeError(w, http.StatusUnauthorized, "用户不存在")
		return
	}
	if !verifyPassword(req.OldPassword, user.PasswordHash) {
		writeError(w, http.StatusUnauthorized, "原密码错误")
		return
	}

	hash, err := hashPassword(req.NewPassword)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "密码处理失败")
		return
	}
	user.PasswordHash = hash
	if err := s.userRepo.Update(r.Context(), user); err != nil {
		writeError(w, http.StatusInternalServerError, "更新密码失败")
		return
	}
	writeOK(w, nil)
}

// ---- 获取当前用户 GET /api/users/me ----

func (s *Server) handleGetMe(w http.ResponseWriter, r *http.Request) {
	token := extractBearerToken(r)
	sess := getSession(token)
	if sess == nil {
		writeError(w, http.StatusUnauthorized, "未登录")
		return
	}

	user, err := s.userRepo.GetByUUID(r.Context(), sess.UserUUID)
	if err != nil || user == nil {
		writeError(w, http.StatusUnauthorized, "用户不存在")
		return
	}
	writeOK(w, user)
}

// ---- 更新当前用户 PUT /api/users/me ----

func (s *Server) handleUpdateMe(w http.ResponseWriter, r *http.Request) {
	token := extractBearerToken(r)
	sess := getSession(token)
	if sess == nil {
		writeError(w, http.StatusUnauthorized, "未登录")
		return
	}

	user, err := s.userRepo.GetByUUID(r.Context(), sess.UserUUID)
	if err != nil || user == nil {
		writeError(w, http.StatusUnauthorized, "用户不存在")
		return
	}

	var req struct {
		DisplayName string `json:"display_name"`
		Email       string `json:"email"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "请求格式错误: "+err.Error())
		return
	}
	if req.DisplayName != "" {
		user.DisplayName = req.DisplayName
	}
	if req.Email != "" {
		user.Email = req.Email
	}
	if err := s.userRepo.Update(r.Context(), user); err != nil {
		writeError(w, http.StatusInternalServerError, "更新用户失败")
		return
	}
	writeOK(w, user)
}

// extractBearerToken 从请求头提取 Bearer Token。
func extractBearerToken(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	if len(auth) > 7 && auth[:7] == "Bearer " {
		return auth[7:]
	}
	return ""
}
