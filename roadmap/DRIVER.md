# PKCS#11 功能映射关系网

## 一、顶层层级关系

```
Library（库）
  └── Slot[1]  "Pikachu Cloud SmartCard Slot"
        └── Token  "Pikachu Secure SmartCard / SN:0123456789A"
              └── Session[1]  (只读/读写 × 公开/用户/SO)
                    ├── Object[1] DATA
                    ├── Object[2] SECRET_KEY
                    ├── Object[3] PUBLIC_KEY
                    └── Object[4] PRIVATE_KEY
```

| 层级 | 标识 | 说明 |
|------|------|------|
| Library | — | Pikachu CSP PKCS #11 Library，制造商: Pikachu SmartCard MiniDriver |
| Slot | ID = 1 | Pikachu Cloud SmartCard Slot |
| Token | Label: Pikachu Secure SmartCard，SN: 0123456789A | PIN 长度 4~32 位 |
| Session | ID = 1 | 只读/读写 × 公开/用户/SO 三种角色 |

---

## 二、会话状态机（Session State Machine）

```
C_OpenSession(只读)  →  RO_PUBLIC
C_OpenSession(读写)  →  RW_PUBLIC

RO_PUBLIC + C_Login(USER)  →  RO_USER
RO_PUBLIC + C_Login(SO)    →  ❌ SESSION_READ_ONLY_EXISTS

RW_PUBLIC + C_Login(USER)  →  RW_USER
RW_PUBLIC + C_Login(SO)    →  RW_SO

RO_USER / RW_USER / RW_SO + C_Logout  →  回到对应 PUBLIC 状态

RW_SO   可执行 C_InitPIN
RW_USER 可执行 C_SetPIN
已登录状态再次 C_Login  →  ❌ ALREADY_LOGGED_IN
```

---

## 三、密钥 ↔ 算法 ↔ 操作 映射关系

| 对象句柄 | 对象类型 | 可用算法 | 支持操作 |
|---------|---------|---------|---------|
| Handle=2 `SECRET_KEY` | 对称密钥 | `CKM_DES3_CBC` | 加密 / 解密 |
| Handle=2 `SECRET_KEY` | 对称密钥 | `CKM_AES_CBC` | 加密 / 解密 |
| Handle=2 `SECRET_KEY` | 对称密钥 | `CKM_XOR_BASE_AND_DATA` | 密钥派生 |
| Handle=2 `SECRET_KEY` | 对称密钥 | `CKM_SHA_1` | 摘要（DigestKey） |
| Handle=3 `PUBLIC_KEY` | RSA 公钥 | `CKM_RSA_PKCS` | 加密 / 验签 / 包装密钥 |
| Handle=3 `PUBLIC_KEY` | RSA 公钥 | `CKM_RSA_PKCS_OAEP` | 加密 |
| Handle=3 `PUBLIC_KEY` | RSA 公钥 | `CKM_SHA1_RSA_PKCS` | 验签 |
| Handle=4 `PRIVATE_KEY` | RSA 私钥 | `CKM_RSA_PKCS` | 解密 / 签名 / 解包密钥 |
| Handle=4 `PRIVATE_KEY` | RSA 私钥 | `CKM_RSA_PKCS_OAEP` | 解密 |
| Handle=4 `PRIVATE_KEY` | RSA 私钥 | `CKM_SHA1_RSA_PKCS` | 签名 |

---

## 四、操作状态机（Active Operation State Machine）

通过全局变量 `pkcs11_mock_active_operation` 驱动，同一时刻只允许一种（或两种组合）操作处于活跃状态。

### 单操作状态

| 触发函数 | 进入状态 | 结束函数 | 退出状态 |
|---------|---------|---------|---------|
| `FindObjectsInit` | FIND | `FindObjectsFinal` | NONE |
| `EncryptInit` | ENCRYPT | `Encrypt` / `EncryptFinal` | NONE |
| `DecryptInit` | DECRYPT | `Decrypt` / `DecryptFinal` | NONE |
| `DigestInit` | DIGEST | `Digest` / `DigestFinal` | NONE |
| `SignInit` | SIGN | `Sign` / `SignFinal` | NONE |
| `SignRecoverInit` | SIGN_RECOVER | `SignRecover` | NONE |
| `VerifyInit` | VERIFY | `Verify` / `VerifyFinal` | NONE |
| `VerifyRecoverInit` | VERIFY_RECOVER | `VerifyRecover` | NONE |

### 双操作叠加状态

| 当前状态 | 叠加触发 | 进入组合状态 |
|---------|---------|---------|
| ENCRYPT | `DigestInit` | DIGEST_ENCRYPT |
| ENCRYPT | `SignInit` | SIGN_ENCRYPT |
| DECRYPT | `DigestInit` | DECRYPT_DIGEST |
| DECRYPT | `VerifyInit` | DECRYPT_VERIFY |
| DIGEST | `EncryptInit` | DIGEST_ENCRYPT |
| DIGEST | `DecryptInit` | DECRYPT_DIGEST |
| SIGN | `EncryptInit` | SIGN_ENCRYPT |
| VERIFY | `DecryptInit` | DECRYPT_VERIFY |

### 组合状态退出规则

| 组合状态 | 某操作完成 | 剩余状态 |
|---------|---------|---------|
| DIGEST_ENCRYPT | DigestFinal | ENCRYPT |
| DIGEST_ENCRYPT | EncryptFinal | DIGEST |
| DECRYPT_DIGEST | DigestFinal | DECRYPT |
| DECRYPT_DIGEST | DecryptFinal | DIGEST |
| SIGN_ENCRYPT | EncryptFinal | SIGN |
| SIGN_ENCRYPT | SignFinal | ENCRYPT |
| DECRYPT_VERIFY | DecryptFinal | VERIFY |
| DECRYPT_VERIFY | VerifyFinal | DECRYPT |

---

## 五、对象查找（FindObjects）映射

`C_FindObjectsInit` 按 `CKA_CLASS` 属性过滤：

| CKA_CLASS 过滤值 | 返回句柄 | 返回数量 | 备注 |
|----------------|---------|---------|------|
| `CKO_DATA` | Handle=1 × 2 | 2 | 故意返回2个，测试客户端多结果处理 |
| `CKO_SECRET_KEY` | Handle=2 | 1 | — |
| `CKO_PUBLIC_KEY` | Handle=3 | 1 | — |
| `CKO_PRIVATE_KEY` | Handle=4 | 1 | — |
| 无匹配 / 其他条件 | CK_INVALID_HANDLE | 0 | — |

---

## 六、Token 支持的机制列表（9种）

| 机制 | 用途 | 最小密钥位 | 最大密钥位 |
|------|------|-----------|-----------|
| `CKM_RSA_PKCS_KEY_PAIR_GEN` | 生成 RSA 密钥对 | 1024 | 1024 |
| `CKM_RSA_PKCS` | RSA 加密/解密/签名/验签/包装/解包 | 1024 | 1024 |
| `CKM_SHA1_RSA_PKCS` | RSA + SHA1 签名/验签 | 1024 | 1024 |
| `CKM_RSA_PKCS_OAEP` | RSA OAEP 加密/解密 | 1024 | 1024 |
| `CKM_DES3_KEY_GEN` | 生成 3DES 密钥 | 192 | 192 |
| `CKM_DES3_CBC` | 3DES CBC 加密/解密 | 192 | 192 |
| `CKM_SHA_1` | SHA-1 摘要 | — | — |
| `CKM_XOR_BASE_AND_DATA` | 对称密钥派生（XOR） | 128 | 256 |
| `CKM_AES_CBC` | AES CBC 加密/解密 | 128 | 256 |

---

## 七、完整关系网总览

```
Library  "Pikachu CSP PKCS #11 Library"
  └── Slot[1]  "Pikachu Cloud SmartCard Slot"
              └── Token  Label="Pikachu Secure SmartCard"  SN="0123456789A"
              │
              ├── 机制列表（9种）
              │     ├── RSA_PKCS_KEY_PAIR_GEN  →  生成 PublicKey + PrivateKey
              │     ├── RSA_PKCS               →  公钥: 加密/验签/包装  私钥: 解密/签名/解包
              │     ├── SHA1_RSA_PKCS          →  私钥签名 | 公钥验签
              │     ├── RSA_PKCS_OAEP          →  公钥加密 | 私钥解密
              │     ├── DES3_KEY_GEN           →  生成 SecretKey
              │     ├── DES3_CBC               →  对称密钥加密/解密
              │     ├── SHA_1                  →  摘要（含 DigestKey）
              │     ├── XOR_BASE_AND_DATA      →  从 SecretKey 派生新 SecretKey
              │     └── AES_CBC                →  对称密钥加密/解密
              │
              └── Session[1]  (只读/读写 × 公开/用户/SO)
                    ├── Object[1]  CKO_DATA        CKA_LABEL="Pkcs11Interop"  CKA_VALUE="Hello world!"
                    ├── Object[2]  CKO_SECRET_KEY  DES3/AES 加解密、SHA1摘要、XOR派生
                    ├── Object[3]  CKO_PUBLIC_KEY  RSA-1024 加密、验签、包装密钥
                    └── Object[4]  CKO_PRIVATE_KEY RSA-1024 解密、签名、解包密钥
```

> **注意**：当前实现**不包含** `CKO_CERTIFICATE`（证书对象）。若需扩展证书支持，需新增对应句柄及属性处理逻辑。
