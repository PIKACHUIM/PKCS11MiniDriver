import React from 'react'
import ReactDOM from 'react-dom/client'
import './i18n' // i18n 初始化必须在 App 之前
import App from './App.tsx'
import 'antd/dist/reset.css'
import './index.css'

ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <App />
  </React.StrictMode>,
)
