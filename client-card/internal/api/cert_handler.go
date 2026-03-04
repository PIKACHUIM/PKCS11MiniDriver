package api

import (
	"encoding/base64"
	"net/http"
	"strconv"

	"github.com/globaltrusts/client-card/internal/card/local"
	"github.com/globaltrusts/client-card/internal/storage"
)

// ---- 证书管理 Handler ----

// handleListCerts GET /api/cards/{card_uuid}/certs
func (s *Server) handleListCerts(w http.ResponseWriter, r *http.Request) {
	cardUUID := r.PathValue("card_uuid")
	certs, err := s.certRepo.ListByCard(r.Context(), cardUUID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "查询证书列表失败: "+err.Error())
		return
	}
	writeOK(w, certs)
}

// handleCreateCert POST /api/cards/{card_uuid}/certs
// 用于导入已有证书（公钥/X.509 DER，base64 编码）
func (s *Server) handleCreateCert(w http.ResponseWriter, r *http.Request) {
	cardUUID := r.PathValue("card_uuid")

	var req struct {
		CertType    string `json:"cert_type"`
		KeyType     string `json:"key_type"`
		CertContent string `json:"cert_content"` // base64 DER
		Remark      string `json:"remark"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "请求格式错误: "+err.Error())
		return
	}

	certDER, err := base64.StdEncoding.DecodeString(req.CertContent)
	if err != nil {
		writeError(w, http.StatusBadRequest, "cert_content 必须是 base64 编码的 DER 数据")
		return
	}

	certRepo := storage.NewCertRepo(s.db)
	km := local.NewKeyManager(certRepo, s.cardRepo)

	cert, err := km.ImportCertificate(r.Context(), cardUUID, certDER, req.Remark)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "导入证书失败: "+err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, Response{Code: 0, Message: "ok", Data: cert})
}

// handleGetCert GET /api/cards/{card_uuid}/certs/{uuid}
func (s *Server) handleGetCert(w http.ResponseWriter, r *http.Request) {
	certUUID := r.PathValue("uuid")
	cert, err := s.certRepo.GetByUUID(r.Context(), certUUID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if cert == nil {
		writeError(w, http.StatusNotFound, "证书不存在")
		return
	}
	writeOK(w, cert)
}

// handleDeleteCert DELETE /api/cards/{card_uuid}/certs/{uuid}
func (s *Server) handleDeleteCert(w http.ResponseWriter, r *http.Request) {
	certUUID := r.PathValue("uuid")
	if err := s.certRepo.Delete(r.Context(), certUUID); err != nil {
		writeError(w, http.StatusInternalServerError, "删除证书失败: "+err.Error())
		return
	}
	writeOK(w, nil)
}

// ---- 密钥生成 Handler ----

// handleKeyGen POST /api/cards/{card_uuid}/keygen
// 在指定卡片中生成密钥对，需要提供卡片密码解锁主密钥
func (s *Server) handleKeyGen(w http.ResponseWriter, r *http.Request) {
	cardUUID := r.PathValue("card_uuid")

	var req struct {
		CardPassword string `json:"card_password"` // 用于解锁主密钥
		CertType     string `json:"cert_type"`     // x509/ssh/gpg
		KeyType      string `json:"key_type"`      // rsa2048/rsa4096/ec256/ec384/ec521
		Remark       string `json:"remark"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "请求格式错误: "+err.Error())
		return
	}

	if req.CardPassword == "" || req.KeyType == "" {
		writeError(w, http.StatusBadRequest, "card_password 和 key_type 不能为空")
		return
	}

	// 获取卡片
	card, err := s.cardRepo.GetByUUID(r.Context(), cardUUID)
	if err != nil || card == nil {
		writeError(w, http.StatusNotFound, "卡片不存在")
		return
	}

	// 创建临时 Slot 解锁主密钥
	slot := local.New(0, card, s.certRepo)
	if err := slot.Login(r.Context(), 1, req.CardPassword); err != nil {
		writeError(w, http.StatusUnauthorized, "卡片密码错误")
		return
	}
	defer slot.Logout(r.Context())

	masterKey := slot.MasterKey()
	if masterKey == nil {
		writeError(w, http.StatusInternalServerError, "获取主密钥失败")
		return
	}

	certType := storage.CertType(req.CertType)
	if certType == "" {
		certType = storage.CertTypeX509
	}

	km := local.NewKeyManager(s.certRepo, s.cardRepo)
	result, err := km.GenerateKeyPair(r.Context(), local.KeyGenRequest{
		CardUUID: cardUUID,
		CertType: certType,
		KeyType:  req.KeyType,
		Remark:   req.Remark,
	}, masterKey)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "生成密钥对失败: "+err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, Response{
		Code:    0,
		Message: "ok",
		Data: map[string]string{
			"cert_uuid":      result.CertUUID,
			"public_key_b64": base64.StdEncoding.EncodeToString(result.PublicKeyDER),
		},
	})
}

// ---- 日志查询 Handler ----

// handleListLogs GET /api/logs?limit=20&offset=0
func (s *Server) handleListLogs(w http.ResponseWriter, r *http.Request) {
	limit := 20
	offset := 0

	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 100 {
			limit = n
		}
	}
	if v := r.URL.Query().Get("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			offset = n
		}
	}

	logs, err := s.logRepo.List(r.Context(), limit, offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "查询日志失败: "+err.Error())
		return
	}
	writeOK(w, logs)
}
