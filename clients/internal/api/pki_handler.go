package api

import (
	"encoding/base64"
	"net/http"

	"github.com/globaltrusts/client-card/internal/pki"
)

// ---- 本地 PKI Handler ----

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

	writeJSON(w, http.StatusCreated, Response{
		Code:    0,
		Message: "ok",
		Data: map[string]string{
			"cert_pem": string(result.CertPEM),
			"key_pem":  string(result.KeyPEM),
			"cert_der":  base64.StdEncoding.EncodeToString(result.CertDER),
		},
	})
}

// handleGenerateCSR POST /api/pki/csr
func (s *Server) handleGenerateCSR(w http.ResponseWriter, r *http.Request) {
	var req pki.CSRRequest
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

	result, err := pki.GenerateCSR(&req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "生成 CSR 失败: "+err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, Response{
		Code:    0,
		Message: "ok",
		Data: map[string]string{
			"csr_pem": string(result.CSRPEM),
			"key_pem": string(result.KeyPEM),
		},
	})
}

// handleCreateLocalCA POST /api/pki/ca
func (s *Server) handleCreateLocalCA(w http.ResponseWriter, r *http.Request) {
	var req struct {
		CommonName   string `json:"common_name"`
		Organization string `json:"organization"`
		Country      string `json:"country"`
		KeyType      string `json:"key_type"`
		ValidDays    int    `json:"valid_days"`
	}
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
	if req.ValidDays <= 0 {
		req.ValidDays = 3650 // CA 默认 10 年
	}

	// 生成 CA 根证书（自签名）
	result, err := pki.GenerateSelfSigned(&pki.SelfSignRequest{
		CommonName:        req.CommonName,
		Organization:      req.Organization,
		Country:           req.Country,
		KeyType:           req.KeyType,
		ValidDays:         req.ValidDays,
		IsCA:              true,
		PathLenConstraint: 1,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "创建本地 CA 失败: "+err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, Response{
		Code:    0,
		Message: "ok",
		Data: map[string]string{
			"cert_pem": string(result.CertPEM),
			"key_pem":  string(result.KeyPEM),
		},
	})
}

// handleIssueCert POST /api/pki/ca/issue
func (s *Server) handleIssueCert(w http.ResponseWriter, r *http.Request) {
	var req struct {
		CACertPEM string `json:"ca_cert_pem"`
		CAKeyPEM  string `json:"ca_key_pem"`
		pki.IssueCertRequest
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "请求格式错误: "+err.Error())
		return
	}

	if req.CACertPEM == "" || req.CAKeyPEM == "" {
		writeError(w, http.StatusBadRequest, "ca_cert_pem 和 ca_key_pem 不能为空")
		return
	}
	if req.CommonName == "" {
		writeError(w, http.StatusBadRequest, "common_name 不能为空")
		return
	}
	if req.KeyType == "" {
		req.KeyType = "ec256"
	}

	// 解析 CA 证书和私钥
	caCert, err := pki.ParseCertificateFromPEM([]byte(req.CACertPEM))
	if err != nil {
		writeError(w, http.StatusBadRequest, "解析 CA 证书失败: "+err.Error())
		return
	}

	caKey, err := pki.ParsePrivateKeyFromPEM([]byte(req.CAKeyPEM))
	if err != nil {
		writeError(w, http.StatusBadRequest, "解析 CA 私钥失败: "+err.Error())
		return
	}

	result, err := pki.IssueCertificate(caCert, caKey, &req.IssueCertRequest)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "签发证书失败: "+err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, Response{
		Code:    0,
		Message: "ok",
		Data: map[string]string{
			"cert_pem": string(result.CertPEM),
			"key_pem":  string(result.KeyPEM),
		},
	})
}

// handleConvertCert POST /api/pki/convert
func (s *Server) handleConvertCert(w http.ResponseWriter, r *http.Request) {
	var req struct {
		InputFormat  string `json:"input_format"`  // pem/der/pkcs12
		OutputFormat string `json:"output_format"` // pem/der/pkcs12
		Data         string `json:"data"`          // base64 编码的输入数据
		Password     string `json:"password"`      // PKCS#12 密码
		ExportPass   string `json:"export_pass"`   // 导出 PKCS#12 时的密码
		KeyPEM       string `json:"key_pem"`       // 导出 PKCS#12 时需要私钥
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "请求格式错误: "+err.Error())
		return
	}

	inputData, err := base64.StdEncoding.DecodeString(req.Data)
	if err != nil {
		writeError(w, http.StatusBadRequest, "data 必须是 base64 编码")
		return
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
		if req.ExportPass == "" || len(req.ExportPass) < 8 {
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
		Data string `json:"data"` // base64 编码的证书数据（PEM 或 DER）
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "请求格式错误: "+err.Error())
		return
	}

	inputData, err := base64.StdEncoding.DecodeString(req.Data)
	if err != nil {
		// 可能直接是 PEM 文本
		inputData = []byte(req.Data)
	}

	cert, err := pki.ParseCertificateAuto(inputData)
	if err != nil {
		writeError(w, http.StatusBadRequest, "解析证书失败: "+err.Error())
		return
	}

	writeOK(w, map[string]interface{}{
		"subject":      cert.Subject.String(),
		"issuer":       cert.Issuer.String(),
		"serial_number": cert.SerialNumber.String(),
		"not_before":   cert.NotBefore,
		"not_after":    cert.NotAfter,
		"is_ca":        cert.IsCA,
		"key_usage":    cert.KeyUsage,
		"dns_names":    cert.DNSNames,
		"ip_addresses": cert.IPAddresses,
		"emails":       cert.EmailAddresses,
		"signature_algorithm": cert.SignatureAlgorithm.String(),
		"public_key_algorithm": cert.PublicKeyAlgorithm.String(),
	})
}
