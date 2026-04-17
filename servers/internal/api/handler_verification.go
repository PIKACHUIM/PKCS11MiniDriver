package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/globaltrusts/server-card/internal/storage"
)

// ---- 主体信息处理器 ----

func (s *Server) handleListSubjectInfos(w http.ResponseWriter, r *http.Request) {
	claims := claimsFromCtx(r.Context())
	userUUID := claims.UserUUID
	if claims.Role == "admin" {
		if q := r.URL.Query().Get("user_uuid"); q != "" {
			userUUID = q
		}
	}
	infos, err := s.verifySvc.ListSubjectInfos(r.Context(), userUUID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"subject_infos": infos, "total": len(infos)})
}

func (s *Server) handleCreateSubjectInfo(w http.ResponseWriter, r *http.Request) {
	claims := claimsFromCtx(r.Context())
	var info storage.SubjectInfo
	if err := json.NewDecoder(r.Body).Decode(&info); err != nil {
		writeError(w, http.StatusBadRequest, "请求格式错误")
		return
	}
	info.UserUUID = claims.UserUUID
	if err := s.verifySvc.CreateSubjectInfo(r.Context(), &info); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, info)
}

func (s *Server) handleDeleteSubjectInfo(w http.ResponseWriter, r *http.Request) {
	claims := claimsFromCtx(r.Context())
	infoUUID := r.PathValue("uuid")

	// 验证归属（只有所有者或管理员可以删除）
	infos, err := s.verifySvc.ListSubjectInfos(r.Context(), claims.UserUUID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	found := false
	for _, info := range infos {
		if info.UUID == infoUUID {
			found = true
			break
		}
	}
	if !found && claims.Role != "admin" {
		writeError(w, http.StatusForbidden, "无权删除此主体信息")
		return
	}

	if err := s.verifySvc.DeleteSubjectInfo(r.Context(), infoUUID); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "主体信息已删除"})
}

func (s *Server) handleApproveSubjectInfo(w http.ResponseWriter, r *http.Request) {
	claims := claimsFromCtx(r.Context())
	infoUUID := r.PathValue("uuid")
	if err := s.verifySvc.ApproveSubjectInfo(r.Context(), infoUUID, claims.UserUUID); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "主体信息已审核通过"})
}

func (s *Server) handleRejectSubjectInfo(w http.ResponseWriter, r *http.Request) {
	claims := claimsFromCtx(r.Context())
	infoUUID := r.PathValue("uuid")
	if err := s.verifySvc.RejectSubjectInfo(r.Context(), infoUUID, claims.UserUUID); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "主体信息已拒绝"})
}

// ---- 扩展信息验证处理器 ----

func (s *Server) handleListExtensionInfos(w http.ResponseWriter, r *http.Request) {
	claims := claimsFromCtx(r.Context())
	infos, err := s.verifySvc.ListExtensionInfos(r.Context(), claims.UserUUID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"extension_infos": infos, "total": len(infos)})
}

func (s *Server) handleCreateExtensionInfo(w http.ResponseWriter, r *http.Request) {
	claims := claimsFromCtx(r.Context())
	var info storage.ExtensionInfo
	if err := json.NewDecoder(r.Body).Decode(&info); err != nil {
		writeError(w, http.StatusBadRequest, "请求格式错误")
		return
	}
	info.UserUUID = claims.UserUUID
	if err := s.verifySvc.CreateExtensionInfo(r.Context(), &info); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"extension_info": info,
		"verify_token":   info.VerifyToken,
		"message":        fmt.Sprintf("请配置验证记录，token: %s", info.VerifyToken),
	})
}

func (s *Server) handleVerifyDNS(w http.ResponseWriter, r *http.Request) {
	infoUUID := r.PathValue("uuid")
	if err := s.verifySvc.VerifyDNSTXT(r.Context(), infoUUID); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "DNS 验证通过"})
}

func (s *Server) handleVerifyEmail(w http.ResponseWriter, r *http.Request) {
	infoUUID := r.PathValue("uuid")
	var req struct {
		Code string `json:"code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "请求格式错误")
		return
	}
	if err := s.verifySvc.VerifyEmailCode(r.Context(), infoUUID, req.Code); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "邮箱验证通过"})
}

// handleVerifyHTTP 通过 HTTP 文件验证域名所有权。
func (s *Server) handleVerifyHTTP(w http.ResponseWriter, r *http.Request) {
	infoUUID := r.PathValue("uuid")

	// 获取扩展信息
	info, err := s.verifySvc.GetExtensionInfo(r.Context(), infoUUID)
	if err != nil {
		writeError(w, http.StatusNotFound, "扩展信息不存在")
		return
	}

	if info.VerifyStatus == "verified" {
		writeJSON(w, http.StatusOK, map[string]string{"message": "已验证通过"})
		return
	}

	// 构建验证 URL
	verifyURL := fmt.Sprintf("http://%s/.well-known/pki-validation/%s", info.Value, info.VerifyToken)

	// 发起 HTTP GET 请求验证文件内容
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(verifyURL)
	if err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("HTTP 验证请求失败: %v", err))
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("HTTP 验证文件不存在（状态码: %d）", resp.StatusCode))
		return
	}

	// 读取文件内容（限制 1KB）
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1024))
	if err != nil {
		writeError(w, http.StatusBadRequest, "读取验证文件失败")
		return
	}

	// 验证文件内容是否包含 token
	content := string(body)
	if !containsToken(content, info.VerifyToken) {
		writeError(w, http.StatusBadRequest, "验证文件内容不匹配")
		return
	}

	// 标记验证通过
	if err := s.verifySvc.MarkVerified(r.Context(), infoUUID); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "HTTP 验证通过"})
}

// containsToken 检查内容是否包含 token。
func containsToken(content, token string) bool {
	return len(content) > 0 && len(token) > 0 &&
		(content == token || containsSubstring(content, token))
}

func containsSubstring(s, sub string) bool {
	if len(sub) > len(s) {
		return false
	}
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

func (s *Server) handleDeleteExtensionInfo(w http.ResponseWriter, r *http.Request) {
	infoUUID := r.PathValue("uuid")
	if err := s.verifySvc.DeleteExtensionInfo(r.Context(), infoUUID); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "扩展信息已删除"})
}
