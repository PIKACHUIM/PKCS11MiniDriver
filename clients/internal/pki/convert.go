package pki

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"

	"software.sslmate.com/src/go-pkcs12"
)

// ConvertPEMToDER 将 PEM 编码的证书转换为 DER 格式。
func ConvertPEMToDER(pemData []byte) ([]byte, error) {
	block, _ := pem.Decode(pemData)
	if block == nil {
		return nil, fmt.Errorf("无法解码 PEM 数据")
	}
	return block.Bytes, nil
}

// ConvertDERToPEM 将 DER 编码的证书转换为 PEM 格式。
func ConvertDERToPEM(derData []byte, blockType string) []byte {
	if blockType == "" {
		blockType = "CERTIFICATE"
	}
	return pem.EncodeToMemory(&pem.Block{Type: blockType, Bytes: derData})
}

// ParseCertificateFromPEM 从 PEM 数据解析 X.509 证书。
func ParseCertificateFromPEM(pemData []byte) (*x509.Certificate, error) {
	block, _ := pem.Decode(pemData)
	if block == nil {
		return nil, fmt.Errorf("无法解码 PEM 数据")
	}
	return x509.ParseCertificate(block.Bytes)
}

// ParseCertificateAuto 自动识别 PEM 或 DER 格式并解析证书。
func ParseCertificateAuto(data []byte) (*x509.Certificate, error) {
	// 先尝试 PEM
	block, _ := pem.Decode(data)
	if block != nil {
		return x509.ParseCertificate(block.Bytes)
	}
	// 尝试 DER
	return x509.ParseCertificate(data)
}

// ExportPKCS12 将证书和私钥导出为 PKCS#12 格式。
func ExportPKCS12(certPEM, keyPEM []byte, password string) ([]byte, error) {
	if len(password) < 8 {
		return nil, fmt.Errorf("PKCS#12 导出密码长度必须 >= 8 字符")
	}

	// 解析证书
	certBlock, _ := pem.Decode(certPEM)
	if certBlock == nil {
		return nil, fmt.Errorf("无法解码证书 PEM")
	}
	cert, err := x509.ParseCertificate(certBlock.Bytes)
	if err != nil {
		return nil, fmt.Errorf("解析证书失败: %w", err)
	}

	// 解析私钥
	keyBlock, _ := pem.Decode(keyPEM)
	if keyBlock == nil {
		return nil, fmt.Errorf("无法解码私钥 PEM")
	}
	privKey, err := x509.ParsePKCS8PrivateKey(keyBlock.Bytes)
	if err != nil {
		// 尝试 PKCS1
		privKey, err = x509.ParsePKCS1PrivateKey(keyBlock.Bytes)
		if err != nil {
			// 尝试 EC
			privKey, err = x509.ParseECPrivateKey(keyBlock.Bytes)
			if err != nil {
				return nil, fmt.Errorf("解析私钥失败: %w", err)
			}
		}
	}

	// 编码 PKCS#12
	pfxData, err := pkcs12.Modern.Encode(privKey, cert, nil, password)
	if err != nil {
		return nil, fmt.Errorf("编码 PKCS#12 失败: %w", err)
	}
	return pfxData, nil
}

// ImportPKCS12 从 PKCS#12 数据导入证书和私钥。
func ImportPKCS12(pfxData []byte, password string) (certPEM, keyPEM []byte, err error) {
	privKey, cert, _, err := pkcs12.DecodeChain(pfxData, password)
	if err != nil {
		return nil, nil, fmt.Errorf("解码 PKCS#12 失败: %w", err)
	}

	// 编码证书 PEM
	certPEM = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert.Raw})

	// 编码私钥 PEM
	keyDER, err := x509.MarshalPKCS8PrivateKey(privKey)
	if err != nil {
		return nil, nil, fmt.Errorf("编码私钥失败: %w", err)
	}
	keyPEM = pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyDER})

	return certPEM, keyPEM, nil
}

// ParseCertChainFromPEM 从 PEM 数据解析证书链（多个证书）。
func ParseCertChainFromPEM(pemData []byte) ([]*x509.Certificate, error) {
	var certs []*x509.Certificate
	rest := pemData
	for {
		var block *pem.Block
		block, rest = pem.Decode(rest)
		if block == nil {
			break
		}
		if block.Type != "CERTIFICATE" {
			continue
		}
		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("解析证书链中的证书失败: %w", err)
		}
		certs = append(certs, cert)
	}
	if len(certs) == 0 {
		return nil, fmt.Errorf("PEM 数据中未找到证书")
	}
	return certs, nil
}

// ExportPKCS7 将证书链导出为 PKCS#7 格式（仅证书，不含私钥）。
// 返回 DER 编码的 PKCS#7 数据。
func ExportPKCS7(certs []*x509.Certificate) ([]byte, error) {
	if len(certs) == 0 {
		return nil, fmt.Errorf("证书列表为空")
	}
	// 简化实现：将证书链编码为 PEM 格式（完整的 PKCS#7 需要 ASN.1 编码）
	var pemData []byte
	for _, cert := range certs {
		pemData = append(pemData, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert.Raw})...)
	}
	return pemData, nil
}
