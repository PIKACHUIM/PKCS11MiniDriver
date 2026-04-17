# OpenCert Manager 云端平台 · 第三轮补全任务清单

> 基于 [02-EVALUATION-ROUND3.md](./02-EVALUATION-ROUND3.md) 评估报告生成
> 评估结论：后端 85% / 前端 35% / 业务逻辑 55% / 综合 ~58%
> 目标：2-3 周内达到 90%+ 需求覆盖度

---

## 📐 阶段划分

```
Phase 1（约 1 周）· 阻塞用户使用的问题 [P0]
  ├─ T1  侧边栏菜单重构 + 12 个前端页面补全
  ├─ T2  证书申请审批自动签发（断链修复）
  ├─ T3  签发引擎：读取 CertExtTmplUUID 写入扩展
  ├─ T4  签发引擎：模板约束验证（KU/EKU/有效期/CA）
  ├─ T5  PIN 会话令牌机制 + 签名/解密强制 PIN
  ├─ T6  CA 导入外部 CA（CertPEM + PrivateKey PEM）
  └─ T7  证书链查询 API（/api/cas/{uuid}/chain）

Phase 2（约 1 周）· 功能完整性 [P1]
  ├─ T8   主体预置字段 API（dn.txt）
  ├─ T9   OID 预置库 API（oids.txt 60+ 条）
  ├─ T10  邮箱验证真实发送（SMTP 集成）
  ├─ T11  扩展信息模板绑定 + 按模板有效期判定
  ├─ T12  CT 真实提交 RFC 6962 add-chain
  ├─ T13  用户自主绑定登录 TOTP API
  ├─ T14  Netscape/CSP/自定义 ASN.1 扩展写入
  ├─ T15  权限检查散弹清理（统一 IsAdmin/IsOperator）
  └─ T16  OCSP 标准 binary 响应（RFC 6960）

Phase 3（按需）· 算法与合规扩展 [P2/P3]
  ├─ T17  SM2/SM3/SM4 国密算法
  ├─ T18  Ed25519/X25519/brainpool
  ├─ T19  RSA 1024-8192 全范围 + SHA3/MD5
  ├─ T20  CRL 按 CRLInterval 独立调度
  ├─ T21  ImportChain SQL 兼容性修复
  ├─ T22  ACME 挑战真实验证 + Finalize 签发
  ├─ T23  数据模型补字段（URI SAN、verify_expires_days 等）
  └─ T24  数据库表 pin_sessions / cert_chains
```

---

## 🎯 Phase 1 · P0 阻塞性问题（7 项）

### T1 · 侧边栏菜单重构 + 12 个前端页面补全

**需求映射**：#2 #3 #4 #13 #14 #16 #17 #18 #19 #20 #21 #22（12 项）

**核心工作**：
- 重构 `servers/front/src/layouts/MainLayout.tsx` 侧边栏分组
- 新增 12 个前端页面（每页 ~300-500 行）

**侧边栏新结构**（分 4 大组）：
```
🏠 我的工作台
  ├─ 平台概览 (Dashboard)
  └─ 个人中心 (Profile)

💳 证书与身份
  ├─ 云端智能卡 (Cards)           ★ 新增
  ├─ 我的证书 (Certs)
  ├─ 主体信息 (SubjectInfos)      ★ 新增
  ├─ 扩展信息 (ExtensionInfos)    ★ 新增
  ├─ 云端 TOTP (CloudTOTP)        ★ 新增
  └─ 个人身份 (Identity)

🛒 订单与支付
  ├─ 证书申请 (CertOrders)
  ├─ 充值管理 (Payment)
  └─ 退款审批 (RefundApproval)    ★ 新增（管理员）

⚙️ 平台管理（仅管理员）
  ├─ CA 管理 (CA)
  ├─ 全局证书 (AllCerts)          ★ 新增
  ├─ 证书颁发模板 (Templates)
  ├─ 证书申请模板 (CertApplyTemplates)
  ├─ 密钥存储模板 (KeyStorageTemplates) ★ 新增
  ├─ OID 管理 (OIDs)              ★ 新增
  ├─ 存储区域 (StorageZones)      ★ 新增
  ├─ 吊销服务 (RevocationServices) ★ 新增
  ├─ ACME 服务 (ACMEConfigs)      ★ 新增
  ├─ CT 记录 (CTRecords)
  ├─ 证书申请审核 (CertApplications) ★ 新增
  ├─ 用户管理 (Users)
  ├─ 支付插件 (PaymentPlugins)    ★ 新增
  ├─ 审计日志 (AuditLogs)
  └─ 系统设置 (Settings)
```

**交付清单**：
- [ ] `pages/Cards/index.tsx` - 云端智能卡管理（卡片列表+CRUD+PIN/PUK/AdminKey）
- [ ] `pages/AllCerts/index.tsx` - 全局证书管理（筛选+吊销+分配+续期+导出）
- [ ] `pages/SubjectInfos/index.tsx` - 主体信息管理（提交+审核状态展示）
- [ ] `pages/ExtensionInfos/index.tsx` - 扩展信息管理（DNS/邮箱/HTTP 验证）
- [ ] `pages/CloudTOTP/index.tsx` - 云端 TOTP 管理（添加+查看验证码）
- [ ] `pages/KeyStorageTemplates/index.tsx` - 密钥存储类型模板 CRUD
- [ ] `pages/OIDs/index.tsx` - OID 管理 CRUD（含 ASN.1 类型选择）
- [ ] `pages/StorageZones/index.tsx` - 存储区域管理
- [ ] `pages/RevocationServices/index.tsx` - 吊销服务管理（CRL/OCSP/CAIssuer 路径配置）
- [ ] `pages/ACMEConfigs/index.tsx` - ACME 服务配置
- [ ] `pages/CertApplications/index.tsx` - 证书申请审核（管理员视图）
- [ ] `pages/PaymentPlugins/index.tsx` - 支付插件管理
- [ ] `pages/RefundApproval/index.tsx` - 退款审批页
- [ ] `App.tsx` 路由补齐
- [ ] `MainLayout.tsx` 侧边栏重构

**估算**：2-3 天

---

### T2 · 证书申请审批自动签发（断链修复）

**需求映射**：#18

**问题**：`workflow.ApproveApplication` 只改状态，不签发证书。

**修改文件**：
- `internal/workflow/service.go` - `ApproveApplication` 方法
- `internal/api/handler_workflow.go` - 注入 `caSvc`

**实现要点**：
```go
// 改造后的 ApproveApplication 流程：
// 1. 校验申请状态为 pending
// 2. 查询关联订单 → 获取 IssuanceTmplUUID / CertApplyTmplUUID
// 3. 从模板读取 CA、有效期、密钥类型等参数
// 4. 从 SubjectJSON / SANJSON 还原签发参数
// 5. 调用 caSvc.IssueCert(ctx, &ca.IssueRequest{...})
// 6. 创建 Certificate 记录（含完整元数据）
// 7. 更新 cert_applications.cert_uuid
// 8. 更新 cert_orders.status = completed
// 9. 冻结金额转消费记录
// 10. 写审计日志 audit_logs
```

**估算**：1 天

---

### T3 · 签发引擎：读取 CertExtTmplUUID 写入扩展

**需求映射**：#10 #11

**修改文件**：
- `internal/ca/issuer.go` - `IssueCert` 方法
- `internal/api/handler_ca.go` - `handleIssueCert`

**实现要点**：
- 在 `handleIssueCert` 中，若提供 `issuance_tmpl_uuid`：
  1. 读取 `IssuanceTemplate` → 提取 `CertExtTmplUUID`
  2. 读取 `CertExtTemplate` → 解析 CRL/OCSP/AIA/EV 字段
  3. 写入 `IssueRequest.CRLDistPoints` / `OCSPServers` / `AIAIssuers` / `EVPolicyOID`
- `issuer.go` 中 `IssueCert` 将 `EVPolicyOID` 写入 `template.PolicyIdentifiers`（`asn1.ObjectIdentifier`）

**估算**：0.5 天

---

### T4 · 签发引擎：模板约束验证

**需求映射**：#6 #9 #11

**问题**：
- 有效期不验证是否在 `IssuanceTemplate.ValidDays` 列表中
- 密钥类型不验证是否在 `AllowedKeyTypes` 中
- CA 不验证是否在 `AllowedCAUUIDs` 中
- EKU 强制 ServerAuth+ClientAuth，忽略 `KeyUsageTemplate`

**修改文件**：
- `internal/ca/issuer.go` - `IssueCert` 增加模板约束检查
- `internal/api/handler_ca.go` - `handleIssueCert` 从 `KeyUsageTemplate` 读取 KU/EKU

**实现要点**：
```go
// IssueCert 开头增加：
if req.IssuanceTmplUUID != "" {
    tmpl, _ := s.issuanceSvc.GetIssuanceTemplate(ctx, req.IssuanceTmplUUID)
    if !containsInt(tmpl.ValidDays, req.ValidDays) {
        return nil, fmt.Errorf("有效期 %d 不在模板允许列表", req.ValidDays)
    }
    if !containsStr(tmpl.AllowedKeyTypes, req.KeyType) {
        return nil, fmt.Errorf("密钥类型 %s 不在模板允许列表", req.KeyType)
    }
    if len(tmpl.AllowedCAUUIDs) > 0 && !containsStr(tmpl.AllowedCAUUIDs, req.CAUUID) {
        return nil, fmt.Errorf("CA 不在模板允许列表")
    }
}
```

**估算**：0.5 天

---

### T5 · PIN 会话令牌机制 + 签名/解密强制 PIN

**需求映射**：用户补充需求 #1 #2

**问题**：`Sign`/`Decrypt`/`GenerateKeyPair`/`DeleteCert`/`ImportCert` 完全不验证 PIN。

**设计**：
```
┌─────────────────────────────────────────────────┐
│  PIN 会话模型                                      │
│  1. POST /api/cards/{uuid}/verify-pin            │
│     → 返回 pin_session_token (有效期 15 分钟)     │
│  2. 签名/解密等敏感操作：                           │
│     Header: X-PIN-Session: <token>               │
│  3. 会话过期 → 返回 401 requires_pin              │
└─────────────────────────────────────────────────┘
```

**修改内容**：
- 新增 `pin_sessions` 表（uuid/card_uuid/user_uuid/token_hash/expires_at）
- 新增 `internal/card/pin_session.go`
- `handleVerifyPIN` 成功后返回 `pin_session_token`
- `handleSign`/`handleDecrypt`/`handleKeyGen`/`handleImportCert`/`handleDeleteCert`：从 Header 读取 `X-PIN-Session`，校验有效
- 中间件 `requirePINSession(next)`

**估算**：1 天

---

### T6 · CA 导入外部 CA

**需求映射**：#5

**问题**：`handleCreateCA` 只支持自签名，无法导入外部 CA。

**新增接口**：`POST /api/cas/import`
```json
{
  "name": "Let's Encrypt R3",
  "cert_pem": "-----BEGIN CERTIFICATE-----...",
  "private_key_pem": "-----BEGIN PRIVATE KEY-----...",
  "chain_pem": "-----BEGIN CERTIFICATE-----...（可选）"
}
```

**修改文件**：
- `internal/ca/service.go` - 新增 `ImportCA(ctx, req)` 方法
- `internal/api/handler_ca.go` - 新增 `handleImportCA`
- `internal/api/server.go` - 注册路由

**安全校验**：
- 验证私钥与证书匹配（`x509.Certificate.CheckSignature`）
- 验证证书为 CA 证书（`IsCA = true`）
- 加密存储私钥（`encryptPrivateKey`）

**估算**：0.5 天

---

### T7 · 证书链查询 API

**需求映射**：#5

**新增接口**：
- `GET /api/cas/{uuid}/chain` - 返回 CA 的完整证书链（PEM）
- `GET /api/certs/{uuid}/chain` - 返回证书 + 其签发 CA 链

**修改文件**：
- `internal/ca/service.go` - 新增 `GetChain(ctx, caUUID)` 递归查 parent
- `internal/api/handler_ca.go` - 新增 `handleGetCAChain` / `handleGetCertChain`
- `internal/api/server.go` - 注册路由

**估算**：0.5 天

---

## 🎯 Phase 2 · P1 功能完整性（9 项）

### T8 · 主体预置字段 API（dn.txt）

**需求映射**：#7

**新增接口**：`GET /api/meta/subject-fields`

**返回数据**（基于 [dn.txt](./ca/dn.txt)）：
```json
{
  "fields": [
    {"name": "C", "display": "国家", "max_length": 2, "pattern": "^[A-Z]{2}$"},
    {"name": "ST", "display": "省/州", "max_length": 128},
    {"name": "L", "display": "城市", "max_length": 128},
    {"name": "O", "display": "组织", "max_length": 64},
    {"name": "OU", "display": "部门", "max_length": 64},
    {"name": "CN", "display": "通用名", "max_length": 64, "required": true},
    {"name": "emailAddress", "display": "邮箱", "max_length": 128},
    {"name": "serialNumber", "display": "序列号"},
    {"name": "givenName", "display": "名"},
    {"name": "surname", "display": "姓"},
    // ... 共 25 个字段
  ]
}
```

**实现**：
- 将 `roadmap/ca/dn.txt` 解析为 Go slice（内置静态数据）
- 新文件 `internal/meta/subject_fields.go`

**估算**：0.5 天

---

### T9 · OID 预置库 API（oids.txt）

**需求映射**：#9 #13

**新增接口**：`GET /api/meta/predefined-oids?category=eku`

**返回数据**（基于 [oids.txt](./ca/oids.txt)）：
```json
{
  "categories": {
    "ssl": [
      {"oid": "1.3.6.1.5.5.7.3.1", "short": "serverAuth", "name": "[SSL] Server Auth"},
      {"oid": "1.3.6.1.5.5.7.3.2", "short": "clientAuth", "name": "[SSL] Client Auth"}
    ],
    "code_sign": [...],
    "email": [...],
    "ipsec": [...],
    "ssh": [...],
    "ms_ca": [...],
    "ev": [...]
    // 共 60+ 个 OID
  }
}
```

**实现**：
- 将 `oids.txt` 解析为分类 Map（按注释行 `# Code Sign ---` 划分）
- 新文件 `internal/meta/predefined_oids.go`
- 前端 `KeyUsageTemplates` / `OIDs` 页面下拉框引用此 API

**估算**：0.5 天

---

### T10 · 邮箱验证真实发送（SMTP 集成）

**需求映射**：#8 #17

**问题**：当前验证码 = token 前 6 位，用户无法获知。

**修改内容**：
- `configs/config.go` - 新增 SMTP 配置（host/port/user/password/from）
- `internal/mailer/` - 新建邮件发送模块（`net/smtp`）
- `internal/verification/service.go`：
  - 生成 6 位数字验证码 + 独立 `verify_code_hash` 字段
  - `CreateExtensionInfo` 对 `info_type=email` 自动发邮件
  - `VerifyEmailCode` 对比 hash 而非 token 前缀

**模型变更**：
- `ExtensionInfo` 新增 `VerifyCodeHash string`
- 数据库迁移：`ALTER TABLE extension_infos ADD COLUMN verify_code_hash TEXT`

**估算**：1 天

---

### T11 · 扩展信息模板绑定 + 按模板有效期判定

**需求映射**：#8 #17

**修改内容**：
- `ExtensionTemplate` 新增 `VerifyExpiresDays int`（默认 90）
- `ExtensionInfo` 新增 `TmplUUID string`（关联模板）
- `CreateExtensionInfo` 接受 `tmpl_uuid` 参数 + 校验 `RequireDNSVerify` 等约束
- `markVerified` 按 `tmpl.VerifyExpiresDays` 计算 `expires_at`

**数据库迁移**：
```sql
ALTER TABLE extension_templates ADD COLUMN verify_expires_days INTEGER DEFAULT 90;
ALTER TABLE extension_infos ADD COLUMN tmpl_uuid TEXT;
```

**估算**：0.5 天

---

### T12 · CT 真实提交 RFC 6962 add-chain

**需求映射**：#15

**问题**：当前 TODO 未实现，SCTData 永远为空。

**修改文件**：`internal/ct/service.go` - `Submit` 方法

**实现**：
```go
// RFC 6962 add-chain 请求
// POST https://<ct-log>/ct/v1/add-chain
// {"chain": ["<base64-cert-der>", "<base64-intermediate-der>"]}
// 响应：{"sct_version":0,"id":"...","timestamp":...,"signature":"..."}

func (s *Service) Submit(ctx context.Context, certDER []byte, chainDER [][]byte, ctServer string) (*CTEntry, error) {
    // 1. 构造 add-chain 请求
    // 2. HTTP POST 到 ctServer + /ct/v1/add-chain
    // 3. 解析响应 → 存入 sct_data (JSON 序列化)
    // 4. status = "submitted" 或 "failed"
}
```

**CT 查询认证**：
- `handleCTSubmit` 读取 `Authorization: Bearer <ct_token>`
- `configs/config.go` 新增 `CTSubmitToken`

**估算**：1 天

---

### T13 · 用户自主绑定登录 TOTP API

**需求映射**：#1 #20

**新增接口**：
- `POST /api/auth/totp/generate` - 生成随机 secret + 返回 otpauth:// URI
- `POST /api/auth/totp/bind` - 提交验证码开启 TOTP 保护
- `DELETE /api/auth/totp/unbind` - 解除 TOTP 绑定（需当前密码）
- `GET /api/auth/totp/status` - 查询当前用户是否已开启

**修改文件**：`internal/api/handler_auth.go`

**数据库**：复用现有 `users.totp_secret` 字段（AES-GCM 加密存储）

**估算**：0.5 天

---

### T14 · Netscape/CSP/自定义 ASN.1 扩展写入

**需求映射**：#10

**修改文件**：`internal/ca/issuer.go` - `IssueCert` 新增扩展写入逻辑

**实现**：
```go
// Netscape Cert Type (2.16.840.1.113730.1.1)
if tmpl.NetscapeConfig != "" {
    var cfg struct { CertType int `json:"cert_type"` }
    json.Unmarshal([]byte(tmpl.NetscapeConfig), &cfg)
    template.ExtraExtensions = append(template.ExtraExtensions, pkix.Extension{
        Id: asn1.ObjectIdentifier{2,16,840,1,113730,1,1},
        Value: mustMarshal(asn1.BitString{Bytes: []byte{byte(cfg.CertType)}, BitLength: 8}),
    })
}

// 自定义 ASN.1 扩展
for _, ext := range tmpl.ASN1Extensions {
    template.ExtraExtensions = append(template.ExtraExtensions, pkix.Extension{
        Id: parseOID(ext.OID),
        Critical: ext.Critical,
        Value: encodeASN1(ext.Type, ext.Value), // 按 OID.ASN1Type 编码
    })
}
```

**估算**：1 天

---

### T15 · 权限检查散弹清理

**问题**：15+ 处硬编码 `claims.Role == "admin"`，未考虑 `super_admin`/`operator`。

**修改内容**：
- 统一用 `auth.IsAdmin(role)` / `auth.IsOperatorOrAbove(role)`
- 批量替换涉及文件：`handler_verification.go` / `handler_card.go` / `handler_workflow.go` / `handler_totp.go` / `handler_ca.go`

**估算**：0.5 天

---

### T16 · OCSP 标准 binary 响应（RFC 6960）

**问题**：当前返回自定义 JSON，不符合 RFC 6960。

**修改文件**：`internal/revocation/service.go`

**实现**：
- 使用 `golang.org/x/crypto/ocsp` 包
- `QueryOCSPStatus` 返回标准 `OCSPResponse DER`
- HTTP 请求解析 OCSP request body（POST application/ocsp-request）
- 响应 Content-Type: `application/ocsp-response`

**估算**：1 天

---

## 🎯 Phase 3 · P2/P3 扩展与优化（8 项）

### T17 · SM2/SM3/SM4 国密算法

**依赖**：`github.com/tjfoc/gmsm`
**估算**：1.5 天

### T18 · Ed25519/X25519/brainpool

**依赖**：`filippo.io/edwards25519`、`github.com/ebfe/brainpool`
**估算**：1 天

### T19 · RSA 1024-8192 全范围 + SHA3/MD5

**估算**：0.5 天

### T20 · CRL 按 CRLInterval 独立调度

**修改文件**：`internal/revocation/service.go`
**实现**：每个 CA 独立 goroutine + ticker(CRLInterval)
**估算**：0.5 天

### T21 · ImportChain SQL 兼容性修复

**问题**：`cert_pem || chain_pem` 在 MySQL/PostgreSQL 语义不同
**方案**：改为应用层 `SELECT cert_pem → 拼接 → UPDATE`
**估算**：0.3 天

### T22 · ACME 挑战真实验证 + Finalize 签发

**修改文件**：`internal/acme/service.go`
**实现**：
- HTTP-01：GET `http://<domain>/.well-known/acme-challenge/<token>` 比对响应
- DNS-01：LookupTXT `_acme-challenge.<domain>` 比对 SHA256(token+thumbprint)
- Finalize：接收 CSR，调用 `caSvc.IssueCert`，返回证书 URL
**估算**：2 天

### T23 · 数据模型补字段

- `ExtensionTemplate`：URI/RID/Other SAN 类型 + `verify_expires_days`
- `Certificate`：`san_uris` / `certificate_policies`
- `CustomOID`：`is_critical`
**估算**：0.5 天

### T24 · 数据库表 pin_sessions / cert_chains

**已在 T5 中包含 pin_sessions**
**新增 cert_chains 视图**（根据 parent_uuid 递归）
**估算**：0.3 天

---

## 📅 时间总表

| 阶段 | 任务数 | 工作量估算 | 优先级 |
|:----:|:-----:|:---------:|:-----:|
| Phase 1 | 7 项 | ~7 天 | P0 |
| Phase 2 | 9 项 | ~6.5 天 | P1 |
| Phase 3 | 8 项 | ~7.6 天 | P2/P3 |
| **合计** | **24 项** | **~21 天** | - |

---

## ✅ 开发纪律

1. **每个任务独立提交**：小步提交，每次编译通过、现有测试通过
2. **渐进式开发**：研究现有代码 → 规划 → 实现 → 测试
3. **最小化修改**：只改相关模块，不扩散到其他代码
4. **遵循现有规范**：`go fmt` / `go vet` / 用现有工具链
5. **每阶段结束统一编译验证**：`go build ./... && go vet ./...`

---

## 🔄 与 Phase 1/2 交叉依赖

```
T1（前端页面） ──需要──> T2/T3/T4/T5/T6/T7 的后端接口
T5（PIN 会话） ──影响──> T1 的 Cards/AllCerts 页面交互
T8/T9（预置数据） ──影响──> T1 的 SubjectInfos/OIDs 页面
T11（扩展模板） ──影响──> T1 的 ExtensionInfos 页面
```

**推荐开发顺序**：**T2 → T3 → T4 → T6 → T7 → T8 → T9 → T10 → T11 → T15 → T5 → T1 → T12-T16 → Phase 3**

> 原因：先完成后端接口 → 再一次性写前端 → 避免前后端反复修改

---

## 📎 参考资料

- [00-REQUIRE.MD](./00-REQUIRE.MD) - 原始需求（22 项 + 2 项补充）
- [02-EVALUATION-ROUND3.md](./02-EVALUATION-ROUND3.md) - 评估报告
- [ca/dn.txt](./ca/dn.txt) - 主体预置字段（25 个）
- [ca/oids.txt](./ca/oids.txt) - OID 预置库（60+ 条）
- [ca/eku.txt](./ca/eku.txt) - 扩展密钥用法
- XCA：https://github.com/chris2511/xca
- RFC 6962（CT）、RFC 6960（OCSP）、RFC 5280（X.509+CRL）、RFC 8555（ACME）、RFC 6238（TOTP）

---

**待用户确认方案后，开始执行 Phase 1 · T2（审批自动签发）**
