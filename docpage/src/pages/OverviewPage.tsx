import DocPage from '../components/DocPage'

const serverModules = [
  { icon: '👤', title: '用户管理', desc: '注册/登录/TOTP 双因素认证、个人信息管理、公钥对管理、RBAC 角色权限' },
  { icon: '💳', title: '卡存储域', desc: '本地数据库、TPM2 硬件、HSM 硬件三种存储区域，支持自定义驱动' },
  { icon: '🃏', title: '云智能卡', desc: '虚拟智能卡管理，支持多用户权限、PIN/PUK/Admin Key 三级密码保护' },
  { icon: '📜', title: '证书管理', desc: 'X.509/GPG/SSH 证书全生命周期，颁发/吊销/续期/分配，关联订单与存储策略' },
  { icon: '🏛️', title: 'CA 管理', desc: '多级 CA 链、导入/颁发 CA、证书链管理、吊销列表管理' },
  { icon: '📋', title: '颁发模板', desc: '颁发模板、主体模板、扩展信息模板、密钥用途模板、证书拓展模板、密钥存储类型模板' },
  { icon: '📝', title: '申请模板', desc: '面向用户的申请配置，指定 CA/有效期/密钥算法，支持审批流程和续期' },
  { icon: '🔍', title: '主体/扩展验证', desc: '域名 TXT/HTTP 验证、邮箱验证码验证、IP 验证，存储验证时间判断有效性' },
  { icon: '🤖', title: 'ACME 服务', desc: '兼容 RFC 8555，支持多实例（/acme/<路径>），不同 CA 和证书颁发模板配置' },
  { icon: '🔑', title: 'OID 管理', desc: '自定义 OID，支持扩展密钥用途、证书主体字段、EV 声明、ASN.1 扩展字段' },
  { icon: '🚫', title: '吊销服务', desc: 'CRL/OCSP/CAIssuer 服务，按 CA 配置，支持自定义路径和 CRL 定时自动更新' },
  { icon: '📊', title: 'CT 透明度', desc: '证书提交 CT 日志，支持 CT 查询和列表管理，密码认证才能提交' },
  { icon: '🛒', title: '订单与支付', desc: '证书购买、订单管理、证书申请审批、多支付插件、用户充值、退款管理' },
  { icon: '🔐', title: 'TOTP验证', desc: '内置 TOTP/HOTP 验证器，存储和查看验证码，支持标准 TOTP URI 格式' },
  { icon: '🌐', title: '门户首页', desc: '面向公众的展示页面，项目功能介绍、证书产品列表、安全特性展示' },
]

const clientModules = [
  { icon: '👤', title: '用户管理', desc: '本地用户（bcrypt 密码）和云端用户（JWT Token）两种类型，支持 PIN 码快速解锁' },
  { icon: '💳', title: '卡片管理', desc: 'Local/TPM2/Cloud 三种卡槽 CRUD，PIN/PUK/Admin Key 管理，多用户权限共享' },
  { icon: '📜', title: '证书管理', desc: '导入（PKCS12/PEM/私钥+证书/纯证书自动匹配私钥）、导出、删除、查看详情' },
  { icon: '🔧', title: 'PKI 工具', desc: 'CSR 生成与管理、本地 CA 管理、证书签发、自签名证书、证书格式转换' },
  { icon: '🔐', title: 'TOTP管理', desc: '添加 TOTP/HOTP 条目，实时显示验证码和倒计时，支持标准 URI 格式导入' },
  { icon: '☁️', title: '云端同步', desc: '云端证书下发到本地/智能卡，自动/手动同步，通过 pkcs11-mock 注册到系统' },
]

const certTypes = [
  { type: 'X.509', desc: 'TLS/SSL、代码签名、邮件加密、客户端认证', badge: 'badge-blue', ca: true, import: true },
  { type: 'GPG', desc: '邮件签名/加密、软件包签名（仅导入）', badge: 'badge-teal', ca: false, import: true },
  { type: 'SSH', desc: 'SSH 身份认证密钥（仅导入）', badge: 'badge-orange', ca: false, import: true },
]

const slotTypes = [
  {
    name: 'local',
    title: '本地智能卡',
    desc: '证书和私钥存储在本地 SQLite 数据库，私钥使用三层 AES-256-GCM 加密，临时密钥由卡片主密钥保护',
    security: '中等',
    color: '#f6ad55',
    offline: true,
    hw: false,
    cross: false,
  },
  {
    name: 'tpmv2',
    title: 'TPM2 智能卡',
    desc: '高安全性：密钥存储在 TPM 内部不可导出；中安全性：TPM 片上密钥加密 + 云端公钥加密备份',
    security: '高',
    color: '#4fd1c5',
    offline: true,
    hw: true,
    cross: false,
  },
  {
    name: 'cloud',
    title: '云端智能卡',
    desc: '证书和私钥存储在云端，本地仅缓存公开信息，所有签名/解密操作通过云端 API 完成，私钥不离开服务器',
    security: '云端托管',
    color: '#63b3ed',
    offline: false,
    hw: true,
    cross: true,
  },
]

const templateTypes = [
  { name: '证书颁发模板', desc: '是否 CA、路径长度、可选有效期、允许私钥类型、可颁发 CA 列表' },
  { name: '主体模板', desc: '规定 CN/O/OU/C/ST/L 等字段的必填性、默认值、允许长度' },
  { name: '扩展信息模板', desc: '邮箱/DNS/URI/RID/IP 的允许数量和验证规则（TXT/HTTP/邮箱验证码）' },
  { name: '密钥用途模板', desc: 'X509 密钥用法和扩展密钥用法，支持自定义 OID' },
  { name: '证书拓展模板', desc: 'CRL/OCSP/AIA/CSP/Netscape/EV/ASN.1/CT 等证书扩展字段配置' },
  { name: '密钥存储类型模板', desc: '存储方式多选、安全等级、云端备份策略、下发次数限制' },
]

export default function OverviewPage() {
  return (
    <DocPage
      title="功能概览"
      subtitle="OpenCert Manager 提供从密钥生成到证书颁发的完整 PKI 管理能力，覆盖云端服务、本地客户端和 PKCS#11 驱动三个层次"
      badge="Overview"
    >
      <h2>🗂️ 云端服务模块（server-card）</h2>
      <p>云端服务提供完整的 CA 管理和证书颁发能力，通过 REST API（端口 :1027）对外提供服务，使用 PostgreSQL 存储数据。</p>
      <div className="card-grid">
        {serverModules.map((m) => (
          <div key={m.title} className="info-card">
            <div className="info-card-title">
              {m.icon} {m.title}
            </div>
            <div className="info-card-body">{m.desc}</div>
          </div>
        ))}
      </div>

      <h2>📋 模板体系</h2>
      <p>证书颁发模板是核心，引用其他五种模板，证书申请模板面向用户，基于颁发模板进一步约束。</p>
      <table>
        <thead>
          <tr>
            <th>模板类型</th>
            <th>主要配置项</th>
          </tr>
        </thead>
        <tbody>
          {templateTypes.map((t) => (
            <tr key={t.name}>
              <td><strong>{t.name}</strong></td>
              <td style={{ fontSize: '0.85rem' }}>{t.desc}</td>
            </tr>
          ))}
        </tbody>
      </table>

      <h2>📜 支持的证书格式</h2>
      <table>
        <thead>
          <tr>
            <th>格式</th>
            <th>用途</th>
            <th>CA 颁发</th>
            <th>外部导入</th>
          </tr>
        </thead>
        <tbody>
          {certTypes.map((c) => (
            <tr key={c.type}>
              <td><span className={`badge ${c.badge}`}>{c.type}</span></td>
              <td>{c.desc}</td>
              <td>{c.ca ? '✅' : '❌'}</td>
              <td>✅（必须含密钥）</td>
            </tr>
          ))}
        </tbody>
      </table>

      <h2>💳 智能卡类型（client-card）</h2>
      <p>client-card 支持三种智能卡槽类型，通过统一的 <code>SlotProvider</code> 接口实现，后续可无缝扩展新类型。</p>
      <div className="card-grid">
        {slotTypes.map((s) => (
          <div key={s.name} className="info-card" style={{ borderTop: `3px solid ${s.color}` }}>
            <div className="info-card-title" style={{ color: s.color }}>
              {s.title}
              <span className="badge badge-blue" style={{ marginLeft: 'auto', fontSize: '0.65rem' }}>
                {s.name}
              </span>
            </div>
            <div className="info-card-body">{s.desc}</div>
            <div style={{ marginTop: '0.75rem', fontSize: '0.75rem', color: 'var(--color-text-muted)', display: 'flex', gap: '0.75rem' }}>
              <span>安全级别：<span style={{ color: s.color }}>{s.security}</span></span>
              <span>离线可用：{s.offline ? '✅' : '❌'}</span>
              <span>跨设备：{s.cross ? '✅' : '❌'}</span>
            </div>
          </div>
        ))}
      </div>

      <h2>🔐 密钥安全策略</h2>
      <p>根据密钥存储类型模板，支持三级安全策略，高安全性使用 TPM EK 认证确保密钥在允许的 TPM 内生成：</p>
      <table>
        <thead>
          <tr>
            <th>安全级别</th>
            <th>存储方式</th>
            <th>可恢复</th>
            <th>可导出</th>
            <th>云端备份</th>
          </tr>
        </thead>
        <tbody>
          <tr>
            <td><span className="badge badge-green">高安全性</span></td>
            <td>TPM 片上存储，密钥不可导出（EK 认证）</td>
            <td>❌</td>
            <td>❌</td>
            <td>❌</td>
          </tr>
          <tr>
            <td><span className="badge badge-orange">中安全性</span></td>
            <td>本地存储，TPM 片上密钥加密 + 用户云端公钥加密</td>
            <td>✅</td>
            <td>❌</td>
            <td>可选</td>
          </tr>
          <tr>
            <td><span className="badge badge-blue">低安全性</span></td>
            <td>本地存储，用户密码加密 + 用户云端公钥加密</td>
            <td>✅</td>
            <td>❌</td>
            <td>可选</td>
          </tr>
        </tbody>
      </table>

      <h2>🖥️ 本地客户端功能（client-card）</h2>
      <p>client-card 是基于 Electron 的跨平台桌面应用，同时支持 Web 访问模式，REST API 监听 :1026，IPC 服务通过 Named Pipe（Windows）或 Unix Socket（Linux/macOS）与 pkcs11-mock 通信。</p>
      <div className="card-grid">
        {clientModules.map((m) => (
          <div key={m.title} className="info-card">
            <div className="info-card-title">
              {m.icon} {m.title}
            </div>
            <div className="info-card-body">{m.desc}</div>
          </div>
        ))}
      </div>

      <h2>⚙️ PKCS#11 驱动（pkcs11-mock）</h2>
      <p>符合 PKCS#11 v2.40 标准的 C 语言动态库，通过 IPC 与 client-card 通信，将虚拟智能卡注册到操作系统。</p>
      <table>
        <thead>
          <tr>
            <th>能力</th>
            <th>说明</th>
          </tr>
        </thead>
        <tbody>
          {[
            ['密钥类型', 'RSA 1024–8192、ECC P-256/384/521、Brainpool、Ed/X25519、SM2'],
            ['摘要算法', 'SHA-1/256/384/512、SHA3、MD5、MD4、SM3'],
            ['加密算法', 'AES-128/256（GCM/CBC）、RC4、ChaCha20-Poly1305、SM4'],
            ['证书类型', 'X.509、GPG、SSH'],
            ['片上生成', '支持片上生成密钥和 CSR，确保密钥安全'],
            ['IPC 通信', 'Named Pipe（Windows）/ Unix Socket（Linux/macOS）'],
            ['平台支持', 'Windows x64（.dll）、Linux x64（.so）、macOS arm64/x64（.dylib）'],
          ].map(([k, v]) => (
            <tr key={k as string}>
              <td><strong>{k}</strong></td>
              <td>{v}</td>
            </tr>
          ))}
        </tbody>
      </table>

      <div className="callout callout-info">
        <span className="callout-icon">ℹ️</span>
        <div className="callout-body">
          <strong>多语言与多主题支持</strong>
          前端使用 React 18 + Ant Design 5.x，支持中英文切换，支持亮色/暗黑/跟随系统三种主题模式，Electron 桌面端支持系统托盘图标。
        </div>
      </div>
    </DocPage>
  )
}
