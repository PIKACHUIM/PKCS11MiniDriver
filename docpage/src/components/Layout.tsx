import { useState, useEffect } from 'react'
import { Outlet, NavLink, useLocation } from 'react-router-dom'
import './Layout.css'

const navItems = [
  { path: '/',            label: '首页',     icon: '⌂', exact: true },
  { path: '/overview',    label: '功能概览', icon: '◈' },
  { path: '/architecture',label: '系统架构', icon: '⬡' },
  { path: '/quickstart',  label: '快速开始', icon: '▶' },
  { path: '/api',         label: 'API 文档', icon: '⚡' },
  { path: '/driver',      label: 'PKCS#11 驱动', icon: '⚙' },
  { path: '/security',    label: '安全设计', icon: '⬡' },
]

export default function Layout() {
  const [sidebarOpen, setSidebarOpen] = useState(false)
  const location = useLocation()

  // 路由切换时关闭移动端侧边栏
  useEffect(() => {
    setSidebarOpen(false)
  }, [location.pathname])

  return (
    <div className="layout">
      {/* 顶部导航栏 */}
      <header className="header">
        <div className="header-inner">
          <div className="header-brand">
            <span className="brand-icon">🔐</span>
            <span className="brand-name">OpenCert</span>
            <span className="brand-tag">Manager</span>
          </div>
          <nav className="header-nav">
            <a
              href="https://github.com/PIKACHUIM/PKCS11MiniDriver"
              target="_blank"
              rel="noopener noreferrer"
              className="header-link"
            >
              <svg width="18" height="18" viewBox="0 0 24 24" fill="currentColor">
                <path d="M12 0C5.374 0 0 5.373 0 12c0 5.302 3.438 9.8 8.207 11.387.599.111.793-.261.793-.577v-2.234c-3.338.726-4.033-1.416-4.033-1.416-.546-1.387-1.333-1.756-1.333-1.756-1.089-.745.083-.729.083-.729 1.205.084 1.839 1.237 1.839 1.237 1.07 1.834 2.807 1.304 3.492.997.107-.775.418-1.305.762-1.604-2.665-.305-5.467-1.334-5.467-5.931 0-1.311.469-2.381 1.236-3.221-.124-.303-.535-1.524.117-3.176 0 0 1.008-.322 3.301 1.23A11.509 11.509 0 0 1 12 5.803c1.02.005 2.047.138 3.006.404 2.291-1.552 3.297-1.23 3.297-1.23.653 1.653.242 2.874.118 3.176.77.84 1.235 1.911 1.235 3.221 0 4.609-2.807 5.624-5.479 5.921.43.372.823 1.102.823 2.222v3.293c0 .319.192.694.801.576C20.566 21.797 24 17.3 24 12c0-6.627-5.373-12-12-12z"/>
              </svg>
              GitHub
            </a>
          </nav>
          <button
            className="hamburger"
            onClick={() => setSidebarOpen(!sidebarOpen)}
            aria-label="切换菜单"
          >
            <span className={`hamburger-line ${sidebarOpen ? 'open' : ''}`} />
            <span className={`hamburger-line ${sidebarOpen ? 'open' : ''}`} />
            <span className={`hamburger-line ${sidebarOpen ? 'open' : ''}`} />
          </button>
        </div>
      </header>

      <div className="layout-body">
        {/* 侧边栏遮罩 */}
        {sidebarOpen && (
          <div className="sidebar-overlay" onClick={() => setSidebarOpen(false)} />
        )}

        {/* 侧边导航 */}
        <aside className={`sidebar ${sidebarOpen ? 'sidebar-open' : ''}`}>
          <div className="sidebar-section-label">文档导航</div>
          <nav className="sidebar-nav">
            {navItems.map((item) => (
              <NavLink
                key={item.path}
                to={item.path}
                end={item.exact}
                className={({ isActive }) =>
                  `sidebar-link ${isActive ? 'sidebar-link-active' : ''}`
                }
              >
                <span className="sidebar-icon">{item.icon}</span>
                {item.label}
              </NavLink>
            ))}
          </nav>

          <div className="sidebar-footer">
          <div className="sidebar-version">
              <span className="badge badge-blue">v2.0.0</span>
              <span className="text-muted" style={{ fontSize: '0.75rem' }}>2026-04-17</span>
            </div>
          </div>
        </aside>

        {/* 主内容区 */}
        <main className="main-content">
          <Outlet />
        </main>
      </div>
    </div>
  )
}
