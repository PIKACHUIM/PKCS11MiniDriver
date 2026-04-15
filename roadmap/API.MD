# OpenCert REST API 规范

> 最后更新：2026-04-15
> 版本：v1.1.0

---

## 一、OpenCert Manager API（端口 :1026）

### 通用说明

- **基础路径**: `http://127.0.0.1:1026/api`
- **认证方式**: `Authorization: Bearer <token>`（启动时自动生成）
- **响应格式**: JSON
- **成功响应**: `{ "data": ... }`
- **错误响应**: `{ "error": "错误描述" }`

### 1.1 健康检查

```
GET /api/health
```

**响应 200**:
```json
{ "data": { "status": "ok", "version": "1.0.0", "time": "1713100000" } }
```

### 1.2 用户管理

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/users` | 用户列表 |
| POST | `/api/users` | 创建用户 |
| GET | `/api/users/{uuid}` | 用户详情 |
| PUT | `/api/users/{uuid}` | 更新用户 |
| DELETE | `/api/users/{uuid}` | 删除用户 |

**POST /api/users 请求体**:
```json
{
  "user_type": "local",
  "display_name": "张三",
  "email": "zhangsan@example.com",
  "password": "secure_password",
  "role": "user"
}
```

### 1.3 卡片管理

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/cards` | 卡片列表 |
| POST | `/api/cards` | 创建卡片 |
| GET | `/api/cards/{uuid}` | 卡片详情 |
| PUT | `/api/cards/{uuid}` | 更新卡片 |
| DELETE | `/api/cards/{uuid}` | 删除卡片 |

**POST /api/cards 请求体**:
```json
{
  "slot_type": "local",
  "card_name": "我的智能卡",
  "user_uuid": "user-uuid-here",
  "password": "card_password"
}
```

### 1.4 证书管理

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/cards/{card_uuid}/certs` | 证书列表 |
| POST | `/api/cards/{card_uuid}/certs` | 导入证书 |
| GET | `/api/cards/{card_uuid}/certs/{uuid}` | 证书详情 |
| DELETE | `/api/cards/{card_uuid}/certs/{uuid}` | 删除证书 |

### 1.5 密钥操作

```
POST /api/cards/{card_uuid}/keygen
```

**请求体**:
```json
{
  "key_type": "ec256",
  "password": "card_password"
}
```

### 1.6 TOTP 管理

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/cards/{card_uuid}/totp` | TOTP 条目列表 |
| POST | `/api/cards/{card_uuid}/totp` | 添加 TOTP 条目 |
| GET | `/api/totp/{id}/code` | 获取当前验证码 |
| DELETE | `/api/totp/{id}` | 删除 TOTP 条目 |

**POST 请求体**:
```json
{
  "issuer": "GitHub",
  "account": "user@example.com",
  "secret": "JBSWY3DPEHPK3PXP",
  "algorithm": "SHA1",
  "digits": 6,
  "period": 30,
  "otp_type": "totp"
}
```

**GET /api/totp/{id}/code 响应**:
```json
{ "data": { "code": "123456", "remaining": 15, "period": 30 } }
```

### 1.7 本地 PKI 工具

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/pki/selfsign` | 生成自签名证书 |
| POST | `/api/pki/csr` | 生成 CSR |
| POST | `/api/pki/ca` | 创建本地 CA |
| POST | `/api/pki/ca/issue` | 使用本地 CA 签发证书 |
| POST | `/api/pki/convert` | 证书格式转换 |
| POST | `/api/pki/parse` | 解析证书文件 |

**POST /api/pki/selfsign 请求体**:
```json
{
  "card_uuid": "card-uuid",
  "password": "card_password",
  "key_type": "ec256",
  "subject": { "cn": "example.com", "o": "Example Inc", "c": "CN" },
  "validity_days": 365,
  "san": { "dns": ["example.com", "*.example.com"], "ip": ["1.2.3.4"] },
  "key_usage": ["digital_signature", "key_encipherment"],
  "ext_key_usage": ["server_auth", "client_auth"]
}
```

### 1.8 Slot 状态

```
GET /api/slots
```

**响应**:
```json
{ "data": [{ "slot_id": 0, "description": "Local Slot", "token_present": true }] }
```

### 1.9 日志查询

```
GET /api/logs?offset=0&limit=20
```

### 1.10 审计日志

```
GET /api/audit?offset=0&limit=20
```

**响应**:
```json
{
  "data": {
    "logs": [],
    "total": 100,
    "integrity_broken": false
  }
}
```

---

## 二、OpenCert Platform API（端口 :1027）

### 通用说明

- **基础路径**: `https://platform.example.com`
- **认证方式**: `Authorization: Bearer <jwt_token>`
- **响应格式**: JSON
- **成功响应**: `{ "data": ... }` 或 `{ "items": [...], "total": N, "page": 1, "page_size": 20 }`
- **错误响应**: `{ "error": "错误描述" }`
- **权限级别**:
  - `公开` — 无需认证
  - `需认证` — 需要有效 JWT Token
  - `仅 user+` — 需要 user 或 admin 角色
  - `仅 admin` — 需要 admin 角色

---

### 2.1 健康检查

| 方法 | 路径 | 权限 | 说明 |
|------|------|------|------|
| GET | `/api/health` | 公开 | 服务健康状态 |

**响应 200**:
```json
{ "data": { "status": "ok", "version": "1.0.0" } }
```

---

### 2.2 认证接口

| 方法 | 路径 | 权限 | 说明 |
|------|------|------|------|
| POST | `/api/auth/login` | 公开 | 登录，返回 JWT Token |
| POST | `/api/auth/register` | 公开 | 注册新用户 |
| POST | `/api/auth/refresh` | 需认证 | 刷新 Token |
| DELETE | `/api/auth/logout` | 需认证 | 登出（Token 加入黑名单） |
| PUT | `/api/auth/password` | 需认证 | 修改密码 |

**POST /api/auth/login 请求体**:
```json
{
  "username": "admin",
  "password": "secure_password"
}
```

**POST /api/auth/login 响应 200**:
```json
{
  "data": {
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "user_uuid": "550e8400-e29b-41d4-a716-446655440000",
    "username": "admin",
    "role": "admin",
    "expires_at": "2026-04-16T17:00:00Z"
  }
}
```

**POST /api/auth/register 请求体**:
```json
{
  "username": "newuser",
  "password": "secure_password",
  "email": "newuser@example.com",
  "display_name": "新用户"
}
```

**PUT /api/auth/password 请求体**:
```json
{
  "old_password": "old_password",
  "new_password": "new_secure_password"
}
```

**错误码**:
| 状态码 | 说明 |
|--------|------|
| 400 | 请求参数错误 |
| 401 | 用户名或密码错误 / Token 无效 |
| 409 | 用户名已存在 |

---

### 2.3 用户管理接口

| 方法 | 路径 | 权限 | 说明 |
|------|------|------|------|
| GET | `/api/users/me` | 需认证 | 获取当前用户信息 |
| PUT | `/api/users/me` | 需认证 | 更新个人信息 |
| PUT | `/api/users/me/pubkey` | 需认证 | 更新云端公钥 |
| GET | `/api/users` | 仅 admin | 用户列表（分页） |
| PUT | `/api/users/{uuid}/role` | 仅 admin | 修改用户角色 |
| PUT | `/api/users/{uuid}/enabled` | 仅 admin | 启用/禁用用户 |

**GET /api/users/me 响应 200**:
```json
{
  "data": {
    "uuid": "550e8400-e29b-41d4-a716-446655440000",
    "username": "admin",
    "display_name": "管理员",
    "email": "admin@example.com",
    "role": "admin",
    "enabled": true,
    "created_at": "2026-01-01T00:00:00Z",
    "last_login_at": "2026-04-15T10:00:00Z"
  }
}
```

**PUT /api/users/me 请求体**:
```json
{
  "display_name": "新显示名称",
  "email": "newemail@example.com"
}
```

**PUT /api/users/me/pubkey 请求体**:
```json
{
  "public_key_pem": "-----BEGIN PUBLIC KEY-----\n...\n-----END PUBLIC KEY-----"
}
```

**GET /api/users 查询参数**:
| 参数 | 类型 | 说明 |
|------|------|------|
| page | int | 页码（默认 1） |
| page_size | int | 每页数量（默认 20，最大 100） |
| role | string | 按角色筛选（admin/user/readonly） |

**PUT /api/users/{uuid}/role 请求体**:
```json
{ "role": "admin" }
```

**PUT /api/users/{uuid}/enabled 请求体**:
```json
{ "enabled": false }
```

---

### 2.4 卡片与证书接口

| 方法 | 路径 | 权限 | 说明 |
|------|------|------|------|
| GET | `/api/cards` | 需认证 | 卡片列表 |
| POST | `/api/cards` | 仅 user+ | 创建卡片 |
| GET | `/api/cards/{uuid}` | 需认证 | 卡片详情 |
| DELETE | `/api/cards/{uuid}` | 仅 user+ | 删除卡片 |
| GET | `/api/cards/{uuid}/certs` | 需认证 | 证书列表 |
| POST | `/api/cards/{uuid}/certs` | 仅 user+ | 导入证书 |
| DELETE | `/api/cards/{uuid}/certs/{cert_uuid}` | 仅 user+ | 删除证书 |
| POST | `/api/cards/{uuid}/keygen` | 需认证 | 生成密钥对 |
| POST | `/api/cards/{uuid}/sign` | 需认证 | 云端签名 |
| POST | `/api/cards/{uuid}/decrypt` | 需认证 | 云端解密 |
| GET | `/api/certs` | 需认证 | 全局证书筛选查询 |
| POST | `/api/certs/{uuid}/revoke` | 仅 admin | 吊销证书 |
| POST | `/api/certs/{uuid}/assign` | 仅 admin | 分配证书到智能卡 |

**POST /api/cards 请求体**:
```json
{
  "card_name": "我的云端卡",
  "storage_zone_uuid": "zone-uuid",
  "security_level": "high"
}
```

**POST /api/cards/{uuid}/certs 请求体**（X509）:
```json
{
  "cert_type": "x509",
  "cert_pem": "-----BEGIN CERTIFICATE-----\n...\n-----END CERTIFICATE-----"
}
```

**POST /api/cards/{uuid}/keygen 请求体**:
```json
{
  "key_type": "ec256",
  "remark": "用于 TLS"
}
```

**POST /api/cards/{uuid}/sign 请求体**:
```json
{
  "cert_uuid": "cert-uuid",
  "data_base64": "base64编码的待签名数据",
  "algorithm": "SHA256withECDSA"
}
```

**GET /api/certs 查询参数**:
| 参数 | 类型 | 说明 |
|------|------|------|
| ca_uuid | string | 按 CA 筛选 |
| template_uuid | string | 按颁发模板筛选 |
| user_uuid | string | 按用户筛选（admin 可用） |
| card_uuid | string | 按卡片筛选 |
| cert_type | string | 证书类型（x509/gpg/ssh） |
| status | string | 状态（valid/revoked/expired） |
| page | int | 页码 |
| page_size | int | 每页数量 |

**POST /api/certs/{uuid}/assign 请求体**:
```json
{ "card_uuid": "target-card-uuid" }
```

---

### 2.5 CA 管理接口

| 方法 | 路径 | 权限 | 说明 |
|------|------|------|------|
| GET | `/api/cas` | 需认证 | CA 列表 |
| POST | `/api/cas` | 仅 admin | 创建自签名 CA |
| GET | `/api/cas/{uuid}` | 需认证 | CA 详情 |
| PUT | `/api/cas/{uuid}` | 仅 admin | 更新 CA 信息 |
| DELETE | `/api/cas/{uuid}` | 仅 admin | 删除 CA |
| POST | `/api/cas/{uuid}/import-chain` | 仅 admin | 导入证书链 |
| GET | `/api/cas/{uuid}/revoked` | 需认证 | 吊销证书列表 |
| POST | `/api/cas/{uuid}/revoke` | 仅 admin | 吊销证书 |
| GET | `/api/cas/{uuid}/crl` | 公开 | 下载 CRL（DER 格式） |
| POST | `/api/cas/{uuid}/issue` | 仅 admin | 签发证书 |

**POST /api/cas 请求体**:
```json
{
  "name": "根 CA",
  "key_type": "ec384",
  "validity_years": 10,
  "subject": {
    "common_name": "OpenCert Root CA",
    "organization": "Example Corp",
    "country": "CN"
  }
}
```

**GET /api/cas 响应 200**:
```json
{
  "items": [
    {
      "uuid": "ca-uuid",
      "name": "根 CA",
      "status": "active",
      "key_type": "ec384",
      "not_before": "2026-01-01T00:00:00Z",
      "not_after": "2036-01-01T00:00:00Z",
      "issued_count": 42,
      "subject": { "common_name": "OpenCert Root CA" }
    }
  ],
  "total": 1
}
```

**POST /api/cas/{uuid}/import-chain 请求体**:
```json
{
  "chain_pem": "-----BEGIN CERTIFICATE-----\n...\n-----END CERTIFICATE-----\n-----BEGIN CERTIFICATE-----\n...\n-----END CERTIFICATE-----"
}
```

**POST /api/cas/{uuid}/revoke 请求体**:
```json
{
  "serial_hex": "0a1b2c3d4e5f",
  "reason": 1
}
```

> 吊销原因码（RFC 5280）：0=unspecified, 1=keyCompromise, 2=cACompromise, 3=affiliationChanged, 4=superseded, 5=cessationOfOperation

**GET /api/cas/{uuid}/revoked 响应 200**:
```json
{
  "items": [
    {
      "serial_hex": "0a1b2c3d4e5f",
      "revoked_at": "2026-04-15T10:00:00Z",
      "reason": 1,
      "reason_text": "keyCompromise"
    }
  ],
  "total": 1
}
```

**POST /api/cas/{uuid}/issue 请求体**:
```json
{
  "key_type": "ec256",
  "validity_days": 365,
  "subject": {
    "common_name": "example.com",
    "organization": "Example Corp",
    "country": "CN"
  },
  "is_ca": false,
  "san": {
    "dns": ["example.com", "www.example.com"],
    "ip": ["1.2.3.4"],
    "email": ["admin@example.com"]
  }
}
```

---

### 2.6 模板管理接口

#### 2.6.1 证书颁发模板

| 方法 | 路径 | 权限 | 说明 |
|------|------|------|------|
| GET | `/api/templates/issuance` | 需认证 | 颁发模板列表 |
| POST | `/api/templates/issuance` | 仅 admin | 创建颁发模板 |
| GET | `/api/templates/issuance/{uuid}` | 需认证 | 颁发模板详情 |
| DELETE | `/api/templates/issuance/{uuid}` | 仅 admin | 删除颁发模板 |

**POST /api/templates/issuance 请求体**:
```json
{
  "name": "标准 SSL 证书",
  "category": "ssl",
  "is_ca": false,
  "validity_options": [365, 730, 1095],
  "allowed_key_types": ["ec256", "ec384", "rsa2048"],
  "allowed_ca_uuids": ["ca-uuid-1"],
  "subject_template_uuid": "subj-tmpl-uuid",
  "extension_template_uuid": "ext-tmpl-uuid",
  "key_usage_template_uuid": "ku-tmpl-uuid",
  "key_storage_template_uuid": "ks-tmpl-uuid",
  "cert_ext_template_uuid": "ce-tmpl-uuid",
  "price_cents": 9900,
  "stock": -1,
  "enabled": true
}
```

#### 2.6.2 主体模板

| 方法 | 路径 | 权限 | 说明 |
|------|------|------|------|
| GET | `/api/templates/subject` | 需认证 | 主体模板列表 |
| POST | `/api/templates/subject` | 仅 admin | 创建主体模板 |
| DELETE | `/api/templates/subject/{uuid}` | 仅 admin | 删除主体模板 |

**POST /api/templates/subject 请求体**:
```json
{
  "name": "企业主体模板",
  "fields": [
    { "name": "common_name", "required": true, "default_value": "", "max_length": 64 },
    { "name": "organization", "required": true, "default_value": "", "max_length": 64 },
    { "name": "country", "required": false, "default_value": "CN", "max_length": 2 }
  ]
}
```

#### 2.6.3 扩展信息模板

| 方法 | 路径 | 权限 | 说明 |
|------|------|------|------|
| GET | `/api/templates/extension` | 需认证 | 扩展信息模板列表 |
| POST | `/api/templates/extension` | 仅 admin | 创建扩展信息模板 |
| DELETE | `/api/templates/extension/{uuid}` | 仅 admin | 删除扩展信息模板 |

**POST /api/templates/extension 请求体**:
```json
{
  "name": "标准 SAN 模板",
  "max_dns": 10,
  "max_email": 5,
  "max_ip": 5,
  "max_uri": 3,
  "require_verification": true
}
```

#### 2.6.4 密钥用途模板

| 方法 | 路径 | 权限 | 说明 |
|------|------|------|------|
| GET | `/api/templates/key-usage` | 需认证 | 密钥用途模板列表 |
| POST | `/api/templates/key-usage` | 仅 admin | 创建密钥用途模板 |
| DELETE | `/api/templates/key-usage/{uuid}` | 仅 admin | 删除密钥用途模板 |

**POST /api/templates/key-usage 请求体**:
```json
{
  "name": "TLS 服务器",
  "key_usage_bits": 160,
  "ext_key_usages": ["1.3.6.1.5.5.7.3.1", "1.3.6.1.5.5.7.3.2"]
}
```

> 密钥用法位掩码（RFC 5280）：digitalSignature=128, nonRepudiation=64, keyEncipherment=32, dataEncipherment=16, keyAgreement=8, keyCertSign=4, cRLSign=2, encipherOnly=1

#### 2.6.5 证书拓展模板（CRL/OCSP/AIA/CT）

| 方法 | 路径 | 权限 | 说明 |
|------|------|------|------|
| GET | `/api/templates/cert-ext` | 需认证 | 证书拓展模板列表 |
| POST | `/api/templates/cert-ext` | 仅 admin | 创建证书拓展模板 |
| GET | `/api/templates/cert-ext/{uuid}` | 需认证 | 证书拓展模板详情 |
| DELETE | `/api/templates/cert-ext/{uuid}` | 仅 admin | 删除证书拓展模板 |

**POST /api/templates/cert-ext 请求体**:
```json
{
  "name": "标准 PKI 拓展",
  "crl_distribution_points": ["http://crl.example.com/root.crl"],
  "ocsp_servers": ["http://ocsp.example.com"],
  "aia_issuers": ["http://ca.example.com/root.crt"],
  "ct_servers": ["https://ct.googleapis.com/logs/argon2024"],
  "ev_policy_oid": "2.23.140.1.1"
}
```

#### 2.6.6 密钥存储类型模板

| 方法 | 路径 | 权限 | 说明 |
|------|------|------|------|
| GET | `/api/templates/key-storage` | 需认证 | 密钥存储类型模板列表 |
| POST | `/api/templates/key-storage` | 仅 admin | 创建密钥存储类型模板 |
| GET | `/api/templates/key-storage/{uuid}` | 需认证 | 密钥存储类型模板详情 |
| PUT | `/api/templates/key-storage/{uuid}` | 仅 admin | 更新密钥存储类型模板 |
| DELETE | `/api/templates/key-storage/{uuid}` | 仅 admin | 删除密钥存储类型模板 |

**POST /api/templates/key-storage 请求体**:
```json
{
  "name": "高安全性模板",
  "allowed_storage_types": ["virtual_card", "physical_card"],
  "virtual_card_security_level": "high",
  "allow_reimport": false,
  "cloud_backup": false,
  "allow_reissue": false,
  "max_reissue_count": 0
}
```

> `allowed_storage_types` 可选值：`file_download`、`cloud_card`、`physical_card`、`virtual_card`
> `virtual_card_security_level` 可选值：`high`、`medium`、`low`
> 高安全性时 `allow_reimport` 和 `cloud_backup` 强制为 false；`file_download` 时 `cloud_backup` 强制为 true，`max_reissue_count` 强制为 -1（无限）

---

### 2.7 支付系统接口

| 方法 | 路径 | 权限 | 说明 |
|------|------|------|------|
| POST | `/api/payment/recharge` | 需认证 | 发起充值 |
| GET | `/api/payment/orders` | 需认证 | 充值订单列表 |
| GET | `/api/payment/balance` | 需认证 | 查询余额 |
| POST | `/api/payment/refund` | 需认证 | 申请退款 |
| POST | `/api/payment/callback/{channel}` | 公开 | 支付回调（内部验签） |
| GET | `/api/payment/plugins` | 仅 admin | 支付插件列表 |
| POST | `/api/payment/plugins` | 仅 admin | 创建支付插件 |
| DELETE | `/api/payment/plugins/{uuid}` | 仅 admin | 删除支付插件 |
| PUT | `/api/payment/refund/{uuid}/approve` | 仅 admin | 审批退款 |

**POST /api/payment/recharge 请求体**:
```json
{
  "amount_cents": 10000,
  "channel": "alipay"
}
```

**POST /api/payment/recharge 响应 200**:
```json
{
  "data": {
    "order_uuid": "order-uuid",
    "pay_url": "https://openapi.alipay.com/...",
    "qr_code": "base64编码的二维码图片"
  }
}
```

**GET /api/payment/balance 响应 200**:
```json
{
  "data": {
    "balance_cents": 50000,
    "total_recharged_cents": 100000,
    "total_consumed_cents": 50000
  }
}
```

**GET /api/payment/orders 查询参数**:
| 参数 | 类型 | 说明 |
|------|------|------|
| status | string | 状态筛选（pending/paid/failed/refunded） |
| page | int | 页码 |
| page_size | int | 每页数量 |

**POST /api/payment/refund 请求体**:
```json
{
  "order_uuid": "order-uuid",
  "reason": "误充值，申请退款"
}
```

**POST /api/payment/plugins 请求体**:
```json
{
  "name": "支付宝",
  "plugin_type": "alipay",
  "config": {
    "app_id": "2021000000000000",
    "private_key": "***",
    "alipay_public_key": "***"
  },
  "enabled": true,
  "sort_weight": 1
}
```

---

### 2.8 云端 TOTP 接口

| 方法 | 路径 | 权限 | 说明 |
|------|------|------|------|
| GET | `/api/totp` | 需认证 | 云端 TOTP 列表 |
| POST | `/api/totp` | 仅 user+ | 添加云端 TOTP |
| GET | `/api/totp/{uuid}/code` | 需认证 | 获取当前验证码 |
| DELETE | `/api/totp/{uuid}` | 仅 user+ | 删除云端 TOTP |

**POST /api/totp 请求体**:
```json
{
  "issuer": "GitHub",
  "account": "user@example.com",
  "secret": "JBSWY3DPEHPK3PXP",
  "algorithm": "SHA1",
  "digits": 6,
  "period": 30
}
```

**GET /api/totp/{uuid}/code 响应 200**:
```json
{
  "data": {
    "code": "123456",
    "remaining": 15,
    "period": 30
  }
}
```

---

### 2.9 主体信息接口

| 方法 | 路径 | 权限 | 说明 |
|------|------|------|------|
| GET | `/api/subject-infos` | 需认证 | 主体信息列表 |
| POST | `/api/subject-infos` | 仅 user+ | 创建主体信息 |
| PUT | `/api/subject-infos/{uuid}/approve` | 仅 admin | 审核通过 |
| PUT | `/api/subject-infos/{uuid}/reject` | 仅 admin | 审核拒绝 |

**POST /api/subject-infos 请求体**:
```json
{
  "subject_template_uuid": "tmpl-uuid",
  "fields": {
    "common_name": "Example Corp",
    "organization": "Example Corp Ltd",
    "country": "CN"
  }
}
```

**GET /api/subject-infos 响应 200**:
```json
{
  "items": [
    {
      "uuid": "subj-info-uuid",
      "subject_template_uuid": "tmpl-uuid",
      "fields": { "common_name": "Example Corp" },
      "status": "approved",
      "created_at": "2026-04-01T00:00:00Z"
    }
  ],
  "total": 1
}
```

---

### 2.10 扩展信息验证接口

| 方法 | 路径 | 权限 | 说明 |
|------|------|------|------|
| GET | `/api/extension-infos` | 需认证 | 扩展信息列表 |
| POST | `/api/extension-infos` | 仅 user+ | 创建扩展信息（域名/邮箱/IP） |
| POST | `/api/extension-infos/{uuid}/verify-dns` | 需认证 | 触发 DNS TXT 验证 |
| POST | `/api/extension-infos/{uuid}/verify-email` | 需认证 | 提交邮箱验证码 |
| DELETE | `/api/extension-infos/{uuid}` | 仅 user+ | 删除扩展信息 |

**POST /api/extension-infos 请求体**（域名）:
```json
{
  "type": "domain",
  "value": "example.com"
}
```

**POST /api/extension-infos 响应 200**（域名，返回验证 Token）:
```json
{
  "data": {
    "uuid": "ext-info-uuid",
    "type": "domain",
    "value": "example.com",
    "verify_token": "opencert-verify=abc123xyz",
    "dns_record": "_opencert.example.com TXT opencert-verify=abc123xyz",
    "status": "pending"
  }
}
```

**POST /api/extension-infos/{uuid}/verify-email 请求体**:
```json
{ "code": "123456" }
```

**GET /api/extension-infos 响应 200**:
```json
{
  "items": [
    {
      "uuid": "ext-info-uuid",
      "type": "domain",
      "value": "example.com",
      "status": "verified",
      "verified_at": "2026-04-10T00:00:00Z",
      "expires_at": "2027-04-10T00:00:00Z"
    }
  ],
  "total": 1
}
```

---

### 2.11 证书订单与申请接口

| 方法 | 路径 | 权限 | 说明 |
|------|------|------|------|
| POST | `/api/cert-orders` | 仅 user+ | 创建证书订单 |
| GET | `/api/cert-orders` | 需认证 | 证书订单列表 |
| POST | `/api/cert-applications` | 仅 user+ | 提交证书申请 |
| GET | `/api/cert-applications` | 需认证 | 证书申请列表 |
| PUT | `/api/cert-applications/{uuid}/approve` | 仅 admin | 审批通过 |
| PUT | `/api/cert-applications/{uuid}/reject` | 仅 admin | 审批拒绝 |

**POST /api/cert-orders 请求体**:
```json
{
  "issuance_template_uuid": "tmpl-uuid",
  "validity_days": 365,
  "key_type": "ec256"
}
```

**POST /api/cert-orders 响应 200**:
```json
{
  "data": {
    "uuid": "order-uuid",
    "issuance_template_uuid": "tmpl-uuid",
    "amount_cents": 9900,
    "status": "pending",
    "created_at": "2026-04-15T10:00:00Z"
  }
}
```

**POST /api/cert-applications 请求体**:
```json
{
  "cert_order_uuid": "order-uuid",
  "subject_info_uuid": "subj-info-uuid",
  "extension_info_uuids": ["ext-info-uuid-1", "ext-info-uuid-2"],
  "key_type": "ec256"
}
```

**PUT /api/cert-applications/{uuid}/reject 请求体**:
```json
{ "reason": "主体信息不符合要求" }
```

**GET /api/cert-applications 查询参数**:
| 参数 | 类型 | 说明 |
|------|------|------|
| status | string | 状态筛选（pending/approved/rejected） |
| user_uuid | string | 按用户筛选（admin 可用） |
| page | int | 页码 |
| page_size | int | 每页数量 |

---

### 2.12 存储区域管理接口

| 方法 | 路径 | 权限 | 说明 |
|------|------|------|------|
| GET | `/api/storage-zones` | 需认证 | 存储区域列表 |
| POST | `/api/storage-zones` | 仅 admin | 创建存储区域 |
| DELETE | `/api/storage-zones/{uuid}` | 仅 admin | 删除存储区域 |

**POST /api/storage-zones 请求体**:
```json
{
  "name": "主数据库存储",
  "storage_type": "database",
  "hsm_driver": ""
}
```

> `storage_type` 可选值：`database`、`hsm`；`hsm_driver` 在 `storage_type=hsm` 时必填

**GET /api/storage-zones 响应 200**:
```json
{
  "items": [
    {
      "uuid": "zone-uuid",
      "name": "主数据库存储",
      "storage_type": "database",
      "status": "active",
      "created_at": "2026-01-01T00:00:00Z"
    }
  ],
  "total": 1
}
```

---

### 2.13 OID 管理接口

| 方法 | 路径 | 权限 | 说明 |
|------|------|------|------|
| GET | `/api/oids` | 需认证 | OID 列表 |
| POST | `/api/oids` | 仅 admin | 创建 OID |
| DELETE | `/api/oids/{uuid}` | 仅 admin | 删除 OID |

**POST /api/oids 请求体**:
```json
{
  "oid": "1.3.6.1.4.1.99999.1",
  "name": "自定义扩展密钥用途",
  "description": "用于内部系统认证",
  "usage_type": "ext_key_usage"
}
```

> `usage_type` 可选值：`ext_key_usage`、`subject_field`、`ev_policy`、`asn1_extension`

---

### 2.14 吊销服务配置接口

| 方法 | 路径 | 权限 | 说明 |
|------|------|------|------|
| GET | `/api/revocation-services` | 仅 admin | 吊销服务列表 |
| POST | `/api/revocation-services` | 仅 admin | 创建吊销服务 |
| DELETE | `/api/revocation-services/{uuid}` | 仅 admin | 删除吊销服务 |

**POST /api/revocation-services 请求体**:
```json
{
  "ca_uuid": "ca-uuid",
  "service_type": "crl",
  "path": "/crl/root.crl",
  "enabled": true,
  "crl_update_interval_minutes": 60
}
```

> `service_type` 可选值：`crl`、`ocsp`、`caissuer`

---

### 2.15 ACME 配置接口

| 方法 | 路径 | 权限 | 说明 |
|------|------|------|------|
| GET | `/api/acme-configs` | 仅 admin | ACME 配置列表 |
| POST | `/api/acme-configs` | 仅 admin | 创建 ACME 配置 |
| DELETE | `/api/acme-configs/{uuid}` | 仅 admin | 删除 ACME 配置 |

**POST /api/acme-configs 请求体**:
```json
{
  "path": "default",
  "ca_uuid": "ca-uuid",
  "issuance_template_uuid": "tmpl-uuid",
  "enabled": true
}
```

> ACME 目录访问路径为 `GET /acme/{path}/directory`

---

### 2.16 CT 记录管理接口

| 方法 | 路径 | 权限 | 说明 |
|------|------|------|------|
| GET | `/api/ct-entries` | 需认证 | CT 记录列表 |
| DELETE | `/api/ct-entries/{uuid}` | 仅 admin | 删除 CT 记录 |

**GET /api/ct-entries 查询参数**:
| 参数 | 类型 | 说明 |
|------|------|------|
| cert_hash | string | 按证书哈希筛选 |
| cert_uuid | string | 按证书 UUID 筛选 |
| page | int | 页码 |
| page_size | int | 每页数量 |

**GET /api/ct-entries 响应 200**:
```json
{
  "items": [
    {
      "uuid": "ct-entry-uuid",
      "cert_uuid": "cert-uuid",
      "cert_hash": "sha256:abc123...",
      "ct_server": "https://ct.googleapis.com/logs/argon2024",
      "sct_data": "base64编码的SCT",
      "submitted_at": "2026-04-15T10:00:00Z"
    }
  ],
  "total": 1
}
```

---

### 2.17 公开服务接口

| 方法 | 路径 | 权限 | 说明 |
|------|------|------|------|
| GET | `/crl/{caUUID}` | 公开 | CRL 文件下载（DER 格式） |
| POST | `/ocsp/{caUUID}` | 公开 | OCSP 状态查询 |
| GET | `/ca/{caUUID}` | 公开 | CA 证书 PEM（AIA CAIssuer） |
| GET | `/acme/{path}/directory` | 公开 | ACME 目录 |
| HEAD | `/acme/{path}/new-nonce` | 公开 | ACME 新 Nonce |
| POST | `/ct/submit` | 公开 | CT 证书提交 |
| GET | `/ct/query` | 公开 | CT 按哈希查询 |

**GET /crl/{caUUID}**:
- 响应 Content-Type: `application/pkix-crl`
- 响应体：DER 格式的 CRL 二进制数据

**POST /ocsp/{caUUID}**:
- 请求 Content-Type: `application/ocsp-request`
- 响应 Content-Type: `application/ocsp-response`
- 查询参数：`?serial=0a1b2c3d4e5f`（十六进制序列号）

**GET /ca/{caUUID}**:
- 响应 Content-Type: `application/x-pem-file`
- 响应体：PEM 格式的 CA 证书

**GET /acme/{path}/directory 响应 200**:
```json
{
  "newNonce": "https://platform.example.com/acme/default/new-nonce",
  "newAccount": "https://platform.example.com/acme/default/new-account",
  "newOrder": "https://platform.example.com/acme/default/new-order",
  "revokeCert": "https://platform.example.com/acme/default/revoke-cert",
  "keyChange": "https://platform.example.com/acme/default/key-change"
}
```

**POST /ct/submit 请求体**:
```json
{
  "cert_pem": "-----BEGIN CERTIFICATE-----\n...\n-----END CERTIFICATE-----",
  "chain_pem": "-----BEGIN CERTIFICATE-----\n...\n-----END CERTIFICATE-----"
}
```

**GET /ct/query 查询参数**:
| 参数 | 类型 | 说明 |
|------|------|------|
| cert_hash | string | 证书 SHA-256 哈希（十六进制） |

---

## 三、IPC 协议（OpenCert Driver ↔ Manager）

### 帧格式

```
+--------+--------+--------+------------------+
| Magic  |  Cmd   |  Len   |    JSON Payload  |
| 4 Byte | 4 Byte | 4 Byte |    Len Bytes     |
+--------+--------+--------+------------------+
  "PK11"   BigEndian  BigEndian
```

### 命令码

| 命令码 | 名称 | 说明 |
|--------|------|------|
| 0x0000 | CmdPing | 心跳 |
| 0x0001 | CmdGetInfo | 获取库信息 |
| 0x0002-0x0006 | Slot/Token/Mechanism | Slot 和 Token 信息查询 |
| 0x0007-0x0009 | Session | 会话管理 |
| 0x000A-0x000B | Login/Logout | 认证 |
| 0x000C-0x000E | FindObjects | 对象查找 |
| 0x000F-0x0013 | Object | 对象属性和管理 |
| 0x0014-0x001D | Crypto | 签名/解密/加密 |
| 0x001E-0x0021 | KeyGen/Random/Digest | 密钥生成和摘要 |
| 0x0022 | CmdGetSessionInfo | 会话信息 |
| 0x0023-0x0024 | PIN | PIN 管理 |
| 0x00FF | CmdHandshake | 版本协商 |
