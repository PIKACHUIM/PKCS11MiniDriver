package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/globaltrusts/server-card/internal/auth"
	"github.com/globaltrusts/server-card/internal/storage"
)

// ---- 审计日志处理器 ----

// handleListAuditLogs 分页查询审计日志（管理员）。
func (s *Server) handleListAuditLogs(w http.ResponseWriter, r *http.Request) {
	userUUID := r.URL.Query().Get("user_uuid")
	action := r.URL.Query().Get("action")
	resourceType := r.URL.Query().Get("resource_type")
	page, pageSize := parsePagination(r)

	logs, total, integrityBroken, err := s.auditLogRepo.List(r.Context(), userUUID, action, resourceType, page, pageSize)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if logs == nil {
		logs = []*storage.AuditLog{}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"items":            logs,
		"total":            total,
		"page":             page,
		"page_size":        pageSize,
		"integrity_broken": integrityBroken,
	})
}

// ---- 证书申请模板处理器 ----

// handleListCertApplyTemplates 查询证书申请模板列表。
func (s *Server) handleListCertApplyTemplates(w http.ResponseWriter, r *http.Request) {
	claims := claimsFromCtx(r.Context())
	onlyEnabled := true
	// 管理员可以查看所有模板（包括禁用的）
	if auth.IsAdmin(claims.Role) {
		onlyEnabled = false
	}

	rows, err := s.db.QueryContext(r.Context(),
		`SELECT uuid, name, issuance_tmpl_uuid, valid_days, ca_uuid, enabled, require_approval, allow_renewal, allowed_key_types, price_cents, description, created_at, updated_at
		 FROM cert_apply_templates WHERE (? = 0 OR enabled = 1) ORDER BY created_at DESC`,
		boolToIntLocal(onlyEnabled),
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()

	var templates []*storage.CertApplyTemplate
	for rows.Next() {
		t := &storage.CertApplyTemplate{}
		var enabled, requireApproval, allowRenewal int
		if err := rows.Scan(&t.UUID, &t.Name, &t.IssuanceTmplUUID, &t.ValidDays, &t.CAUUID,
			&enabled, &requireApproval, &allowRenewal, &t.AllowedKeyTypes, &t.PriceCents, &t.Description,
			&t.CreatedAt, &t.UpdatedAt); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		t.Enabled = enabled == 1
		t.RequireApproval = requireApproval == 1
		t.AllowRenewal = allowRenewal == 1
		templates = append(templates, t)
	}
	if templates == nil {
		templates = []*storage.CertApplyTemplate{}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"items": templates, "total": len(templates)})
}

// handleGetCertApplyTemplate 查询单个证书申请模板。
func (s *Server) handleGetCertApplyTemplate(w http.ResponseWriter, r *http.Request) {
	tmplUUID := r.PathValue("uuid")
	t := &storage.CertApplyTemplate{}
	var enabled, requireApproval, allowRenewal int
	err := s.db.QueryRowContext(r.Context(),
		`SELECT uuid, name, issuance_tmpl_uuid, valid_days, ca_uuid, enabled, require_approval, allow_renewal, allowed_key_types, price_cents, description, created_at, updated_at
		 FROM cert_apply_templates WHERE uuid = ?`, tmplUUID,
	).Scan(&t.UUID, &t.Name, &t.IssuanceTmplUUID, &t.ValidDays, &t.CAUUID,
		&enabled, &requireApproval, &allowRenewal, &t.AllowedKeyTypes, &t.PriceCents, &t.Description,
		&t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		writeError(w, http.StatusNotFound, "证书申请模板不存在")
		return
	}
	t.Enabled = enabled == 1
	t.RequireApproval = requireApproval == 1
	t.AllowRenewal = allowRenewal == 1
	writeJSON(w, http.StatusOK, t)
}

// handleCreateCertApplyTemplate 创建证书申请模板（管理员）。
func (s *Server) handleCreateCertApplyTemplate(w http.ResponseWriter, r *http.Request) {
	var t storage.CertApplyTemplate
	if err := json.NewDecoder(r.Body).Decode(&t); err != nil {
		writeError(w, http.StatusBadRequest, "请求格式错误")
		return
	}
	if t.Name == "" {
		writeError(w, http.StatusBadRequest, "模板名称不能为空")
		return
	}
	if t.AllowedKeyTypes == "" {
		t.AllowedKeyTypes = `["ec256","rsa2048"]`
	}

	t.UUID = uuid.New().String()
	t.CreatedAt = time.Now()
	t.UpdatedAt = time.Now()

	_, err := s.db.ExecContext(r.Context(),
		`INSERT INTO cert_apply_templates (uuid, name, issuance_tmpl_uuid, valid_days, ca_uuid, enabled, require_approval, allow_renewal, allowed_key_types, price_cents, description, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		t.UUID, t.Name, t.IssuanceTmplUUID, t.ValidDays, t.CAUUID,
		storage.BoolToInt(t.Enabled), storage.BoolToInt(t.RequireApproval), storage.BoolToInt(t.AllowRenewal),
		t.AllowedKeyTypes, t.PriceCents, t.Description, t.CreatedAt, t.UpdatedAt,
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// 写入审计日志
	claims := claimsFromCtx(r.Context())
	s.auditLogRepo.Create(r.Context(), &storage.AuditLog{ //nolint:errcheck
		UserUUID:     claims.UserUUID,
		Action:       "create_cert_apply_template",
		ResourceType: "cert_apply_template",
		ResourceUUID: t.UUID,
		Detail:       `{"name":"` + t.Name + `"}`,
		IPAddress:    r.RemoteAddr,
	})

	writeJSON(w, http.StatusCreated, t)
}

// handleUpdateCertApplyTemplate 更新证书申请模板（管理员）。
func (s *Server) handleUpdateCertApplyTemplate(w http.ResponseWriter, r *http.Request) {
	tmplUUID := r.PathValue("uuid")
	var t storage.CertApplyTemplate
	if err := json.NewDecoder(r.Body).Decode(&t); err != nil {
		writeError(w, http.StatusBadRequest, "请求格式错误")
		return
	}

	_, err := s.db.ExecContext(r.Context(),
		`UPDATE cert_apply_templates SET name=?, issuance_tmpl_uuid=?, valid_days=?, ca_uuid=?, enabled=?, require_approval=?, allow_renewal=?, allowed_key_types=?, price_cents=?, description=?, updated_at=?
		 WHERE uuid=?`,
		t.Name, t.IssuanceTmplUUID, t.ValidDays, t.CAUUID,
		storage.BoolToInt(t.Enabled), storage.BoolToInt(t.RequireApproval), storage.BoolToInt(t.AllowRenewal),
		t.AllowedKeyTypes, t.PriceCents, t.Description, time.Now(), tmplUUID,
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "证书申请模板已更新"})
}

// handleDeleteCertApplyTemplate 删除证书申请模板（管理员）。
func (s *Server) handleDeleteCertApplyTemplate(w http.ResponseWriter, r *http.Request) {
	tmplUUID := r.PathValue("uuid")
	_, err := s.db.ExecContext(r.Context(), `DELETE FROM cert_apply_templates WHERE uuid = ?`, tmplUUID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "证书申请模板已删除"})
}
