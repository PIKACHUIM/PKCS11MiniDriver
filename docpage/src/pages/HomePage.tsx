import { Link } from 'react-router-dom'
import './HomePage.css'

const features = [
  {
    icon: '🏛️',
    title: 'CA 证书管理',
    desc: '完整的证书颁发机构管理，支持多级 CA 链、CRL/OCSP 吊销服务、CT 透明度日志',
    badge: 'X.509',
    color: 'blue',
  },
  {
    icon: '💳',
    title: '智能卡管理',
    desc: '支持本地虚拟卡、TPM2 硬件卡、云端智能卡三种模式，密钥安全分级存储',
    badge: 'PKCS#11',
    color: 'teal',
  },
  {
    icon: '🔑',
    title: '多格式证书',
    desc: '支持 X.509、GPG、SSH 三种证书格式，覆盖 TLS、代码签名、身份认证等场景',
    badge: 'Multi-Format',
    color: 'orange',
  },
  {
    icon: '🛡️',
    title: '安全密钥存储',
    desc: 'TPM2 片上密钥、AES-256 加密、HMAC 派生密钥，三级安全策略保护私钥',
    badge: 'TPM2',
    color: 'green',
  },
  {
    icon: '⚡',
    title: 'ACME 自动化',
    desc: '内置 ACME 协议服务，支持自动证书申请、续期，兼容 Let\'s Encrypt 客户端',
    badge: 'RFC 8555',
    color: 'blue',
  },
  {
    icon: '🔐',
    title: 'TOTP/FIDO 认证',
    desc: '集成 TOTP/HOTP 验证器、FIDO2 认证，支持多因素身份验证',
    badge: 'MFA',
    color: 'teal',
  },
]

const components = [
  {
    name: 'pkcs11-mock',
    desc: 'PKCS#11 驱动层，注册到系统，提供标准密码学接口',
    lang: 'C',
    color: '#f6ad55',
  },
  {
    name: 'client-card',
    desc: '本地管理端，Electron 桌面应用，管理智能卡与证书',
    lang: 'Go + React',
    color: '#63b3ed',
  },
  {
    name: 'server-card',
    desc: '云端服务，提供 CA 管理、证书颁发、用户认证等 REST API',
    lang: 'Go',
    color: '#4fd1c5',
  },
]

export default function HomePage() {
  return (
    <div className="home">
      {/* Hero 区域 */}
      <section className="hero">
        <div className="hero-bg">
          <div className="hero-grid" />
          <div className="hero-glow hero-glow-1" />
          <div className="hero-glow hero-glow-2" />
        </div>
        <div className="hero-content">
          <div className="animate-fade-in-up delay-1">
            <div className="hero-eyebrow">
              <span className="badge badge-blue">v1.1.0</span>
              <span className="badge badge-teal">开源</span>
              <span className="badge badge-orange">企业级</span>
            </div>
          </div>
          <h1 className="hero-title animate-fade-in-up delay-2">
            <span className="hero-title-main">OpenCert</span>
            <br />
            <span className="hero-title-sub">Manager</span>
          </h1>
          <p className="hero-desc animate-fade-in-up delay-3">
            企业级 CA + 智能卡 + X.509 / GPG / SSH 证书管理平台
            <br />
            从密钥生成到证书颁发，从吊销管理到 PKCS#11 驱动，一站式解决方案
          </p>
          <div className="hero-actions animate-fade-in-up delay-4">
            <Link to="/quickstart" className="btn-primary">
              快速开始 →
            </Link>
            <Link to="/overview" className="btn-secondary">
              功能概览
            </Link>
            <a
              href="https://github.com/PIKACHUIM/PKCS11MiniDriver"
              target="_blank"
              rel="noopener noreferrer"
              className="btn-ghost"
            >
              <svg width="16" height="16" viewBox="0 0 24 24" fill="currentColor">
                <path d="M12 0C5.374 0 0 5.373 0 12c0 5.302 3.438 9.8 8.207 11.387.599.111.793-.261.793-.577v-2.234c-3.338.726-4.033-1.416-4.033-1.416-.546-1.387-1.333-1.756-1.333-1.756-1.089-.745.083-.729.083-.729 1.205.084 1.839 1.237 1.839 1.237 1.07 1.834 2.807 1.304 3.492.997.107-.775.418-1.305.762-1.604-2.665-.305-5.467-1.334-5.467-5.931 0-1.311.469-2.381 1.236-3.221-.124-.303-.535-1.524.117-3.176 0 0 1.008-.322 3.301 1.23A11.509 11.509 0 0 1 12 5.803c1.02.005 2.047.138 3.006.404 2.291-1.552 3.297-1.23 3.297-1.23.653 1.653.242 2.874.118 3.176.77.84 1.235 1.911 1.235 3.221 0 4.609-2.807 5.624-5.479 5.921.43.372.823 1.102.823 2.222v3.293c0 .319.192.694.801.576C20.566 21.797 24 17.3 24 12c0-6.627-5.373-12-12-12z"/>
              </svg>
              GitHub
            </a>
          </div>
        </div>
      </section>

      {/* 架构概览 */}
      <section className="section">
        <div className="section-inner">
          <h2 className="section-title animate-fade-in-up">系统架构</h2>
          <p className="section-desc animate-fade-in-up delay-1">
            三层架构设计，驱动层、客户端、云端服务各司其职
          </p>
          <div className="arch-flow animate-fade-in-up delay-2">
            {components.map((comp, i) => (
              <div key={comp.name} className="arch-flow-item">
                <div className="arch-card" style={{ '--accent': comp.color } as React.CSSProperties}>
                  <div className="arch-card-header">
                    <span className="arch-card-name">{comp.name}</span>
                    <span className="arch-card-lang">{comp.lang}</span>
                  </div>
                  <p className="arch-card-desc">{comp.desc}</p>
                </div>
                {i < components.length - 1 && (
                  <div className="arch-arrow">
                    <span>⟷</span>
                  </div>
                )}
              </div>
            ))}
          </div>
        </div>
      </section>

      {/* 功能特性 */}
      <section className="section section-alt">
        <div className="section-inner">
          <h2 className="section-title animate-fade-in-up">核心功能</h2>
          <p className="section-desc animate-fade-in-up delay-1">
            覆盖证书全生命周期管理的完整功能集
          </p>
          <div className="features-grid">
            {features.map((f, i) => (
              <div
                key={f.title}
                className={`feature-card animate-fade-in-up delay-${Math.min(i + 1, 6)}`}
              >
                <div className="feature-icon">{f.icon}</div>
                <div className="feature-body">
                  <div className="feature-header">
                    <h3 className="feature-title">{f.title}</h3>
                    <span className={`badge badge-${f.color}`}>{f.badge}</span>
                  </div>
                  <p className="feature-desc">{f.desc}</p>
                </div>
              </div>
            ))}
          </div>
        </div>
      </section>

      {/* 快速导航 */}
      <section className="section">
        <div className="section-inner">
          <h2 className="section-title animate-fade-in-up">开始探索</h2>
          <div className="nav-cards animate-fade-in-up delay-1">
            <Link to="/quickstart" className="nav-card">
              <span className="nav-card-icon">▶</span>
              <div>
                <div className="nav-card-title">快速开始</div>
                <div className="nav-card-desc">5 分钟部署运行</div>
              </div>
            </Link>
            <Link to="/architecture" className="nav-card">
              <span className="nav-card-icon">⬡</span>
              <div>
                <div className="nav-card-title">系统架构</div>
                <div className="nav-card-desc">了解设计原理</div>
              </div>
            </Link>
            <Link to="/api" className="nav-card">
              <span className="nav-card-icon">⚡</span>
              <div>
                <div className="nav-card-title">API 文档</div>
                <div className="nav-card-desc">REST API 参考</div>
              </div>
            </Link>
            <Link to="/driver" className="nav-card">
              <span className="nav-card-icon">⚙</span>
              <div>
                <div className="nav-card-title">PKCS#11 驱动</div>
                <div className="nav-card-desc">驱动集成指南</div>
              </div>
            </Link>
          </div>
        </div>
      </section>

      {/* Footer */}
      <footer className="footer">
        <div className="footer-inner">
          <span className="text-muted">OpenCert Manager</span>
          <span className="text-muted">·</span>
          <a
            href="https://github.com/PIKACHUIM/PKCS11MiniDriver"
            target="_blank"
            rel="noopener noreferrer"
            className="text-secondary"
          >
            PIKACHUIM/PKCS11MiniDriver
          </a>
          <span className="text-muted">·</span>
          <span className="text-muted">MIT License</span>
        </div>
      </footer>
    </div>
  )
}
