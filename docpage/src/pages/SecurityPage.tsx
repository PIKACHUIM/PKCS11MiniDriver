import DocPage from '../components/DocPage'

export default function SecurityPage() {
  return (
    <DocPage
      title="安全设计"
      subtitle="OpenCert Manager 的密钥保护、认证机制、威胁模型与安全边界设计"
      badge="Security v2.0"
    >
      <h2>🛡️ 威胁模型</h2>
      <p>系统针对以下主要威胁场景进行了专项防护设计：</p>
      <table>
        <thead>
          <tr><th>威胁</th><th>攻击面</th><th>缓解措施</th></tr>
        </thead>
        <tbody>
          {[
            ['本地数据库被拷贝', '物理访问', 'SQLCipher 全库加密，密钥由 TPM 封装或用户主密码派生'],
            ['IPC 通道被劫持', '本地恶意进程', 'Windows DACL / Unix 0600 权限隔离'],
            ['REST API 未授权访问', '本地恶意进程', '启动时生成随机 Bearer Token，写入 0600 权限文件'],
            ['Cloud Slot 中间人攻击', '网络', '强制 HTTPS，验证 TLS 证书，不跳过证书验证'],
            ['暴力破解 PIN/密码', '网络/本地', '5 次失败锁定 15 分钟，Argon2id 密钥派生'],
            ['API 滥用/DDoS', '网络', '令牌桶速率限制（100 req/min/IP）'],
            ['JWT Token 泄露', '网络', 'Token 轮换，登出黑名单，短有效期（15 分钟）'],
            ['审计日志篡改', '物理/内部', '链式哈希完整性保护'],
            ['内存中私钥泄露', '内存转储', '使用后立即清零（清零字节切片）'],
          ].map(([t, a, m]) => (
            <tr key={t as string}>
              <td><strong>{t}</strong></td>
              <td><span className="badge badge-orange">{a}</span></td>
              <td style={{ fontSize: '0.8rem' }}>{m}</td>
            </tr>
          ))}
        </tbody>
      </table>

      <h2>🔐 三层加密架构</h2>
      <p>系统采用三层密钥保护，确保私钥在存储和传输过程中的安全性：</p>
      <pre>{`┌─────────────────────────────────────────────────────────────┐
│                      私钥保护层次                             │
├─────────────────────────────────────────────────────────────┤
│  Layer 1: 用户密码 → 卡片主密钥                               │
│    salt = CSPRNG(32 bytes)                                   │
│    derived = Argon2id(password, salt, t=3, m=64MB, p=4)     │
│    enc_master = AES256-GCM(derived, card_master_key)        │
├─────────────────────────────────────────────────────────────┤
│  Layer 2: 卡片主密钥 → 临时密钥                               │
│    salt_cert = CSPRNG(32 bytes)                              │
│    temp_key = HMAC-SHA256(card_master_key, salt_cert)       │
├─────────────────────────────────────────────────────────────┤
│  Layer 3: 临时密钥 → 私钥（每证书独立）                        │
│    enc_privkey = AES256-GCM(                                 │
│      nonce=random(12), AAD=card_uuid+cert_uuid,             │
│      key=temp_key, plaintext=private_key                    │
│    )                                                         │
├─────────────────────────────────────────────────────────────┤
│  Layer 4 (TPM2 高安全): 主密钥由 TPM 保护                     │
│    card_master_key 由 TPM 内部密钥加密，不可导出              │
└─────────────────────────────────────────────────────────────┘`}</pre>

      <div className="callout callout-info">
        <span className="callout-icon">ℹ️</span>
        <div className="callout-body">
          <strong>AAD 绑定防替换攻击</strong>
          AES-256-GCM 加密时附加 <code>card_uuid + cert_uuid</code> 作为 AAD（附加认证数据），
          防止密文被跨上下文替换。每个证书拥有独立的临时密钥，单个证书泄露不影响其他证书。
        </div>
      </div>

      <h2>🔑 密钥派生与密码哈希</h2>
      <h3>Argon2id 密钥派生（优先）</h3>
      <pre>{`# 新用户密码哈希 / 密钥派生（Argon2id）
salt         = CSPRNG(32 bytes)
derived_key  = Argon2id(
  password   = user_password,
  salt       = salt,
  time       = 3,          # 迭代次数
  memory     = 65536,      # 64 MB 内存
  threads    = 4,          # 并行度
  keyLen     = 32          # 输出 32 字节
)
enc_master   = AES256-GCM(derived_key, card_master_key)

# 卡片独立 PIN 码保护（任何持有此 PIN 的人可访问此卡）
salt_pin     = CSPRNG(32 bytes)
derived_pin  = Argon2id(card_pin, salt_pin, ...)
enc_master2  = AES256-GCM(derived_pin, card_master_key)

# 证书临时密钥
salt_cert    = CSPRNG(32 bytes)
temp_key     = HMAC-SHA256(card_master_key, salt_cert)
enc_privkey  = AES256-GCM(temp_key, private_key)`}</pre>

      <h3>密码哈希算法对比</h3>
      <table>
        <thead>
          <tr><th>算法</th><th>参数</th><th>用途</th><th>说明</th></tr>
        </thead>
        <tbody>
          <tr>
            <td><span className="badge badge-green">Argon2id（优先）</span></td>
            <td>t=3, m=64MB, p=4</td>
            <td>新用户密码哈希</td>
            <td>抗暴力破解，内存硬函数</td>
          </tr>
          <tr>
            <td><span className="badge badge-blue">bcrypt（兼容）</span></td>
            <td>cost ≥ 13</td>
            <td>旧用户兼容</td>
            <td>登录时自动迁移为 Argon2id</td>
          </tr>
        </tbody>
      </table>

      <h2>🔓 PIN / PUK / Admin Key 安全设计</h2>
      <p>PIN、PUK、Admin Key 均采用<strong>加密存储</strong>而非简单哈希验证，三者均不以明文存储：</p>
      <pre>{`# PIN 码加密存储流程
1. 生成 32 字节随机 salt
2. Argon2id(PIN, salt) → 派生密钥
3. 派生密钥 → AES-256-GCM 加密卡片主密钥
4. 存储 salt + 加密后的主密钥

# 验证时
1. Argon2id(输入PIN, salt) → 派生密钥
2. 尝试 AES-256-GCM 解密
3. GCM 认证标签验证通过 = PIN 正确（无需明文比对）`}</pre>

      <h3>权限层级</h3>
      <table>
        <thead>
          <tr><th>凭据</th><th>权限</th><th>最大错误次数</th><th>锁定后操作</th></tr>
        </thead>
        <tbody>
          <tr>
            <td><span className="badge badge-teal">Admin Key</span></td>
            <td>可重置 PUK 和 PIN，执行所有管理操作</td>
            <td>无限制（有速率限制）</td>
            <td>—</td>
          </tr>
          <tr>
            <td><span className="badge badge-orange">PUK 码</span></td>
            <td>可重置 PIN 码，PIN 超限后使用</td>
            <td>10 次</td>
            <td>需要 Admin Key 解锁</td>
          </tr>
          <tr>
            <td><span className="badge badge-blue">PIN 码</span></td>
            <td>导入/删除证书、签名/解密操作</td>
            <td>5 次</td>
            <td>需要 PUK 解锁</td>
          </tr>
        </tbody>
      </table>

      <div className="callout callout-warning">
        <span className="callout-icon">⚠️</span>
        <div className="callout-body">
          <strong>强制加密存储</strong>
          PIN、PUK、Admin Key 均不以明文或简单哈希存储，而是作为 AES-256-GCM 的解密密钥使用。
          只有正确的凭据才能解密卡片主密钥，错误凭据会导致 GCM 认证标签验证失败。
        </div>
      </div>

      <h2>👤 用户认证</h2>
      <div className="card-grid">
        <div className="info-card">
          <div className="info-card-title">🔒 本地用户</div>
          <div className="info-card-body">
            密码优先使用 Argon2id（t=3, m=64MB）存储，旧用户 bcrypt 登录后自动迁移。
            支持 TOTP 双因素认证，可设置 PIN 码快速解锁。
          </div>
        </div>
        <div className="info-card">
          <div className="info-card-title">☁️ 云端用户</div>
          <div className="info-card-body">
            密码不在本地存储，实时通过云端 API 认证。
            登录后返回 JWT Token（Access 15min / Refresh 7d），本地可设置 PIN 码加密存储 Token。
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
            优先使用 ES256（ECDSA P-256）签名，支持 RS256 / HS256。
            Access Token 15 分钟有效期，Refresh Token 7 天，登出后加入黑名单。
          </div>
        </div>
      </div>

      <h2>🛡️ IPC 通道安全</h2>
      <p>pkcs11-mock 与 client-card 之间通过系统级 IPC 通信，而非网络端口：</p>
      <table>
        <thead>
          <tr><th>平台</th><th>传输方式</th><th>路径</th><th>安全隔离</th></tr>
        </thead>
        <tbody>
          {[
            ['Windows', 'Named Pipe', '\\\\.\\pipe\\opencert-pkcs11', 'DACL（仅当前用户 + SYSTEM）'],
            ['Linux', 'Unix Domain Socket', '/tmp/opencert-pkcs11.sock', '文件权限 0600'],
            ['macOS', 'Unix Domain Socket', '/tmp/opencert-pkcs11.sock', '文件权限 0600'],
          ].map(([p, t, path, s]) => (
            <tr key={p as string}>
              <td><strong>{p}</strong></td>
              <td>{t}</td>
              <td><code style={{ fontSize: '0.75rem' }}>{path}</code></td>
              <td style={{ fontSize: '0.8rem' }}>{s}</td>
            </tr>
          ))}
        </tbody>
      </table>

      <h3>REST API 安全（本地 client-card）</h3>
      <table>
        <thead>
          <tr><th>机制</th><th>说明</th></tr>
        </thead>
        <tbody>
          {[
            ['随机 Bearer Token', '启动时生成随机 Token，写入 0600 权限文件，驱动读取后认证'],
            ['localhost 绑定', 'client-card API 默认仅监听 127.0.0.1，非本地绑定输出安全警告'],
            ['速率限制', '100 请求/分钟/IP，令牌桶算法'],
            ['CORS 限制', '仅允许 localhost 来源的跨域请求'],
            ['输入验证', '所有 API 参数严格校验，防止注入攻击'],
          ].map(([m, d]) => (
            <tr key={m as string}>
              <td><strong>{m}</strong></td>
              <td>{d}</td>
            </tr>
          ))}
        </tbody>
      </table>

      <h2>🔒 TPM2 安全模式</h2>
      <table>
        <thead>
          <tr><th>安全等级</th><th>密钥存储</th><th>可恢复</th><th>可导出</th><th>云端备份</th></tr>
        </thead>
        <tbody>
          <tr>
            <td><span className="badge badge-green">高安全性</span></td>
            <td>TPM 内部，片上不可导出</td>
            <td>❌</td>
            <td>❌</td>
            <td>❌</td>
          </tr>
          <tr>
            <td><span className="badge badge-orange">中安全性</span></td>
            <td>本地 DB + TPM 片上密钥加密 + 用户云端公钥加密</td>
            <td>✅</td>
            <td>❌</td>
            <td>可选</td>
          </tr>
          <tr>
            <td><span className="badge badge-blue">低安全性</span></td>
            <td>本地 DB + Argon2id 密码加密 + 用户云端公钥加密</td>
            <td>✅</td>
            <td>❌</td>
            <td>可选</td>
          </tr>
        </tbody>
      </table>

      <h2>🔒 证书存储策略</h2>
      <p>管理员可通过密钥存储类型模板配置证书的存储和下发策略：</p>
      <table>
        <thead>
          <tr><th>策略项</th><th>说明</th></tr>
        </thead>
        <tbody>
          {[
            ['允许文件下载', '是否允许用户直接下载私钥文件（文件下载默认无限次下发）'],
            ['允许云端智能卡', '是否允许存储到云端虚拟智能卡'],
            ['允许本地智能卡', '是否允许存储到本地虚拟智能卡'],
            ['允许实体智能卡', '是否允许导入到物理智能卡'],
            ['允许重新导入', '高安全性模式下禁止重新导入到其他卡'],
            ['云端备份私钥', '中低安全性和实体卡可选，备份后支持恢复'],
            ['允许重新下发', '备份了私钥的证书可选，支持下发到新设备'],
            ['最大下发次数', '限制证书可被下发的次数（0 表示无限制）'],
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
          并配置 TLS 1.2+ 和强密码套件。数据库连接必须启用 SSL。
        </div>
      </div>
      <ul>
        <li>所有外部 API 强制 HTTPS，TLS 1.2+</li>
        <li>JWT Access Token 有效期 15 分钟，Refresh Token 7 天</li>
        <li>登录失败 5 次锁定 15 分钟，防暴力破解</li>
        <li>OCSP/CRL 服务可公开访问，无需认证</li>
        <li>ACME 服务通过域名/IP 验证确保申请者身份</li>
        <li>Cloud Slot 通信强制 HTTPS，不跳过 TLS 证书验证</li>
        <li>PostgreSQL 数据库连接启用 SSL（<code>sslmode=require</code>）</li>
      </ul>

      <h2>📋 审计日志（链式哈希）</h2>
      <p>审计日志采用链式哈希保护完整性，每条日志包含前一条的 SHA-256 哈希，直接修改数据库记录会被检测到：</p>
      <pre>{`# 审计日志记录格式
{
  "uuid":      "...",
  "prev_hash": "SHA256(前一条日志的完整内容)",
  "log_type":  "security",
  "level":     "warn",
  "title":     "登录失败",
  "content":   "用户 xxx 密码错误，第 3 次尝试",
  "created_at":"2026-04-17T12:00:00Z"
}

# 完整性验证
读取时自动验证哈希链，断链标记 integrity_broken = true`}</pre>

      <table>
        <thead>
          <tr><th>事件类型</th><th>日志级别</th><th>记录内容</th></tr>
        </thead>
        <tbody>
          {[
            ['用户登录/登出', 'INFO', '用户 UUID、IP、时间、成功/失败'],
            ['密码修改', 'INFO', '用户 UUID、时间'],
            ['TOTP 验证失败', 'WARN', '用户 UUID、IP、失败次数'],
            ['PIN 验证失败/重置', 'WARN', '卡片 UUID、失败次数、操作类型'],
            ['卡片操作', 'INFO', '创建/删除/解锁卡片，卡片 UUID'],
            ['密钥操作', 'INFO', '卡片 UUID、密钥 ID、操作类型（签名/解密）'],
            ['证书颁发', 'INFO', 'CA UUID、证书 UUID、主体、有效期'],
            ['证书吊销', 'WARN', '证书 UUID、吊销原因、操作者'],
            ['非法访问', 'ERROR', 'IP、请求路径、Token 信息'],
            ['系统事件', 'INFO', '服务启动/停止、配置变更'],
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

      <h2>✅ 安全加固清单</h2>
      <h3>本地管理端（client-card）</h3>
      <ul>
        <li>API 绑定地址为 127.0.0.1</li>
        <li>Bearer Token 文件权限 0600</li>
        <li>SQLite 数据库文件权限 0600</li>
        <li>IPC 通道权限正确设置（DACL / 0600）</li>
        <li>审计日志已启用，定期验证哈希链完整性</li>
        <li>TPM 可用性已确认（如需高安全性模式）</li>
        <li>内存中私钥使用后立即清零</li>
      </ul>
      <h3>云端平台（server-card）</h3>
      <ul>
        <li>TLS 证书有效且自动续期</li>
        <li>数据库连接使用 SSL（sslmode=require）</li>
        <li>JWT 密钥长度 ≥ 256 位，优先使用 ES256</li>
        <li>支付插件配置参数已加密存储</li>
        <li>审计日志已启用，定期验证完整性</li>
        <li>配置反向代理（Nginx/Caddy）</li>
        <li>防火墙规则已配置，仅开放必要端口</li>
        <li>定期备份数据库（全量 + WAL 增量）</li>
      </ul>
    </DocPage>
  )
}
