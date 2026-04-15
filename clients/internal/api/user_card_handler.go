package api

import (
	"net/http"
	"time"

	"github.com/globaltrusts/client-card/internal/storage"
)

// ---- 用户管理 Handler ----

// handleListUsers GET /api/users
func (s *Server) handleListUsers(w http.ResponseWriter, r *http.Request) {
	users, err := s.userRepo.List(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "查询用户列表失败: "+err.Error())
		return
	}
	writeOK(w, users)
}

// handleCreateUser POST /api/users
func (s *Server) handleCreateUser(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserType    string `json:"user_type"`
		DisplayName string `json:"display_name"`
		Email       string `json:"email"`
		Password    string `json:"password"`
		CloudURL    string `json:"cloud_url"`
		Role        string `json:"role"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "请求格式错误: "+err.Error())
		return
	}

	if req.DisplayName == "" {
		writeError(w, http.StatusBadRequest, "display_name 不能为空")
		return
	}

	user := &storage.User{
		UserType:    storage.UserType(req.UserType),
		DisplayName: req.DisplayName,
		Email:       req.Email,
		CloudURL:    req.CloudURL,
		Role:        req.Role,
		Enabled:     true,
	}

	if user.UserType == "" {
		user.UserType = storage.UserTypeLocal
	}

	// 本地用户需要密码
	if user.UserType == storage.UserTypeLocal && req.Password != "" {
		hash, err := hashPassword(req.Password)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "密码哈希失败")
			return
		}
		user.PasswordHash = hash
	}

	if err := s.userRepo.Create(r.Context(), user); err != nil {
		writeError(w, http.StatusInternalServerError, "创建用户失败: "+err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, Response{Code: 0, Message: "ok", Data: user})
}

// handleGetUser GET /api/users/{uuid}
func (s *Server) handleGetUser(w http.ResponseWriter, r *http.Request) {
	userUUID := r.PathValue("uuid")
	user, err := s.userRepo.GetByUUID(r.Context(), userUUID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if user == nil {
		writeError(w, http.StatusNotFound, "用户不存在")
		return
	}
	writeOK(w, user)
}

// handleUpdateUser PUT /api/users/{uuid}
func (s *Server) handleUpdateUser(w http.ResponseWriter, r *http.Request) {
	userUUID := r.PathValue("uuid")

	user, err := s.userRepo.GetByUUID(r.Context(), userUUID)
	if err != nil || user == nil {
		writeError(w, http.StatusNotFound, "用户不存在")
		return
	}

	var req struct {
		DisplayName string `json:"display_name"`
		Email       string `json:"email"`
		Enabled     *bool  `json:"enabled"`
		CloudURL    string `json:"cloud_url"`
		Password    string `json:"password"`
		Role        string `json:"role"`
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
	if req.Enabled != nil {
		user.Enabled = *req.Enabled
	}
	if req.CloudURL != "" {
		user.CloudURL = req.CloudURL
	}
	if req.Role != "" {
		user.Role = req.Role
	}
	if req.Password != "" {
		hash, err := hashPassword(req.Password)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "密码哈希失败")
			return
		}
		user.PasswordHash = hash
	}

	if err := s.userRepo.Update(r.Context(), user); err != nil {
		writeError(w, http.StatusInternalServerError, "更新用户失败: "+err.Error())
		return
	}
	writeOK(w, user)
}

// handleDeleteUser DELETE /api/users/{uuid}
func (s *Server) handleDeleteUser(w http.ResponseWriter, r *http.Request) {
	userUUID := r.PathValue("uuid")
	if err := s.userRepo.Delete(r.Context(), userUUID); err != nil {
		// 区分"不存在"和"内部错误"
		if isNotFoundErr(err) {
			writeError(w, http.StatusNotFound, "用户不存在")
			return
		}
		writeError(w, http.StatusInternalServerError, "删除用户失败: "+err.Error())
		return
	}
	writeOK(w, nil)
}

// ---- 卡片管理 Handler ----

// handleListCards GET /api/cards
func (s *Server) handleListCards(w http.ResponseWriter, r *http.Request) {
	userUUID := r.URL.Query().Get("user_uuid")

	var (
		cards []*storage.Card
		err   error
	)
	if userUUID != "" {
		cards, err = s.cardRepo.ListByUser(r.Context(), userUUID)
	} else {
		cards, err = s.cardRepo.ListAll(r.Context())
	}

	if err != nil {
		writeError(w, http.StatusInternalServerError, "查询卡片列表失败: "+err.Error())
		return
	}
	writeOK(w, cards)
}

// handleCreateCard POST /api/cards
func (s *Server) handleCreateCard(w http.ResponseWriter, r *http.Request) {
	var req struct {
		SlotType     string `json:"slot_type"`
		CardName     string `json:"card_name"`
		UserUUID     string `json:"user_uuid"`
		UserPassword string `json:"user_password"` // 用于加密主密钥
		CardPassword string `json:"card_password"` // 可选
		ExpiresAt    string `json:"expires_at"`    // RFC3339 格式
		Remark       string `json:"remark"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "请求格式错误: "+err.Error())
		return
	}

	if req.CardName == "" || req.UserUUID == "" || req.UserPassword == "" {
		writeError(w, http.StatusBadRequest, "card_name、user_uuid、user_password 不能为空")
		return
	}

	// 验证用户存在
	user, err := s.userRepo.GetByUUID(r.Context(), req.UserUUID)
	if err != nil || user == nil {
		writeError(w, http.StatusBadRequest, "用户不存在")
		return
	}

	// 验证用户密码
	if !verifyPassword(req.UserPassword, user.PasswordHash) {
		writeError(w, http.StatusUnauthorized, "用户密码错误")
		return
	}

	card, err := createLocalCard(r.Context(), s.cardRepo, req.UserUUID, req.CardName, req.UserPassword, req.CardPassword, req.Remark)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "创建卡片失败: "+err.Error())
		return
	}

	// 设置过期时间
	if req.ExpiresAt != "" {
		t, err := time.Parse(time.RFC3339, req.ExpiresAt)
		if err == nil {
			card.ExpiresAt = &t
			s.cardRepo.Update(r.Context(), card)
		}
	}

	writeJSON(w, http.StatusCreated, Response{Code: 0, Message: "ok", Data: card})
}

// handleGetCard GET /api/cards/{uuid}
func (s *Server) handleGetCard(w http.ResponseWriter, r *http.Request) {
	cardUUID := r.PathValue("uuid")
	card, err := s.cardRepo.GetByUUID(r.Context(), cardUUID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if card == nil {
		writeError(w, http.StatusNotFound, "卡片不存在")
		return
	}
	writeOK(w, card)
}

// handleUpdateCard PUT /api/cards/{uuid}
func (s *Server) handleUpdateCard(w http.ResponseWriter, r *http.Request) {
	cardUUID := r.PathValue("uuid")
	card, err := s.cardRepo.GetByUUID(r.Context(), cardUUID)
	if err != nil || card == nil {
		writeError(w, http.StatusNotFound, "卡片不存在")
		return
	}

	var req struct {
		CardName  string `json:"card_name"`
		ExpiresAt string `json:"expires_at"`
		Remark    string `json:"remark"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "请求格式错误: "+err.Error())
		return
	}

	if req.CardName != "" {
		card.CardName = req.CardName
	}
	if req.Remark != "" {
		card.Remark = req.Remark
	}
	if req.ExpiresAt != "" {
		t, err := time.Parse(time.RFC3339, req.ExpiresAt)
		if err == nil {
			card.ExpiresAt = &t
		}
	}

	if err := s.cardRepo.Update(r.Context(), card); err != nil {
		writeError(w, http.StatusInternalServerError, "更新卡片失败: "+err.Error())
		return
	}
	writeOK(w, card)
}

// handleDeleteCard DELETE /api/cards/{uuid}
func (s *Server) handleDeleteCard(w http.ResponseWriter, r *http.Request) {
	cardUUID := r.PathValue("uuid")
	if err := s.cardRepo.Delete(r.Context(), cardUUID); err != nil {
		if isNotFoundErr(err) {
			writeError(w, http.StatusNotFound, "卡片不存在")
			return
		}
		writeError(w, http.StatusInternalServerError, "删除卡片失败: "+err.Error())
		return
	}
	writeOK(w, nil)
}
