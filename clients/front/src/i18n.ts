import i18n from 'i18next';
import { initReactI18next } from 'react-i18next';
import zhCN from './locales/zh-CN.json';
import enUS from './locales/en-US.json';

// 从 localStorage 读取用户语言偏好，默认中文
const savedLang = localStorage.getItem('opencert-lang') || 'zh-CN';

i18n.use(initReactI18next).init({
  resources: {
    'zh-CN': { translation: zhCN },
    'en-US': { translation: enUS },
  },
  lng: savedLang,
  fallbackLng: 'zh-CN',
  interpolation: {
    escapeValue: false, // React 已自动转义
  },
});

// 语言切换时保存到 localStorage
i18n.on('languageChanged', (lng: string) => {
  localStorage.setItem('opencert-lang', lng);
});

export default i18n;
