import React, { useEffect, useState } from 'react'

interface BeforeInstallPromptEvent extends Event {
  prompt: () => Promise<void>
  userChoice: Promise<{ outcome: 'accepted' | 'dismissed'; platform: string }>
}

export const InstallPwaModal: React.FC = () => {
  const [deferredPrompt, setDeferredPrompt] = useState<BeforeInstallPromptEvent | null>(null)
  const [isModalOpen, setIsModalOpen] = useState(false)
  const [isIOS, setIsIOS] = useState(false)
  const [isStandalone, setIsStandalone] = useState(false)

  useEffect(() => {
    // Check if already running in standalone PWA / Home-Screen mode
    const standaloneMode =
      window.matchMedia('(display-mode: standalone)').matches ||
      (window.navigator as unknown as { standalone?: boolean }).standalone === true
    setIsStandalone(Boolean(standaloneMode))

    // Detect iOS
    const userAgent = window.navigator.userAgent.toLowerCase()
    const isAppleDevice = /iphone|ipad|ipod/.test(userAgent) && !(window as unknown as { MSStream?: unknown }).MSStream
    setIsIOS(isAppleDevice)

    // Capture Android / Chrome beforeinstallprompt event
    const handleBeforeInstallPrompt = (e: Event) => {
      e.preventDefault()
      setDeferredPrompt(e as BeforeInstallPromptEvent)
    }

    window.addEventListener('beforeinstallprompt', handleBeforeInstallPrompt)

    return () => {
      window.removeEventListener('beforeinstallprompt', handleBeforeInstallPrompt)
    }
  }, [])

  const handleInstallClick = async () => {
    if (deferredPrompt) {
      await deferredPrompt.prompt()
      const { outcome } = await deferredPrompt.userChoice
      if (outcome === 'accepted') {
        setDeferredPrompt(null)
      }
      setIsModalOpen(false)
    } else {
      setIsModalOpen(true)
    }
  }

  // If already installed and running standalone, do not render banner
  if (isStandalone) return null

  return (
    <>
      {/* Install Button Trigger (Shown in Header / Nav) */}
      <button
        type="button"
        className="pwa-install-nav-btn"
        onClick={handleInstallClick}
        title="Install AquaDoc App on your device"
      >
        <span className="pwa-icon">📲</span>
        <span className="pwa-text">Install App</span>
      </button>

      {/* Instruction Modal (Especially for iOS or Manual Installation) */}
      {isModalOpen && (
        <div className="pwa-modal-backdrop" onClick={() => setIsModalOpen(false)}>
          <div className="pwa-modal-card" onClick={(e) => e.stopPropagation()}>
            <div className="pwa-modal-header">
              <div className="pwa-brand-icon">
                <img src="/icons/icon.svg" alt="AquaDoc Icon" width="48" height="48" />
              </div>
              <div>
                <h3>Install AquaDoc Mobile</h3>
                <p>Run full-screen with offline support & instant access.</p>
              </div>
              <button
                type="button"
                className="pwa-close-btn"
                onClick={() => setIsModalOpen(false)}
                aria-label="Close"
              >
                ✕
              </button>
            </div>

            <div className="pwa-modal-body">
              {isIOS ? (
                <div className="pwa-steps-list">
                  <div className="pwa-step-item">
                    <span className="step-num">1</span>
                    <div className="step-content">
                      <strong>Tap the Share Button</strong>
                      <p>Tap the <strong>Share</strong> icon (📤) in your Safari toolbar at the bottom.</p>
                    </div>
                  </div>
                  <div className="pwa-step-item">
                    <span className="step-num">2</span>
                    <div className="step-content">
                      <strong>Add to Home Screen</strong>
                      <p>Scroll down and tap <strong>"Add to Home Screen"</strong> (➕).</p>
                    </div>
                  </div>
                  <div className="pwa-step-item">
                    <span className="step-num">3</span>
                    <div className="step-content">
                      <strong>Tap "Add"</strong>
                      <p>Tap <strong>Add</strong> in the top-right corner to place the AquaDoc App on your phone.</p>
                    </div>
                  </div>
                </div>
              ) : (
                <div className="pwa-android-info">
                  <p>
                    Install AquaDoc directly onto your phone or desktop for the best native app experience:
                  </p>
                  <ul>
                    <li>🚀 Instant loading & offline cache</li>
                    <li>📱 Zero browser URL bars / Full-screen mode</li>
                    <li>⚡ 1-Tap home screen access</li>
                  </ul>
                  {deferredPrompt && (
                    <button
                      type="button"
                      className="pwa-primary-install-btn"
                      onClick={handleInstallClick}
                    >
                      📲 Add to Home Screen Now
                    </button>
                  )}
                </div>
              )}
            </div>

            <div className="pwa-modal-footer">
              <button
                type="button"
                className="pwa-dismiss-btn"
                onClick={() => setIsModalOpen(false)}
              >
                Got It
              </button>
            </div>
          </div>
        </div>
      )}
    </>
  )
}
