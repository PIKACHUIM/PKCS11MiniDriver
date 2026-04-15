import DocPage from '../components/DocPage'

export default function ArchitecturePage() {
  return (
    <DocPage
      title="系统架构"
      subtitle="三层架构设计：PKCS#11 驱动层 ↔ 本地客户端 ↔ 云端服务，各层通过标准接口解耦"
      badge="Architecture"
    >
      <h2>⬡ 整体架构</h2>
      <p>OpenCert Manager 采用三层架构，每层职责清晰，通过标准接口通信：</p>
      <pre>{`┌─────────────────────────────────────────────────────────┐
│                    应用层 (Applications)                  │
│         浏览器 / 邮件客户端 / SSH / 代码签名工具           │
└──────────────────────┬──────────────────────────────────┘
                       │ PKCS#11 标准接口
┌──────────────────────▼──────────────────────────────────┐
│                  pkcs11-mock (C DLL)                     │
│         PKCS#11 驱动，注册到操作系统，提供密码学接口        │
└──────────────────────┬──────────────────────────────────┘
                       │ HTTP REST API (localhost:1026)
┌──────────────────────▼──────────────────────────────────┐
│                  client-card (Go + Electron)              │
│    本地管理端：智能卡管理、证书管理、TOTP、CSR 生成         │
│    ┌──────────┐  ┌──────────┐  ┌──────────────────────┐ │
│    │  local   │  │  tpmv2   │  │       cloud          │ │
│    │ 本地虚拟卡│  │ TPM2 卡  │  │    云端智能卡         │ │
│    └──────────┘  └──────────┘  └──────────────────────┘ │
└──────────────────────┬──────────────────────────────────┘
                       │ HTTPS REST API
┌──────────────────────▼──────────────────────────────────┐
│                  server-card (Go)                        │
│    云端服务：CA 管理、证书颁发、用户认证、ACME、CT          │
│    ┌──────────┐  ┌──────────┐  ┌──────────────────────┐ │
│    │ 本地数据库│  │  HSM 硬件│  │     证书存储          │ │
│    └──────────┘  └──────────┘  └──────────────────────┘ │
└─────────────────────────────────────────────────────────┘`}</pre>

      <h2>📦 组件说明</h2>
      <div className="card-grid">
        <div className="info-card" style={{ borderTop: '3px solid #f6ad55' }}>
          <div className="info-card-title" style={{ color: '#f6ad55' }}>pkcs11-mock</div>
          <div className="info-card-body">
            <p>C 语言编写的 PKCS#11 动态库（.dll/.so），注册到操作系统后，应用程序可通过标准 PKCS#11 接口调用密码学功能。</p>
            <ul style={{ marginTop: '0.5rem', paddingLeft: '1rem', fontSize: '0.8rem' }}>
              <li>通过 HTTP 调用 client-card API</li>
              <li>支持 Slot/Token/Session 管理</li>
              <li>支持签名、加密、密钥生成等操作</li>
            </ul>
          </div>
        </div>
        <div className="info-card" style={{ borderTop: '3px solid #63b3ed' }}>
          <div className="info-card-title" style={{ color: '#63b3ed' }}>client-card</div>
          <div className="info-card-body">
            <p>Go 语言后端 + React 前端的 Electron 桌面应用，是整个系统的核心枢纽。</p>
            <ul style={{ marginTop: '0.5rem', paddingLeft: '1rem', fontSize: '0.8rem' }}>
              <li>管理三种类型的虚拟智能卡</li>
              <li>本地 SQLite 数据库存储</li>
              <li>与云端服务同步证书</li>
              <li>监听 localhost:1026 供驱动调用</li>
            </ul>
          </div>
        </div>
        <div className="info-card" style={{ borderTop: '3px solid #4fd1c5' }}>
          <div className="info-card-title" style={{ color: '#4fd1c5' }}>server-card</div>
          <div className="info-card-body">
            <p>Go 语言编写的云端服务，提供完整的 PKI 管理能力。</p>
            <ul style={{ marginTop: '0.5rem', paddingLeft: '1rem', fontSize: '0.8rem' }}>
              <li>CA 证书颁发和吊销管理</li>
              <li>用户认证（JWT + TOTP）</li>
              <li>ACME 协议服务</li>
              <li>CRL/OCSP/CT 服务</li>
            </ul>
          </div>
        </div>
      </div>

      <h2>🗄️ 数据模型</h2>
      <h3>卡片管理</h3>
      <table>
        <thead>
          <tr><th>字段</th><th>类型</th><th>说明</th></tr>
        </thead>
        <tbody>
          {[
            ['slot_type', 'enum', 'local / tpmv2 / cloud'],
            ['card_name', 'string', '卡片显示名称'],
            ['card_uuid', 'uuid', '卡片唯一标识'],
            ['user_uuid', 'uuid', '所属用户'],
            ['created_at', 'timestamp', '创建时间'],
            ['expires_at', 'timestamp', '有效期'],
            ['card_password', 'bytes', '加密存储的主密钥列表'],
            ['remark', 'string', '备注信息'],
          ].map(([f, t, d]) => (
            <tr key={f as string}>
              <td><code>{f}</code></td>
              <td><span className="badge badge-teal">{t}</span></td>
              <td>{d}</td>
            </tr>
          ))}
        </tbody>
      </table>

      <h3>本地证书</h3>
      <table>
        <thead>
          <tr><th>字段</th><th>类型</th><th>说明</th></tr>
        </thead>
        <tbody>
          {[
            ['slot_type', 'enum', 'local / tpmv2 / cloud'],
            ['card_uuid', 'uuid', '所属卡片'],
            ['cert_type', 'enum', 'X509 / SSH / GPG / TOTP / FIDO / Login / Text / Note / Payment'],
            ['key_type', 'string', '密钥类型（RSA2048、EC256 等）'],
            ['temp_key', 'bytes', '临时密钥（AES256 加密）'],
            ['cert_data', 'bytes', '证书公开部分'],
            ['key_data', 'bytes', '私钥/私密数据（加密存储）'],
            ['remark', 'string', '备注信息'],
          ].map(([f, t, d]) => (
            <tr key={f as string}>
              <td><code>{f}</code></td>
              <td><span className="badge badge-teal">{t}</span></td>
              <td>{d}</td>
            </tr>
          ))}
        </tbody>
      </table>

      <h2>🔐 密钥加密流程</h2>
      <pre>{`# 本地/TPM2 卡片密码存储
card_master_key = random(32 bytes)

# 用户密码保护（每个用户一条记录）
salt_user = random(32 bytes)
derived_key = HMAC(user_password, salt_user)
encrypted_master = AES256_GCM(derived_key, card_master_key)
store: [salt_user, encrypted_master]

# 卡片独立密码保护
salt_card = random(32 bytes)
derived_key2 = HMAC(card_password, salt_card)
encrypted_master2 = AES256_GCM(derived_key2, card_master_key)
store: [salt_card, encrypted_master2]

# 证书私钥加密
salt_cert = random(32 bytes)
temp_key = HMAC(card_master_key, salt_cert)
encrypted_privkey = AES256_GCM(temp_key, private_key)
store: [salt_cert, encrypted_privkey]`}</pre>

      <div className="callout callout-warning">
        <span className="callout-icon">⚠️</span>
        <div className="callout-body">
          <strong>TPM2 高安全模式</strong>
          在 TPM2 高安全模式下，card_master_key 由 TPM 内部生成并永不离开 TPM，
          私钥使用 TPM 公钥加密后存储，解密必须通过 TPM 完成，密钥不可导出或恢复。
        </div>
      </div>

      <h2>🌐 服务端口规划</h2>
      <table>
        <thead>
          <tr><th>服务</th><th>端口</th><th>协议</th><th>说明</th></tr>
        </thead>
        <tbody>
          {[
            ['client-card API', '1026', 'HTTP', '供 pkcs11-mock 调用，仅监听 localhost'],
            ['client-card UI', '5173', 'HTTP', '开发模式 Web UI'],
            ['server-card API', '8080', 'HTTPS', '云端 REST API'],
            ['OCSP 服务', '8080', 'HTTP', '/ocsp/<path>'],
            ['CRL 服务', '8080', 'HTTP', '/crl/<path>'],
            ['ACME 服务', '8080', 'HTTPS', '/acme/<path>'],
          ].map(([s, p, proto, d]) => (
            <tr key={s as string}>
              <td><code>{s}</code></td>
              <td><span className="badge badge-blue">{p}</span></td>
              <td>{proto}</td>
              <td>{d}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </DocPage>
  )
}
