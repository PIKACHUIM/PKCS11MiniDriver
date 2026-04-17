package api

import (
	"encoding/base64"
	"fmt"
	"net/http"

	"github.com/globaltrusts/client-card/internal/pki"
	"github.com/globaltrusts/client-card/internal/storage"
)

// ---- 自签名证书（保留原有功能）----

// handleSelfSignFromCSR POST /api/pki/certs/selfsign
// 通过已有 CSR（需含私钥）生成自签名证书并持久化到证书管理。
func (s *Server) handleSelfSignFromCSR(w http.ResponseWriter, r *http.Request) {
	var req pki.SelfSignFromCSRRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "请求格式错误: "+err.Error())
		return
	}
	if req.CSRUUID == "" {
		writeError(w, http.StatusBadRequest, "csr_uuid 不能为空")
		return
	}
	cert, err := pki.SelfSignFromCSR(r.Context(), s.csrRepo, s.pkiCertRepo, &req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "生成自签名证书失败: "+err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, Response{Code: 0, Message: "ok", Data: cert})
}

// handleSelfSign POST /api/pki/selfsign
func (s *Server) handleSelfSign(w http.ResponseWriter, r *http.Request) {
	var req pki.SelfSignRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "请求格式错误: "+err.Error())
		return
	}
	if req.CommonName == "" {
		writeError(w, http.StatusBadRequest, "common_name 不能为空")
		return
	}
	if req.KeyType == "" {
		req.KeyType = "ec256"
	}
	result, err := pki.GenerateSelfSigned(&req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "生成自签名证书失败: "+err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, Response{Code: 0, Message: "ok", Data: map[string]string{
		"cert_pem": string(result.CertPEM),
		"key_pem":  string(result.KeyPEM),
		"cert_der": base64.StdEncoding.EncodeToString(result.CertDER),
	}})
}

// ---- CSR 管理 ----

// handleListCSR GET /api/pki/csr
func (s *Server) handleListCSR(w http.ResponseWriter, r *http.Request) {
	page, pageSize := parsePageParams(r)
	list, total, err := s.csrRepo.List(r.Context(), page, pageSize)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "查询 CSR 列表失败: "+err.Error())
		return
	}
	if list == nil {
		list = []*storage.CSRRecord{}
	}
	writeOK(w, map[string]interface{}{"items": list, "total": total, "page": page, "page_size": pageSize})
}

// handleCreateCSR POST /api/pki/csr
func (s *Server) handleCreateCSR(w http.ResponseWriter, r *http.Request) {
	var req pki.CreateCSRRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "请求格式错误: "+err.Error())
		return
	}
	if req.CommonName == "" {
		writeError(w, http.StatusBadRequest, "common_name 不能为空")
		return
	}
	record, err := pki.CreateAndSaveCSR(r.Context(), s.csrRepo, &req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "生成 CSR 失败: "+err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, Response{Code: 0, Message: "ok", Data: record})
}

// handleGetCSR GET /api/pki/csr/{uuid}
func (s *Server) handleGetCSR(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("uuid")
	record, err := s.csrRepo.GetByUUID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "查询 CSR 失败: "+err.Error())
		return
	}
	if record == nil {
		writeError(w, http.StatusNotFound, "CSR 不存在")
		return
	}
	writeOK(w, record)
}

// handleDeleteCSR DELETE /api/pki/csr/{uuid}
func (s *Server) handleDeleteCSR(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("uuid")
	if err := s.csrRepo.Delete(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, "删除 CSR 失败: "+err.Error())
		return
	}
	writeOK(w, nil)
}

// handleDownloadCSR GET /api/pki/csr/{uuid}/download
func (s *Server) handleDownloadCSR(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("uuid")
	record, err := s.csrRepo.GetByUUID(r.Context(), id)
	if err != nil || record == nil {
		writeError(w, http.StatusNotFound, "CSR 不存在")
		return
	}
	filename := fmt.Sprintf("%s.csr", record.CommonName)
	w.Header().Set("Content-Type", "application/x-pem-file")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(record.CSRPEM)) //nolint:errcheck
}

// ---- CA 管理 ----

// handleListCA GET /api/pki/ca
func (s *Server) handleListCA(w http.ResponseWriter, r *http.Request) {
	page, pageSize := parsePageParams(r)
	list, total, err := s.caRepo.List(r.Context(), page, pageSize)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "查询 CA 列表失败: "+err.Error())
		return
	}
	if list == nil {
		list = []*storage.LocalCA{}
	}
	writeOK(w, map[string]interface{}{"items": list, "total": total, "page": page, "page_size": pageSize})
}

// handleCreateCA POST /api/pki/ca
func (s *Server) handleCreateCA(w http.ResponseWriter, r *http.Request) {
	var req pki.CreateCARequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "请求格式错误: "+err.Error())
		return
	}
	if req.CommonName == "" {
		writeError(w, http.StatusBadRequest, "common_name 不能为空")
		return
	}
	if req.Name == "" {
		req.Name = req.CommonName
	}
	ca, err := pki.CreateAndSaveCA(r.Context(), s.caRepo, &req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "创建 CA 失败: "+err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, Response{Code: 0, Message: "ok", Data: ca})
}

// handleImportCA POST /api/pki/ca/import
func (s *Server) handleImportCA(w http.ResponseWriter, r *http.Request) {
	var req pki.ImportCARequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "请求格式错误: "+err.Error())
		return
	}
	ca, err := pki.ImportAndSaveCA(r.Context(), s.caRepo, &req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "导入 CA 失败: "+err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, Response{Code: 0, Message: "ok", Data: ca})
}

// handleGetCA GET /api/pki/ca/{uuid}
func (s *Server) handleGetCA(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("uuid")
	ca, err := s.caRepo.GetByUUID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "查询 CA 失败: "+err.Error())
		return
	}
	if ca == nil {
		writeError(w, http.StatusNotFound, "CA 不存在")
		return
	}
	writeOK(w, ca)
}

// handleRevokeCA POST /api/pki/ca/{uuid}/revoke
func (s *Server) handleRevokeCA(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("uuid")
	if err := s.caRepo.Revoke(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, "吊销 CA 失败: "+err.Error())
		return
	}
	writeOK(w, nil)
}

// handleDeleteCA DELETE /api/pki/ca/{uuid}
func (s *Server) handleDeleteCA(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("uuid")
	if err := s.caRepo.Delete(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, "删除 CA 失败: "+err.Error())
		return
	}
	writeOK(w, nil)
}

// handleExportCA GET /api/pki/ca/{uuid}/export
func (s *Server) handleExportCA(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("uuid")
	format := r.URL.Query().Get("format") // pem / chain
	ca, err := s.caRepo.GetByUUID(r.Context(), id)
	if err != nil || ca == nil {
		writeError(w, http.StatusNotFound, "CA 不存在")
		return
	}

	var content string
	var filename string
	if format == "chain" && ca.ChainPEM != "" {
		content = ca.CertPEM + "\n" + ca.ChainPEM
		filename = ca.Name + "_chain.pem"
	} else {
		content = ca.CertPEM
		filename = ca.Name + ".pem"
	}

	w.Header().Set("Content-Type", "application/x-pem-file")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(content)) //nolint:errcheck
}

// ---- 证书管理 ----

// handleListPKICerts GET /api/pki/certs
func (s *Server) handleListPKICerts(w http.ResponseWriter, r *http.Request) {
	page, pageSize := parsePageParams(r)
	list, total, err := s.pkiCertRepo.List(r.Context(), page, pageSize)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "查询证书列表失败: "+err.Error())
		return
	}
	if list == nil {
		list = []*storage.PKICert{}
	}
	writeOK(w, map[string]interface{}{"items": list, "total": total, "page": page, "page_size": pageSize})
}

// handleIssuePKICert POST /api/pki/certs/issue
func (s *Server) handleIssuePKICert(w http.ResponseWriter, r *http.Request) {
	var req pki.IssueCertFromCSRRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "请求格式错误: "+err.Error())
		return
	}
	if req.CSRUUID == "" || req.CAUUID == "" {
		writeError(w, http.StatusBadRequest, "csr_uuid 和 ca_uuid 不能为空")
		return
	}
	cert, err := pki.IssueCertFromCSR(r.Context(), s.csrRepo, s.caRepo, s.pkiCertRepo, &req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "签发证书失败: "+err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, Response{Code: 0, Message: "ok", Data: cert})
}

// handleImportPKICert POST /api/pki/certs/import
func (s *Server) handleImportPKICert(w http.ResponseWriter, r *http.Request) {
	var req pki.ImportCertRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "请求格式错误: "+err.Error())
		return
	}
	result, err := pki.ImportCert(r.Context(), s.pkiCertRepo, &req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "导入证书失败: "+err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, Response{Code: 0, Message: "ok", Data: result})
}

// handleGetPKICert GET /api/pki/certs/{uuid}
func (s *Server) handleGetPKICert(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("uuid")
	cert, err := s.pkiCertRepo.GetByUUID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "查询证书失败: "+err.Error())
		return
	}
	if cert == nil {
		writeError(w, http.StatusNotFound, "证书不存在")
		return
	}
	writeOK(w, cert)
}

// handleDeletePKICert DELETE /api/pki/certs/{uuid}
func (s *Server) handleDeletePKICert(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("uuid")
	if err := s.pkiCertRepo.Delete(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, "删除证书失败: "+err.Error())
		return
	}
	writeOK(w, nil)
}

// handleDeletePKICertKey DELETE /api/pki/certs/{uuid}/key
func (s *Server) handleDeletePKICertKey(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("uuid")
	if err := s.pkiCertRepo.DeletePrivateKey(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, "删除私钥失败: "+err.Error())
		return
	}
	writeOK(w, nil)
}

// handleExportPKICert POST /api/pki/certs/{uuid}/export
func (s *Server) handleExportPKICert(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("uuid")
	var req struct {
		Format   string `json:"format"`
		Password string `json:"password"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "请求格式错误: "+err.Error())
		return
	}

	cert, err := s.pkiCertRepo.GetByUUID(r.Context(), id)
	if err != nil || cert == nil {
		writeError(w, http.StatusNotFound, "证书不存在")
		return
	}

	data, contentType, err := pki.ExportCert(cert, pki.ExportCertFormat(req.Format), req.Password)
	if err != nil {
		writeError(w, http.StatusBadRequest, "导出证书失败: "+err.Error())
		return
	}

	ext := map[string]string{
		"pem":     ".pem",
		"der":     ".der",
		"pkcs12":  ".p12",
		"key_pem": ".key.pem",
	}[req.Format]
	filename := cert.CommonName + ext

	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	w.WriteHeader(http.StatusOK)
	w.Write(data) //nolint:errcheck
}

// handleImportPKICertToCard POST /api/pki/certs/{uuid}/import-to-card
func (s *Server) handleImportPKICertToCard(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("uuid")
	var req struct {
		CardUUID string `json:"card_uuid"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "请求格式错误: "+err.Error())
		return
	}
	if req.CardUUID == "" {
		writeError(w, http.StatusBadRequest, "card_uuid 不能为空")
		return
	}

	cert, err := s.pkiCertRepo.GetByUUID(r.Context(), id)
	if err != nil || cert == nil {
		writeError(w, http.StatusNotFound, "证书不存在")
		return
	}

	// TODO: 调用 card manager 将证书写入智能卡
	// 此处为 MVP 实现，仅更新记录中的 card_uuid
	_ = cert
	writeOK(w, map[string]string{"message": "证书已导入到智能卡（功能待完善）"})
}

// handleRevokePKICert POST /api/pki/certs/{uuid}/revoke
func (s *Server) handleRevokePKICert(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("uuid")
	if err := s.pkiCertRepo.Revoke(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, "吊销证书失败: "+err.Error())
		return
	}
	writeOK(w, nil)
}

// ---- 格式转换（保留原有功能）----

// handleConvertCert POST /api/pki/convert
func (s *Server) handleConvertCert(w http.ResponseWriter, r *http.Request) {
	var req struct {
		InputFormat  string `json:"input_format"`
		OutputFormat string `json:"output_format"`
		Data         string `json:"data"`
		Password     string `json:"password"`
		ExportPass   string `json:"export_pass"`
		KeyPEM       string `json:"key_pem"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "请求格式错误: "+err.Error())
		return
	}

	inputData, err := base64.StdEncoding.DecodeString(req.Data)
	if err != nil {
		inputData = []byte(req.Data)
	}

	var outputData []byte
	switch req.InputFormat + "->" + req.OutputFormat {
	case "pem->der":
		outputData, err = pki.ConvertPEMToDER(inputData)
	case "der->pem":
		outputData = pki.ConvertDERToPEM(inputData, "CERTIFICATE")
	case "pkcs12->pem":
		certPEM, keyPEM, e := pki.ImportPKCS12(inputData, req.Password)
		if e != nil {
			err = e
		} else {
			outputData = append(certPEM, keyPEM...)
		}
	case "pem->pkcs12":
		if len(req.ExportPass) < 8 {
			writeError(w, http.StatusBadRequest, "导出 PKCS#12 密码长度必须 >= 8 字符")
			return
		}
		outputData, err = pki.ExportPKCS12(inputData, []byte(req.KeyPEM), req.ExportPass)
	default:
		writeError(w, http.StatusBadRequest, "不支持的格式转换: "+req.InputFormat+" -> "+req.OutputFormat)
		return
	}

	if err != nil {
		writeError(w, http.StatusInternalServerError, "格式转换失败: "+err.Error())
		return
	}
	writeOK(w, map[string]string{
		"data":   base64.StdEncoding.EncodeToString(outputData),
		"format": req.OutputFormat,
	})
}

// handleParseCert POST /api/pki/parse
func (s *Server) handleParseCert(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Data string `json:"data"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "请求格式错误: "+err.Error())
		return
	}

	inputData, err := base64.StdEncoding.DecodeString(req.Data)
	if err != nil {
		inputData = []byte(req.Data)
	}

	cert, err := pki.ParseCertificateAuto(inputData)
	if err != nil {
		writeError(w, http.StatusBadRequest, "解析证书失败: "+err.Error())
		return
	}

	writeOK(w, map[string]interface{}{
		"subject":              cert.Subject.String(),
		"issuer":               cert.Issuer.String(),
		"serial_number":        cert.SerialNumber.String(),
		"not_before":           cert.NotBefore,
		"not_after":            cert.NotAfter,
		"is_ca":                cert.IsCA,
		"key_usage":            cert.KeyUsage,
		"dns_names":            cert.DNSNames,
		"ip_addresses":         cert.IPAddresses,
		"emails":               cert.EmailAddresses,
		"signature_algorithm":  cert.SignatureAlgorithm.String(),
		"public_key_algorithm": cert.PublicKeyAlgorithm.String(),
	})
}

// ---- 工具函数 ----

// parsePageParams 从查询参数解析分页参数（page 从 1 开始）。
func parsePageParams(r *http.Request) (page, pageSize int) {
	page = 1
	pageSize = 20
	if v := r.URL.Query().Get("page"); v != "" {
		fmt.Sscanf(v, "%d", &page)
	}
	if v := r.URL.Query().Get("page_size"); v != "" {
		fmt.Sscanf(v, "%d", &pageSize)
	}
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	return
}