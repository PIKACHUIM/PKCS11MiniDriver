import DocPage from '../components/DocPage'

export default function SecurityPage() {
  return (
    <DocPage
      title="安全设计"
      subtitle="OpenCert Manager 的密钥保护、认证机制和安全边界设计"
      badge="Security"
    >
      <h2>🔐 密钥保护体系</h2>
      <p>系统采用多层密钥保护，确保私钥在存储和传输过程中的安全性：</p>
      <pre>{`┌─────────────────────────────────────────────────────────┐
│                    私钥保护层次                           │
├─────────────────────────────────────────────────────────┤
│  Layer 1: 证书私钥                                       │
│    encrypted_privkey = AES256-GCM(temp_key, private_key) │
├─────────────────────────────────────────────────────────┤
│  Layer 2: 临时密钥                                       │
│    temp_key = HMAC(card_master_key, salt_cert)           │
├─────────────────────────────────────────────────────────┤
│  Layer 3: 卡片主密钥                                     │
│    encrypted_master = AES256-GCM(                        │
│      HMAC(user_password, salt_user),                     │
│      card_master_key                                     │
│    )                                                     │
├─────────────────────────────────────────────────────────┤
│  Layer 4 (TPM2): 主密钥由 TPM 保护                       │
│    card_master_key 由 TPM 内部密钥加密，不可导出          │
└─────────────────────────────────────────────────────────┘`}</pre>

      <h2>🔑 密钥派生</h2>
      <p>所有加密密钥均通过 HMAC 派生，避免直接使用用户密码：</p>
      <pre>{`# 用户密码保护卡片主密钥
salt_user    = CSPRNG(32 bytes)
derived_key  = HMAC-SHA256(user_password, salt_user)
enc_master   = AES256-GCM(derived_key, card_master_key)

# 卡片独立密码（任何持有此密码的人可访问此卡）
salt_card    = CSPRNG(32 bytes)
derived_key2 = HMAC-SHA256(card_password, salt_card)
enc_master2  = AES256-GCM(derived_key2, card_master_key)

# 证书临时密钥
salt_cert    = CSPRNG(32 bytes)
temp_key     = HMAC-SHA256(card_master_key, salt_cert)
enc_privkey  = AES256-GCM(temp_key, private_key)`}</pre>

      <h2>👤 用户认证</h2>
      <div className="card-grid">
        <div className="info-card">
          <div className="info-card-title">🔒 本地用户</div>
          <div className="info-card-body">
            密码使用 bcrypt（cost=12）存储，防止彩虹表攻击。
            支持 TOTP 双因素认证。
          </div>
        </div>
        <div className="info-card">
          <div className="info-card-title">☁️ 云端用户</div>
          <div className="info-card-body">
            密码不在本地存储，实时通过云端 API 认证。
            登录后返回 JWT Token（含有效期），本地可设置 PIN 码加密存储 Token。
          </div>
        </div>
        <div className="info-card">
          <div className="info-card-title">📱 TOTP 认证</div>
          <div className="info-card-body">
            支持 TOTP（RFC 6238）和 HOTP（RFC 4226）。
            TOTP 密钥加密存储在智能卡中，与私钥同等保护级别。
          </div>
        </div>
        <div className="info-card">
          <div className="info-card-title">🔑 JWT Token</div>
          <div className="info-card-body">
            使用 HMAC-SHA256 签名，包含用户 UUID、角色、过期时间。
            支持 Token 刷新，过期后需重新登录。
          </div>
        </div>
      </div>

      <h2>🛡️ 驱动通信安全</h2>
      <p>pkcs11-mock 与 client-card 之间的通信安全机制：</p>
      <table>
        <thead>
          <tr><th>机制</th><th>说明</th></tr>
        </thead>
        <tbody>
          {[
            ['随机 Bearer Token', '每次启动 client-card 时生成 32 字节随机 Token，驱动通过此 Token 认证'],
            ['localhost 绑定', 'client-card API 仅监听 127.0.0.1，不对外网暴露'],
            ['请求超时', '所有 API 请求设置超时，防止驱动阻塞'],
            ['操作审计', '所有密钥操作记录到日志，包含时间、操作类型、调用方信息'],
          ].map(([m, d]) => (
            <tr key={m as string}>
              <td><strong>{m}</strong></td>
              <td>{d}</td>
            </tr>
          ))}
        </tbody>
      </table>

      <h2>🔒 证书安全策略</h2>
      <h3>密钥存储类型模板</h3>
      <p>管理员可配置证书的密钥存储策略，限制用户的操作权限：</p>
      <table>
        <thead>
          <tr><th>策略项</th><th>说明</th></tr>
        </thead>
        <tbody>
          {[
            ['允许文件下载', '是否允许用户直接下载私钥文件'],
            ['允许云端智能卡', '是否允许存储到云端虚拟智能卡'],
            ['允许本地智能卡', '是否允许存储到本地虚拟智能卡'],
            ['允许实体智能卡', '是否允许导入到物理智能卡'],
            ['允许重新导入', '高安全性模式下禁止重新导入到其他卡'],
            ['云端备份私钥', '是否允许在云端备份私钥（加密存储）'],
            ['允许重新下发', '备份后是否允许重新下发到新设备'],
            ['最大下发次数', '限制证书可被下发的次数'],
          ].map(([p, d]) => (
            <tr key={p as string}>
              <td><code>{p}</code></td>
              <td>{d}</td>
            </tr>
          ))}
        </tbody>
      </table>

      <h2>🌐 网络安全</h2>
      <div className="callout callout-warning">
        <span className="callout-icon">⚠️</span>
        <div className="callout-body">
          <strong>生产环境建议</strong>
          云端服务（server-card）必须通过 HTTPS 部署，建议使用反向代理（Nginx/Caddy）
          并配置 TLS 1.2+ 和强密码套件。
        </div>
      </div>
      <ul>
        <li>所有外部 API 强制 HTTPS，TLS 1.2+</li>
        <li>JWT Token 有效期建议不超过 24 小时</li>
        <li>OCSP/CRL 服务可公开访问，无需认证</li>
        <li>ACME 服务通过域名验证确保申请者身份</li>
        <li>管理员操作需要额外的 TOTP 验证</li>
        <li>支持 IP 白名单限制管理接口访问</li>
      </ul>

      <h2>📋 安全审计</h2>
      <p>系统记录以下安全事件到日志：</p>
      <table>
        <thead>
          <tr><th>事件类型</th><th>日志级别</th><th>记录内容</th></tr>
        </thead>
        <tbody>
          {[
            ['用户登录/登出', 'INFO', '用户 UUID、IP、时间、成功/失败'],
            ['TOTP 验证失败', 'WARN', '用户 UUID、IP、失败次数'],
            ['密钥操作', 'INFO', '卡片 UUID、密钥 ID、操作类型、调用方'],
            ['证书颁发', 'INFO', 'CA UUID、证书 UUID、主体、有效期'],
            ['证书吊销', 'WARN', '证书 UUID、吊销原因、操作者'],
            ['非法访问', 'ERROR', 'IP、请求路径、Token 信息'],
          ].map(([e, l, c]) => (
            <tr key={e as string}>
              <td>{e}</td>
              <td>
                <span className={`badge ${l === 'INFO' ? 'badge-blue' : l === 'WARN' ? 'badge-orange' : 'badge-teal'}`}>
                  {l}
                </span>
              </td>
              <td style={{ fontSize: '0.8rem' }}>{c}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </DocPage>
  )
}
