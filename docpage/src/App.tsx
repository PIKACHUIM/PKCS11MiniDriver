import { HashRouter, Routes, Route } from 'react-router-dom'
import Layout from './components/Layout'
import HomePage from './pages/HomePage'
import OverviewPage from './pages/OverviewPage'
import ArchitecturePage from './pages/ArchitecturePage'
import ApiPage from './pages/ApiPage'
import DriverPage from './pages/DriverPage'
import SecurityPage from './pages/SecurityPage'
import QuickStartPage from './pages/QuickStartPage'

export default function App() {
  return (
    <HashRouter>
      <Routes>
        <Route path="/" element={<Layout />}>
          <Route index element={<HomePage />} />
          <Route path="overview" element={<OverviewPage />} />
          <Route path="architecture" element={<ArchitecturePage />} />
          <Route path="quickstart" element={<QuickStartPage />} />
          <Route path="api" element={<ApiPage />} />
          <Route path="driver" element={<DriverPage />} />
          <Route path="security" element={<SecurityPage />} />
        </Route>
      </Routes>
    </HashRouter>
  )
}
