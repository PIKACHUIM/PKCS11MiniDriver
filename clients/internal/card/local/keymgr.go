package local

import (
	"context"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"fmt"

	cryptoutil "github.com/globaltrusts/client-card/internal/crypto"
	"github.com/globaltrusts/client-card/internal/storage"
	"github.com/google/uuid"
)

// KeyGenRequest 是生成密钥对的请求参数。
type KeyGenRequest struct {
	CardUUID string
	CertType storage.CertType
	// KeyType: rsa2048 / rsa4096 / ec256 / ec384 / ec521
	KeyType string
	Remark  string
}

// KeyGenResult 是生成密钥对的结果。
type KeyGenResult struct {
	CertUUID    string
	PublicKeyDER []byte // DER 格式公钥（PKIX）
}

// KeyManager 提供本地卡片的密钥管理操作。
type KeyManager struct {
	certRepo *storage.CertRepo
	cardRepo *storage.CardRepo
}

// NewKeyManager 创建密钥管理器。
func NewKeyManager(certRepo *storage.CertRepo, cardRepo *storage.CardRepo) *KeyManager {
	return &KeyManager{certRepo: certRepo, cardRepo: cardRepo}
}

// CreateCard 创建一张新的本地智能卡。
// userPassword 是用户密码（明文），用于加密主密钥。
// cardPassword 可选，是卡片独立密码（留空则不设置）。
func CreateCard(ctx context.Context, cardRepo *storage.CardRepo, userUUID, cardName, userPassword, cardPassword, remark string) (*storage.Card, error) {
	// 1. 生成 32 字节随机主密钥
	masterKey, err := cryptoutil.GenerateKey()
	if err != nil {
		return nil, fmt.Errorf("生成主密钥失败: %w", err)
	}
	defer zeroBytes(masterKey)

	card := &storage.Card{
		UUID:     uuid.New().String(),
		SlotType: storage.SlotTypeLocal,
		CardName: cardName,
		UserUUID: userUUID,
		Remark:   remark,
	}

	// 2. 用用户密码加密主密钥
	userEntry, err := encryptMasterKey(masterKey, []byte(userPassword), "user", userUUID)
	if err != nil {
		return nil, fmt.Errorf("加密主密钥（用户密码）失败: %w", err)
	}
	card.CardKeys = append(card.CardKeys, *userEntry)

	// 3. 如果设置了卡片密码，额外加密一份
	if cardPassword != "" {
		cardEntry, err := encryptMasterKey(masterKey, []byte(cardPassword), "card", "")
		if err != nil {
			return nil, fmt.Errorf("加密主密钥（卡片密码）失败: %w", err)
		}
		card.CardKeys = append(card.CardKeys, *cardEntry)
	}

	if err := cardRepo.Create(ctx, card); err != nil {
		return nil, fmt.Errorf("保存卡片失败: %w", err)
	}

	return card, nil
}

// GenerateKeyPair 在指定卡片中生成密钥对并存储。
// masterKey 是已解锁的卡片主密钥。
func (m *KeyManager) GenerateKeyPair(ctx context.Context, req KeyGenRequest, masterKey []byte) (*KeyGenResult, error) {
	// 1. 生成密钥对
	privDER, pubDER, err := generateKeyPair(req.KeyType)
	if err != nil {
		return nil, fmt.Errorf("生成密钥对失败: %w", err)
	}

	// 2. 加密私钥
	cert, err := encryptAndStoreCert(ctx, m.certRepo, req, masterKey, privDER, pubDER)
	if err != nil {
		return nil, err
	}

	return &KeyGenResult{
		CertUUID:    cert.UUID,
		PublicKeyDER: pubDER,
	}, nil
}

// ImportCertificate 导入已有证书（公钥部分）到卡片。
// 用于导入 X.509 证书与已存储私钥关联。
func (m *KeyManager) ImportCertificate(ctx context.Context, cardUUID string, certDER []byte, remark string) (*storage.Certificate, error) {
	cert := &storage.Certificate{
		UUID:        uuid.New().String(),
		SlotType:    storage.SlotTypeLocal,
		CardUUID:    cardUUID,
		CertType:    storage.CertTypeX509,
		KeyType:     "x509",
		CertContent: certDER,
		Remark:      remark,
	}

	if err := m.certRepo.Create(ctx, cert); err != nil {
		return nil, fmt.Errorf("导入证书失败: %w", err)
	}
	return cert, nil
}

// ImportPrivateKey 导入私钥到卡片（DER 格式，已有主密钥加密）。
func (m *KeyManager) ImportPrivateKey(ctx context.Context, req KeyGenRequest, masterKey, privDER, pubDER []byte) (*KeyGenResult, error) {
	cert, err := encryptAndStoreCert(ctx, m.certRepo, req, masterKey, privDER, pubDER)
	if err != nil {
		return nil, err
	}
	return &KeyGenResult{
		CertUUID:    cert.UUID,
		PublicKeyDER: pubDER,
	}, nil
}

// ---- 内部工具函数 ----

// encryptMasterKey 用密码加密主密钥，生成一条 CardKeyEntry。
func encryptMasterKey(masterKey, password []byte, keyType, userUUID string) (*storage.CardKeyEntry, error) {
	salt, err := cryptoutil.GenerateSalt()
	if err != nil {
		return nil, err
	}

	// AES 密钥 = HMAC(password, salt)
	aesKey := cryptoutil.HMACSHA256(password, salt)
	encMasterKey, err := cryptoutil.EncryptAES256GCM(aesKey, masterKey)
	if err != nil {
		return nil, err
	}

	return &storage.CardKeyEntry{
		KeyType:      keyType,
		UserUUID:     userUUID,
		Salt:         salt,
		EncMasterKey: encMasterKey,
	}, nil
}

// encryptAndStoreCert 加密私钥并存储证书记录。
func encryptAndStoreCert(ctx context.Context, certRepo *storage.CertRepo, req KeyGenRequest, masterKey, privDER, pubDER []byte) (*storage.Certificate, error) {
	// 1. 生成临时密钥（32 字节随机）
	tempKey, err := cryptoutil.GenerateKey()
	if err != nil {
		return nil, fmt.Errorf("生成临时密钥失败: %w", err)
	}
	defer zeroBytes(tempKey)

	// 2. 生成临时密钥盐值
	tempKeySalt, err := cryptoutil.GenerateSalt()
	if err != nil {
		return nil, fmt.Errorf("生成临时密钥盐值失败: %w", err)
	}

	// 3. 用 HMAC(masterKey, salt) 加密临时密钥
	tempKeyAESKey := cryptoutil.HMACSHA256(masterKey, tempKeySalt)
	tempKeyEnc, err := cryptoutil.EncryptAES256GCM(tempKeyAESKey, tempKey)
	if err != nil {
		return nil, fmt.Errorf("加密临时密钥失败: %w", err)
	}

	// 4. 用临时密钥加密私钥
	privateData, err := cryptoutil.EncryptAES256GCM(tempKey, privDER)
	if err != nil {
		return nil, fmt.Errorf("加密私钥失败: %w", err)
	}

	cert := &storage.Certificate{
		UUID:        uuid.New().String(),
		SlotType:    storage.SlotTypeLocal,
		CardUUID:    req.CardUUID,
		CertType:    req.CertType,
		KeyType:     req.KeyType,
		CertContent: pubDER,
		TempKeySalt: tempKeySalt,
		TempKeyEnc:  tempKeyEnc,
		PrivateData: privateData,
		Remark:      req.Remark,
	}

	if err := certRepo.Create(ctx, cert); err != nil {
		return nil, fmt.Errorf("保存证书失败: %w", err)
	}
	return cert, nil
}

// generateKeyPair 根据 keyType 生成密钥对，返回 (privDER, pubDER)。
func generateKeyPair(keyType string) (privDER, pubDER []byte, err error) {
	switch keyType {
	case "rsa1024":
		return generateRSA(1024)
	case "rsa2048":
		return generateRSA(2048)
	case "rsa4096":
		return generateRSA(4096)
	case "rsa8192":
		return generateRSA(8192)
	case "ec256":
		return generateEC(elliptic.P256())
	case "ec384":
		return generateEC(elliptic.P384())
	case "ec521":
		return generateEC(elliptic.P521())
	case "ed25519":
		return generateEd25519()
	default:
		return nil, nil, fmt.Errorf("不支持的密钥类型: %s（支持 rsa1024/rsa2048/rsa4096/rsa8192/ec256/ec384/ec521/ed25519）", keyType)
	}
}

func generateRSA(bits int) (privDER, pubDER []byte, err error) {
	key, err := rsa.GenerateKey(rand.Reader, bits)
	if err != nil {
		return nil, nil, fmt.Errorf("生成 RSA-%d 密钥失败: %w", bits, err)
	}

	privDER, err = x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		return nil, nil, fmt.Errorf("序列化 RSA 私钥失败: %w", err)
	}

	pubDER, err = x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		return nil, nil, fmt.Errorf("序列化 RSA 公钥失败: %w", err)
	}
	return
}

func generateEC(curve elliptic.Curve) (privDER, pubDER []byte, err error) {
	key, err := ecdsa.GenerateKey(curve, rand.Reader)
	if err != nil {
		return nil, nil, fmt.Errorf("生成 EC 密钥失败: %w", err)
	}

	privDER, err = x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		return nil, nil, fmt.Errorf("序列化 EC 私钥失败: %w", err)
	}

	pubDER, err = x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		return nil, nil, fmt.Errorf("序列化 EC 公钥失败: %w", err)
	}
	return
}

// generateEd25519 生成 Ed25519 密钥对。
func generateEd25519() (privDER, pubDER []byte, err error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, nil, fmt.Errorf("生成 Ed25519 密钥失败: %w", err)
	}

	privDER, err = x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		return nil, nil, fmt.Errorf("序列化 Ed25519 私钥失败: %w", err)
	}

	pubDER, err = x509.MarshalPKIXPublicKey(pub)
	if err != nil {
		return nil, nil, fmt.Errorf("序列化 Ed25519 公钥失败: %w", err)
	}
	return
}

// SupportedKeyTypes 返回所有支持的密钥类型列表。
func SupportedKeyTypes() []string {
	return []string{
		"rsa1024", "rsa2048", "rsa4096", "rsa8192",
		"ec256", "ec384", "ec521",
		"ed25519",
	}
}

// zeroBytes 清零字节切片（防止内存泄露密钥）。
func zeroBytes(b []byte) {
	for i := range b {
		b[i] = 0
	}
}
