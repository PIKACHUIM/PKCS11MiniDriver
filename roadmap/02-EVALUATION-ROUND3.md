# OpenCert Manager 云端平台 · 第三轮全面评估报告

> 评估时间：2026-04-17
> 评估范围：`servers/` 目录（server-card 云端平台）后端 Go 代码 + 前端 React/TypeScript 代码
> 评估基准：`roadmap/00-REQUIRE.MD` 22 项核心需求 + 用户补充 2 项（PIN/PUK/Admin Key 强制加密）
> 参考资料：`roadmap/ca/dn.txt`、`roadmap/ca/oids.txt`、`roadmap/ca/eku.txt`、XCA 项目

---

## 📊 总览

| 维度         | 后端实现 | 前端页面 | 业务逻辑完整性 | 综合完成度 |
|--------------|:-------:|:-------:|:-------------:|:---------:|
| **评估结果** | 85%     | 35%     | 55%           | **~58%**  |

**核心结论**：项目后端架构和核心能力有扎实基础，路由/模型基本齐全；但存在**两大类严重问题**：

1. **前端菜单缺失**：后端 12+ 个核心 API 没有对应的前端页面入口，导致用户"看不到"功能
2. **业务链路断裂**：签发引擎未使用模板约束、审批通过后不自动签发证书、智能卡签名未强制 PIN 等关键逻辑残缺

---

## 🔴 第一类问题：前端页面严重缺失（用户误以为"完全没有"的根源）

后端路由存在，前端 `MainLayout.tsx` 侧边栏只有 8 个菜单项，覆盖率不足 40%。

| 需求编号 | 功能模块 | 后端路由 | 前端菜单 | 前端页面 | 状态 |
|:-------:|---------|:-------:|:-------:|:-------:|:----:|
| #2 | 云端智能卡存储区域 | ✅ `/api/storage-zones` | ❌ | ❌ | 🔴 无入口 |
| #3 | 云端智能卡管理 | ✅ `/api/cards` | ❌ | ❌ | 🔴 无入口 |
| #4 | 颁发证书统一管理 | ✅ `/api/certs` | ❌ | ❌ | 🔴 无入口 |
| #13 | OID 管理 | ✅ `/api/oids` | ❌ | ❌ | 🔴 无入口 |
| #14 | 吊销服务管理 | ✅ `/api/revocation-services` | ❌ | ❌ | 🔴 无入口 |
| #16 | 主体信息管理 | ✅ `/api/subject-infos` | ❌ | ❌ | 🔴 无入口 |
| #17 | 扩展信息管理 | ✅ `/api/extension-infos` | ❌ | ❌ | 🔴 无入口 |
| #18 | 证书申请审核（管理员视图） | ✅ `/api/cert-applications/approve` | ❌ | ❌ | 🔴 无入口 |
| #19 | ACME 服务配置 | ✅ `/api/acme-configs` | ❌ | ❌ | 🔴 无入口 |
| #20 | 云端 TOTP 管理 | ✅ `/api/cloud-totp` | ❌ | ❌ | 🔴 无入口 |
| #21 | 密钥存储类型模板 | ✅ `/api/templates/key-storage` | ❌ | ❌ | 🔴 无入口 |
| #22 | 支付插件/退款管理 | ✅ `/api/payment/plugins` | ❌ | ❌ | 🔴 无入口 |

---

## 🟠 第二类问题：后端业务逻辑严重缺陷

### 2.1 签发引擎（`internal/ca/issuer.go`）

- ❌ **`IssueRequest.IssuanceTmplUUID` 字段定义但从未使用**——签发前不做模板约束验证（有效期/密钥类型/允许 CA 列表）
- ❌ **不支持 EV Policy OID 写入**（`CertificatePolicies` 扩展字段未设置）
- ❌ **不支持 CT SCT embedding**（需要 poison extension + precertificate 流程）
- ❌ **不支持 Netscape 扩展、CSP 扩展、自定义 ASN.1 扩展**
- ❌ **`handleIssueCert` 强制固定 EKU = ServerAuth + ClientAuth**，完全忽略 `KeyUsageTemplate` 的用户配置
- ❌ **没有从 `IssuanceTemplate.CertExtTmplUUID` 读取并写入扩展**（CRL/OCSP/AIA/CT/EV）
- ❌ **不支持 brainpool/Ed25519/X25519/SM2/SM3/SM4**（需求 #3.2 明确要求）
- ❌ **不支持 URI SAN**（只有 DNS/IP/Email 三种）
- ❌ **不支持 RSA 1024-8192 全范围**（只有 2048/4096）

### 2.2 申请审批链路（`internal/workflow/service.go`）

- ❌ **`ApproveApplication` 只更新状态，不调用 CA 签发！** 用户审批后无法拿到证书
- ❌ 没有把签发结果的 `cert_uuid` 回写到 `cert_applications` 表
- ❌ `INSERT INTO cert_orders` 语句漏掉 `cert_apply_tmpl_uuid`、`frozen_cents`、`paid_at` 等字段

### 2.3 CA 管理（`internal/ca/service.go`）

- ❌ **`handleCreateCA` 只支持自签名 CA**，不支持**导入外部 CA（CertPEM + PrivateKey PEM）**
- ❌ **`ImportChain` 使用 `cert_pem || chain_pem` SQL 串接**，`||` 在 MySQL/PostgreSQL 上语义不同
- ❌ **没有"查询证书完整链"的 API**（用户导出证书时必须获取）
- ❌ 没有独立的 CA 根证书 PEM 下载接口

### 2.4 CT 功能（`internal/ct/service.go`）

- ❌ **`TODO: 实际调用 CT 日志服务器 API`** 完全未实现，`SCTData` 永远为空
- ❌ 状态硬编码为 "submitted"，但没有真正提交，属于**虚假状态**
- ❌ **无密码/token 认证**（需求 #15 要求"支持密码认证才能提交"）
- ❌ **没有自己的 CT 日志树**，不能作为 CT 日志服务器响应他人查询

### 2.5 验证服务（`internal/verification/service.go`）

- ❌ **邮箱验证没有真正发送邮件**，`验证码 = token 前 6 位`，用户无法获知验证码
- ❌ **HTTP 文件验证**虽有 handler，但未集成到 verification service 的统一验证流程
- ❌ **验证有效期固定 90 天**，没有按 `ExtensionTemplate` 模板配置
- ❌ **扩展信息创建时没有绑定模板**，`RequireDNSVerify`/`RequireEmailVerify` 约束未生效
- ⚠️ 缺少 CAA 记录验证（高级场景）

### 2.6 智能卡签名/解密（`internal/card/service.go`）

- ❌ **`Sign`/`Decrypt`/`GenerateKeyPair`/`DeleteCert`/`ImportCert` 完全不验证 PIN**
  - 严重违反用户补充需求："本地驱动、导入/删除证书/进行签名解密等操作的时候需要PIN码"
- ❌ **没有 PIN 会话令牌机制**，每次调用都传 PIN 不现实
- ⚠️ `ImportCert` 不支持 PKCS12 格式导入

### 2.7 吊销服务（`internal/revocation/service.go`）

- ⚠️ **`StartCRLRefreshLoop` 使用固定 1 小时 ticker**，没有按各个配置的 `CRLInterval` 独立调度
- ⚠️ 没有支持"单独启用 OCSP / 禁用 CRL" 的精细开关（虽有 `enabled` 字段但只是总开关）
- ⚠️ **OCSP 响应是自定义 JSON 格式**，而非标准 OCSP binary 响应（不符合 RFC 6960）

### 2.8 ACME 服务（`internal/acme/service.go`）

- ⚠️ `ValidateChallenge` 只改状态，**没有真正去验证 HTTP-01/DNS-01 挑战**
- ⚠️ `Finalize` 处理器是否真正签发证书并返回？需要深入审查

### 2.9 登录 TOTP 保护

- ❌ **没有"用户绑定自己登录 TOTP"的 API**（生成 secret → 显示二维码 → 验证后启用）
  - `user.TOTPSecret` 字段存在且登录验证链路有调用，但**普通用户无法自主启用**保护
- ⚠️ `verifyUserTOTPCode` 只取第一个条目，忽略多个 TOTP

### 2.10 模板预置数据缺失

- ❌ **主体模板没有预置 dn.txt 字段**（C/ST/L/O/OU/CN/emailAddress/serialNumber 等 25 个字段）
- ❌ **没有 API 返回 oids.txt 完整 EKU 列表**（共 60+ 个标准 OID，涵盖 SSL/CodeSign/Email/IPSEC/SSH/MS/EV/CFCA 等）
- ❌ **`ExtensionTemplate` 缺少 URI/RID/Other SAN 类型**
- ❌ **`ExtensionTemplate` 缺少 `verify_expires_days` 字段**（需求 #17"根据拓展信息模板规定时间判断有效性"）
- ❌ **`SubjectTemplate.Fields` 字段没有预置约束表**，允许任意 JSON

### 2.11 权限检查散弹问题

大量 handler 里硬编码 `claims.Role == "admin"`，未考虑新增的 `super_admin`/`operator` 角色：

- `handleListSubjectInfos`、`handleDeleteSubjectInfo`、`handleListCertApplications`
- `handleRenewCert`、`handleExportCert`、`handleListCertsFiltered`
- `handleGetTOTPCode`、`handleGetCertOrder`

---

## 🟡 第三类问题：数据模型小问题

| 问题                                                       | 位置                     |
|------------------------------------------------------------|--------------------------|
| `ExtensionTemplate` 缺 URI/RID/Other 类型、缺 `verify_expires_days` | `models.go`             |
| `Certificate` 缺 `SANURIs`/`CertificatePolicies`           | `models.go`             |
| `CustomOID` 缺 `is_critical` 扩展标志                       | `models.go`             |
| 缺 `PINSession` 表（PIN 会话令牌）                          | 需新增                   |
| 缺 `CertChain` 表或视图（证书链查询）                       | 需新增                   |

---

## 📋 修改建议清单（按优先级分层）

### P0 · 阻塞用户使用的问题

1. **补全 12 个前端菜单/页面**：StorageZones、Cards、AllCerts、OIDs、RevocationServices、SubjectInfos、ExtensionInfos、CertApplications、ACMEConfigs、CloudTOTP、KeyStorageTemplates、PaymentPlugins
2. **审批通过自动签发**：`workflow.ApproveApplication` → 调用 `caSvc.IssueCert`，回写 `cert_uuid`
3. **签发引擎写入拓展模板**：从 `CertExtTmplUUID` 读取 CRL/OCSP/AIA/EV，自动写入证书扩展
4. **Sign/Decrypt/KeyGen/删除证书 强制 PIN 验证** + **PIN 会话令牌机制**（避免每次调用传 PIN）
5. **CA 导入外部 CA 接口**（CertPEM + PrivateKey PEM，含证书链）
6. **完整证书链查询 API**：`GET /api/cas/{uuid}/chain`、`GET /api/certs/{uuid}/chain`

### P1 · 功能完整性

7. **dn.txt 预置主体字段 API**：`GET /api/meta/subject-fields`
8. **oids.txt 预置 EKU/OID API**：`GET /api/meta/predefined-oids`（按类别分组）
9. **签发模板约束验证**：有效期在 `ValidDays` 列表中、密钥类型在 `AllowedKeyTypes` 列表中、CA 在 `AllowedCAUUIDs` 列表中
10. **邮箱验证真正发送邮件**（集成 SMTP 配置）
11. **HTTP/DNS 验证绑定 ExtensionTemplate**：按模板要求执行验证，按模板有效期判断
12. **CT 真正提交到 CT 日志服务器**（RFC 6962 add-chain HTTP API），返回真实 SCT
13. **用户绑定登录 TOTP API**：`POST /api/auth/totp/generate-secret`、`POST /api/auth/totp/bind`、`DELETE /api/auth/totp/unbind`
14. **Key Usage Template 写入签发**：替换 `handleIssueCert` 中硬编码的 EKU
15. **Netscape/CSP/ASN.1 扩展写入签发**（使用 `encoding/asn1` 写自定义扩展）
16. **OCSP 返回标准 binary 响应**（RFC 6960，用 `golang.org/x/crypto/ocsp`）

### P2 · 合规与密码算法扩展

17. **支持 brainpool、Ed25519、X25519**（使用 `filippo.io/edwards25519`）
18. **支持 SM2/SM3/SM4**（使用 `github.com/tjfoc/gmsm`）
19. **支持 RSA 1024-8192 全范围**
20. **支持 SHA3、MD5/MD4**（兼容老证书）

### P3 · 工程质量

21. **权限检查统一用 `auth.IsAdmin(role)`/`IsOperatorOrAbove(role)`**，消除散弹式 `role == "admin"`
22. **CRL 定时器按 `CRLInterval` 独立调度**（每个 CA 独立 ticker）
23. **`ImportChain` SQL 兼容性修复**（不用 `||` 字符串拼接，改用应用层拼接后 UPDATE）
24. **ACME finalize 真正签发 + 挑战真正验证**

---

## 🎯 建议路线图

```
Phase 1（约 1 周）· 前端页面补全 + 关键业务断链修复
  ├─ 12 个前端页面开发
  ├─ ApproveApplication → 自动签发证书
  ├─ 证书拓展模板签发时写入（CRL/OCSP/AIA/EV）
  └─ PIN 会话机制 + 签名/解密强制 PIN

Phase 2（约 1 周）· 签发引擎深化 + 数据预置
  ├─ dn.txt / oids.txt 预置数据 API
  ├─ 模板约束验证（签发前校验）
  ├─ 导入外部 CA + 证书链查询
  ├─ Key Usage Template 实际应用
  ├─ 邮箱验证发送 / HTTP 验证绑定模板
  └─ CT 真实提交到外部 CT 日志

Phase 3（按需）· 算法与合规扩展
  ├─ SM2/SM3/SM4 国密
  ├─ Ed25519/X25519/brainpool
  ├─ 登录 TOTP 绑定流程
  └─ 标准 OCSP binary 响应
```

---

## ✅ 已达成需求（做得好的地方）

- ✅ 多数据库可选支持（SQLite/MySQL/PostgreSQL）
- ✅ RBAC 五角色权限体系（super_admin/admin/operator/user/readonly）
- ✅ 审计日志链式哈希完整性
- ✅ PIN/PUK/Admin Key AES-256-GCM 加密存储（三级保护，满足用户新增需求 #2）
- ✅ 订单完整 9 状态机（pending_payment → paid → applying → reviewing → issuing → completed/rejected/cancelled/refunded）
- ✅ 支付系统余额冻结/解冻 + 退款工单
- ✅ ACME 协议路由框架（directory/nonce/account/order/authz/challenge/finalize/cert 全路由）
- ✅ CRL 生成（RFC 5280）+ 自定义路径路由（`/pki/crl/<path>` 等）
- ✅ TOTP 验证码计算（RFC 6238）+ 登录二次验证
- ✅ 证书续期 + 多格式导出（PEM/DER/PKCS12）
- ✅ 前端 API 定义完整（所有接口 TypeScript 类型齐全）

---

## 📝 结论

项目后端架构和核心能力已有**扎实基础**（约 58% 完成度），主要缺口集中在：

1. **前端页面覆盖不足**（35%）
2. **签发链路业务逻辑补全**（申请审批 → 自动签发 → 扩展写入 → PIN 会话）
3. **模板预置数据未集成**（dn.txt / oids.txt）

按 P0-P3 分阶段修复，预计 **2-3 周**可达到 90%+ 的需求覆盖度。

---

**附：重点参考文件**

- 需求：`roadmap/00-REQUIRE.MD`（22 项核心 + 2 项补充）
- 预置数据：`roadmap/ca/dn.txt`（25 个主体字段）、`roadmap/ca/oids.txt`（60+ OID）、`roadmap/ca/eku.txt`
- XCA 开源项目：https://github.com/chris2511/xca （参考证书拓展模板、主体字段、EKU 分类）
- RFC 6962（CT）、RFC 6960（OCSP）、RFC 5280（X.509 + CRL）、RFC 8555（ACME）、RFC 6238（TOTP）
