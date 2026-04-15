import './DocPage.css'

interface DocPageProps {
  title: string
  subtitle?: string
  badge?: string
  children: React.ReactNode
}

export default function DocPage({ title, subtitle, badge, children }: DocPageProps) {
  return (
    <div className="doc-page">
      <div className="doc-header">
        <div className="doc-header-inner">
          {badge && <span className="badge badge-blue">{badge}</span>}
          <h1 className="doc-title">{title}</h1>
          {subtitle && <p className="doc-subtitle">{subtitle}</p>}
        </div>
      </div>
      <div className="doc-body">
        <div className="doc-content">
          {children}
        </div>
      </div>
    </div>
  )
}
