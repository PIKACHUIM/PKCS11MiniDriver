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
	out, err := CreateCardWithCreds(ctx, cardRepo, CreateCardArgs{
		UserUUID:     userUUID,
		CardName:     cardName,
		UserPassword: userPassword,
		CardPassword: cardPassword,
		Remark:       remark,
	})
	if err != nil {
		return nil, err
	}
	return out.Card, nil
}

// CreateCardArgs 是带 PIN/PUK/AdminKey 的卡片创建参数。
type CreateCardArgs struct {
	UserUUID     string
	CardName     string
	UserPassword string // 可选，与 CardPassword 至少一个
	CardPassword string // 可选
	PIN          string // 可选；为空且 GeneratePIN=true 时自动生成
	PUK          string // 可选；为空且 GeneratePUK=true 时自动生成（默认 true）
	AdminKey     string // 可选；为空且 GenerateAdmin=true 时自动生成（默认 true）
	GeneratePIN  bool
	GeneratePUK  bool
	GenerateAdmin bool
	Remark       string
}

// CreateCardResult 是创建结果，包含卡片及一次性返回的明文 PUK/AdminKey。
// 调用方必须把 PUK 与 AdminKey 提示给用户保存；后端只存加密副本。
type CreateCardResult struct {
	Card     *storage.Card
	PIN      string // 仅当自动生成时返回
	PUK      string // 仅当自动生成时返回
	AdminKey string // 仅当自动生成时返回
}

// CreateCardWithCreds 创建卡片并可选自动生成 PIN/PUK/AdminKey 三级凭据。
// 默认行为：PUK 与 AdminKey 若未提供则自动生成 16 字节随机值（hex 编码 32 字符）。
// 所有凭据都作为独立 CardKeyEntry 各自加密一份"主密钥副本"存储，互不推导。
func CreateCardWithCreds(ctx context.Context, cardRepo *storage.CardRepo, args CreateCardArgs) (*CreateCardResult, error) {
	// 1. 生成 32 字节随机主密钥
	masterKey, err := cryptoutil.GenerateKey()
	if err != nil {
		return nil, fmt.Errorf("生成主密钥失败: %w", err)
	}
	defer zeroBytes(masterKey)

	card := &storage.Card{
		UUID:     uuid.New().String(),
		SlotType: storage.SlotTypeLocal,
		CardName: args.CardName,
		UserUUID: args.UserUUID,
		Remark:   args.Remark,
	}

	out := &CreateCardResult{}

	// 2. 用户密码（可选）
	if args.UserPassword != "" {
		userEntry, err := encryptMasterKey(masterKey, []byte(args.UserPassword), "user", args.UserUUID)
		if err != nil {
			return nil, fmt.Errorf("加密主密钥（用户密码）失败: %w", err)
		}
		card.CardKeys = append(card.CardKeys, *userEntry)
	}

	// 3. 卡片密码（可选，兼容旧字段）
	if args.CardPassword != "" {
		cardEntry, err := encryptMasterKey(masterKey, []byte(args.CardPassword), "card", "")
		if err != nil {
			return nil, fmt.Errorf("加密主密钥（卡片密码）失败: %w", err)
		}
		card.CardKeys = append(card.CardKeys, *cardEntry)
	}

	// 4. PIN（与 card 同义，保留独立类型便于区分 UI）
	pin := args.PIN
	if pin == "" && args.GeneratePIN {
		pin = randomCred(6) // 6 位数字感观
	}
	if pin != "" {
		pinEntry, err := encryptMasterKey(masterKey, []byte(pin), "pin", "")
		if err != nil {
			return nil, fmt.Errorf("加密主密钥（PIN）失败: %w", err)
		}
		card.CardKeys = append(card.CardKeys, *pinEntry)
		if args.GeneratePIN {
			out.PIN = pin
		}
	}

	// 5. PUK
	puk := args.PUK
	generatePUK := args.GeneratePUK || (args.PUK == "")
	if puk == "" && generatePUK {
		puk = randomCred(16)
	}
	if puk != "" {
		pukEntry, err := encryptMasterKey(masterKey, []byte(puk), "puk", "")
		if err != nil {
			return nil, fmt.Errorf("加密主密钥（PUK）失败: %w", err)
		}
		card.CardKeys = append(card.CardKeys, *pukEntry)
		if generatePUK {
			out.PUK = puk
		}
	}

	// 6. Admin Key
	admin := args.AdminKey
	generateAdmin := args.GenerateAdmin || (args.AdminKey == "")
	if admin == "" && generateAdmin {
		admin = randomCred(16)
	}
	if admin != "" {
		adminEntry, err := encryptMasterKey(masterKey, []byte(admin), "admin", "")
		if err != nil {
			return nil, fmt.Errorf("加密主密钥（AdminKey）失败: %w", err)
		}
		card.CardKeys = append(card.CardKeys, *adminEntry)
		if generateAdmin {
			out.AdminKey = admin
		}
	}

	if err := cardRepo.Create(ctx, card); err != nil {
		return nil, fmt.Errorf("保存卡片失败: %w", err)
	}

	out.Card = card
	return out, nil
}

// ResetPIN 用 PUK 或 AdminKey 解锁主密钥，然后用 newPIN 重新加密 pin 条目。
// 同时清零 puk/pin 的 Attempts/Locked。
// keyType: "puk" 或 "admin"
func ResetPIN(ctx context.Context, cardRepo *storage.CardRepo, card *storage.Card, keyType, secret, newPIN string) error {
	if keyType != "puk" && keyType != "admin" {
		return fmt.Errorf("keyType 必须为 puk 或 admin")
	}
	if newPIN == "" {
		return fmt.Errorf("新 PIN 不能为空")
	}

	masterKey, err := tryUnlockByType(card, keyType, secret)
	if err != nil {
		// 失败：递增对应条目的 Attempts，超过阈值则锁定
		if ferr := bumpFailure(ctx, cardRepo, card, keyType); ferr != nil {
			return fmt.Errorf("%w (记录失败次数时出错: %v)", err, ferr)
		}
		return err
	}
	defer zeroBytes(masterKey)

	// 清零 puk/pin 失败状态
	for i := range card.CardKeys {
		kt := card.CardKeys[i].KeyType
		if kt == "pin" || kt == "puk" || kt == "card" || kt == "user" {
			card.CardKeys[i].Attempts = 0
			card.CardKeys[i].Locked = false
		}
	}

	// 用新 PIN 重新加密一条 pin 条目（若不存在则新增，若存在则覆盖第一条）
	newEntry, err := encryptMasterKey(masterKey, []byte(newPIN), "pin", "")
	if err != nil {
		return fmt.Errorf("加密新 PIN 失败: %w", err)
	}
	replaced := false
	for i := range card.CardKeys {
		if card.CardKeys[i].KeyType == "pin" {
			card.CardKeys[i] = *newEntry
			replaced = true
			break
		}
	}
	if !replaced {
		card.CardKeys = append(card.CardKeys, *newEntry)
	}

	// 同步清零卡片级 PIN 失败标记
	card.PINFailedCount = 0
	card.PINLocked = false

	return cardRepo.Update(ctx, card)
}

// ResetPUK 用 AdminKey 解锁主密钥，然后用 newPUK 重新加密 puk 条目。
func ResetPUK(ctx context.Context, cardRepo *storage.CardRepo, card *storage.Card, adminKey, newPUK string) error {
	if newPUK == "" {
		return fmt.Errorf("新 PUK 不能为空")
	}
	masterKey, err := tryUnlockByType(card, "admin", adminKey)
	if err != nil {
		if ferr := bumpFailure(ctx, cardRepo, card, "admin"); ferr != nil {
			return fmt.Errorf("%w (记录失败次数时出错: %v)", err, ferr)
		}
		return err
	}
	defer zeroBytes(masterKey)

	newEntry, err := encryptMasterKey(masterKey, []byte(newPUK), "puk", "")
	if err != nil {
		return fmt.Errorf("加密新 PUK 失败: %w", err)
	}
	replaced := false
	for i := range card.CardKeys {
		if card.CardKeys[i].KeyType == "puk" {
			card.CardKeys[i] = *newEntry
			replaced = true
			break
		}
	}
	if !replaced {
		card.CardKeys = append(card.CardKeys, *newEntry)
	}
	return cardRepo.Update(ctx, card)
}

// tryUnlockByType 只用指定类型的条目尝试解密主密钥。
// 遍历相同 keyType 的所有 entry；若没有该类型条目或全部失败则返回错误。
// Locked=true 的条目会被跳过，若全部被锁则报 CKR_PIN_LOCKED 等价错误。
func tryUnlockByType(card *storage.Card, keyType, secret string) ([]byte, error) {
	hasType := false
	allLocked := true
	for _, e := range card.CardKeys {
		if e.KeyType != keyType {
			continue
		}
		hasType = true
		if e.Locked {
			continue
		}
		allLocked = false
		aesKey := cryptoutil.HMACSHA256([]byte(secret), e.Salt)
		masterKey, err := cryptoutil.DecryptAES256GCM(aesKey, e.EncMasterKey)
		if err == nil {
			return masterKey, nil
		}
	}
	if !hasType {
		return nil, fmt.Errorf("卡片未设置 %s 凭据", keyType)
	}
	if allLocked {
		return nil, fmt.Errorf("%s 已被锁定", keyType)
	}
	return nil, fmt.Errorf("%s 验证失败", keyType)
}

// bumpFailure 为指定 keyType 的所有条目递增失败次数；
// PUK 达到 10 次锁定；Admin 达到 10 次锁定。
func bumpFailure(ctx context.Context, cardRepo *storage.CardRepo, card *storage.Card, keyType string) error {
	maxAttempts := 10
	if keyType == "pin" {
		maxAttempts = card.PINRetries
		if maxAttempts <= 0 {
			maxAttempts = 3
		}
	}
	changed := false
	for i := range card.CardKeys {
		if card.CardKeys[i].KeyType != keyType {
			continue
		}
		card.CardKeys[i].Attempts++
		if card.CardKeys[i].Attempts >= maxAttempts {
			card.CardKeys[i].Locked = true
		}
		changed = true
	}
	if !changed {
		return nil
	}
	return cardRepo.Update(ctx, card)
}

// randomCred 生成 n 字节随机 hex 凭据（返回 2n 字符字符串）。
func randomCred(nBytes int) string {
	b := make([]byte, nBytes)
	if _, err := rand.Read(b); err != nil {
		// 兜底：时间戳（不应该发生）
		return fmt.Sprintf("fallback-%d", nBytes)
	}
	out := make([]byte, 2*nBytes)
	const hexCh = "0123456789abcdef"
	for i, v := range b {
		out[i*2] = hexCh[v>>4]
		out[i*2+1] = hexCh[v&0x0f]
	}
	return string(out)
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
