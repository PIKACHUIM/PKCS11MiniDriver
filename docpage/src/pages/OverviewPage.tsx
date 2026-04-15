import DocPage from '../components/DocPage'

const modules = [
  { icon: '👤', title: '用户管理', desc: '注册/登录/TOTP 双因素认证、个人信息管理、公钥对管理' },
  { icon: '💳', title: '智能卡存储', desc: '本地数据库、TPM2 硬件、HSM 硬件三种存储区域' },
  { icon: '🃏', title: '云端智能卡', desc: '虚拟智能卡管理，支持多用户权限、卡片密码保护' },
  { icon: '📜', title: '证书管理', desc: 'X.509/GPG/SSH 证书全生命周期，颁发/吊销/续期/分配' },
  { icon: '🏛️', title: 'CA 管理', desc: '多级 CA 链、CRL/OCSP 吊销服务、CAIssuer 服务' },
  { icon: '📋', title: '证书模板', desc: '颁发模板、主体模板、拓展信息模板、密钥用途模板' },
  { icon: '🔍', title: '主体/拓展验证', desc: '域名 TXT/HTTP 验证、邮箱验证码验证、IP 验证' },
  { icon: '🤖', title: 'ACME 服务', desc: '兼容 RFC 8555，支持多 CA、多证书模板配置' },
  { icon: '🛒', title: '订单/支付', desc: '证书购买、订单管理、多支付插件、充值记录' },
  { icon: '🔐', title: 'TOTP 验证器', desc: '内置 TOTP/HOTP 验证器，存储和查看验证码' },
  { icon: '📊', title: 'CT 透明度', desc: '证书提交 CT 日志，支持 CT 查询和列表管理' },
  { icon: '🔑', title: 'OID 管理', desc: '自定义 OID，支持拓展密钥用途、主体字段、EV 声明' },
]

const certTypes = [
  { type: 'X.509', desc: 'TLS/SSL、代码签名、邮件加密、客户端认证', badge: 'badge-blue' },
  { type: 'GPG', desc: '邮件签名/加密、软件包签名（仅导入）', badge: 'badge-teal' },
  { type: 'SSH', desc: 'SSH 身份认证密钥（仅导入）', badge: 'badge-orange' },
]

const slotTypes = [
  {
    name: 'local',
    title: '本地智能卡',
    desc: '证书和私钥存储在本地数据库，私钥使用 AES-256 加密，临时密钥由用户密码保护',
    security: '中等',
    color: '#f6ad55',
  },
  {
    name: 'tpmv2',
    title: 'TPM2 智能卡',
    desc: '私钥由 TPM 内部产生的公钥加密，再由临时密钥二次加密，密钥不可从 TPM 导出',
    security: '高',
    color: '#4fd1c5',
  },
  {
    name: 'cloud',
    title: '云端智能卡',
    desc: '证书和私钥存储在云端，本地仅缓存公开信息，所有密钥操作通过云端 API 完成',
    security: '云端托管',
    color: '#63b3ed',
  },
]

export default function OverviewPage() {
  return (
    <DocPage
      title="功能概览"
      subtitle="OpenCert Manager 提供从密钥生成到证书颁发的完整 PKI 管理能力，覆盖云端服务、本地客户端和 PKCS#11 驱动三个层次"
      badge="Overview"
    >
      <h2>🗂️ 云端服务模块</h2>
      <p>云端服务（server-card）提供完整的 CA 管理和证书颁发能力，通过 REST API 对外提供服务。</p>
      <div className="card-grid">
        {modules.map((m) => (
          <div key={m.title} className="info-card">
            <div className="info-card-title">
              {m.icon} {m.title}
            </div>
            <div className="info-card-body">{m.desc}</div>
          </div>
        ))}
      </div>

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
              <td>{c.type === 'X.509' ? '✅' : '❌'}</td>
              <td>✅</td>
            </tr>
          ))}
        </tbody>
      </table>

      <h2>💳 智能卡类型</h2>
      <p>client-card 支持三种智能卡槽类型，后续可通过实现接口扩展新类型。</p>
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
            <div style={{ marginTop: '0.75rem', fontSize: '0.75rem', color: 'var(--color-text-muted)' }}>
              安全级别：<span style={{ color: s.color }}>{s.security}</span>
            </div>
          </div>
        ))}
      </div>

      <h2>🔐 密钥安全策略</h2>
      <p>根据密钥存储类型模板，支持三级安全策略：</p>
      <table>
        <thead>
          <tr>
            <th>安全级别</th>
            <th>存储方式</th>
            <th>可恢复</th>
            <th>云端备份</th>
          </tr>
        </thead>
        <tbody>
          <tr>
            <td><span className="badge badge-green">高安全性</span></td>
            <td>TPM 片上存储，密钥不可导出</td>
            <td>❌ 不可恢复</td>
            <td>❌</td>
          </tr>
          <tr>
            <td><span className="badge badge-orange">中安全性</span></td>
            <td>本地存储，TPM 片上密钥加密 + 云端公钥加密</td>
            <td>✅ 可恢复</td>
            <td>可选</td>
          </tr>
          <tr>
            <td><span className="badge badge-blue">低安全性</span></td>
            <td>本地存储，用户密码加密 + 云端公钥加密</td>
            <td>✅ 可恢复</td>
            <td>可选</td>
          </tr>
        </tbody>
      </table>

      <h2>🖥️ 本地客户端功能</h2>
      <p>client-card 是基于 Electron 的跨平台桌面应用，同时支持 Web 访问模式。</p>
      <ul>
        <li>用户登录与个人信息管理</li>
        <li>本地/TPM2/云端智能卡的 CRUD 管理</li>
        <li>证书导入、删除、查看详情</li>
        <li>从智能卡生成 CSR 并提交（驱动签名确保片上生成）</li>
        <li>TOTP 验证器：添加条目、查看实时验证码</li>
        <li>云端证书导入到本地智能卡</li>
        <li>系统托盘图标，支持暗黑/透明模式</li>
      </ul>

      <div className="callout callout-info">
        <span className="callout-icon">ℹ️</span>
        <div className="callout-body">
          <strong>多语言支持</strong>
          前端使用 React + Ant Design，支持中英文切换，支持暗黑模式和透明模式。
        </div>
      </div>
    </DocPage>
  )
}
