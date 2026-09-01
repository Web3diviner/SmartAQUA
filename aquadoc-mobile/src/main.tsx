import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { BrowserRouter } from 'react-router-dom'

import { App } from '@/app/router'
import { AppProviders } from '@/app/providers'
import '@/styles.css'

// Register Progressive Web App Service Worker for standalone mobile caching
if ('serviceWorker' in navigator) {
  window.addEventListener('load', () => {
    navigator.serviceWorker
      .register('/sw.js')
      .then((reg) => {
        console.log('[AquaDoc PWA] Service Worker active:', reg.scope)
      })
      .catch((err) => {
        console.warn('[AquaDoc PWA] Service Worker notice:', err)
      })
  })
}

const container = document.getElementById('root')
if (!container) throw new Error('Root element #root not found')

createRoot(container).render(
  <StrictMode>
    <BrowserRouter>
      <AppProviders>
        <App />
      </AppProviders>
    </BrowserRouter>
  </StrictMode>,
)
