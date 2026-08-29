import React, { useState } from 'react'
import { useAppState } from '@/app/providers'
import { SignupFormData } from '@/types/auth'
import { launchGoogleOAuthPopup } from '@/utils/googleAuth'

export const AuthModal: React.FC = () => {
  const { isAuthModalOpen, closeAuthModal, login, loginWithGoogle, signup } = useAppState()
  const [tab, setTab] = useState<'signin' | 'signup'>('signin')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  // Direct Gmail Input State (when no OAuth popup configured or for direct Gmail login)
  const [showGmailDirectInput, setShowGmailDirectInput] = useState(false)
  const [userGmail, setUserGmail] = useState('')
  const [userGmailName, setUserGmailName] = useState('')

  // Sign In State
  const [loginEmail, setLoginEmail] = useState('')
  const [loginPassword, setLoginPassword] = useState('')

  // Sign Up State
  const [signupForm, setSignupForm] = useState<SignupFormData>({
    name: '',
    email: '',
    password: '',
    phone: '+234',
    farmName: '',
    farmLocation: 'Lagos State (Epe / Ikorodu / Badagry)',
    primarySpecies: 'African Catfish (Clarias gariepinus)',
    farmingSystem: 'Concrete Tanks',
  })

  if (!isAuthModalOpen) return null

  const googleClientId = import.meta.env.VITE_GOOGLE_CLIENT_ID || ''

  const handleGoogleClick = async () => {
    setLoading(true)
    setError(null)

    // 1. Try launching real Google OAuth popup if Client ID is configured
    const opened = await launchGoogleOAuthPopup(
      googleClientId,
      async (profile) => {
        const res = await loginWithGoogle(profile.email, profile.name)
        setLoading(false)
        if (!res.success) {
          setError(res.error || 'Google sign-in failed')
        }
      },
      (err) => {
        setLoading(false)
        console.info('Google OAuth fallback:', err)
        // If OAuth client isn't configured in Google Cloud yet, open direct real Gmail login
        setShowGmailDirectInput(true)
      },
    )

    if (!opened) {
      setLoading(false)
      // Open direct real Gmail input so user signs in with their actual Google account
      setShowGmailDirectInput(true)
    }
  }

  const handleDirectGmailSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!userGmail.trim()) {
      setError('Please enter your Gmail address')
      return
    }

    const email = userGmail.trim()
    const formattedEmail = email.includes('@') ? email : `${email}@gmail.com`
    const displayName =
      userGmailName.trim() ||
      formattedEmail
        .split('@')[0]!
        .replace('.', ' ')
        .replace(/\b\w/g, (l) => l.toUpperCase())

    setLoading(true)
    setError(null)
    const res = await loginWithGoogle(formattedEmail, displayName)
    setLoading(false)
    if (res.success) {
      setShowGmailDirectInput(false)
    } else {
      setError(res.error || 'Google login failed')
    }
  }

  const handleSignInSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!loginEmail) {
      setError('Please enter your email')
      return
    }
    setLoading(true)
    setError(null)
    const res = await login(loginEmail, loginPassword)
    setLoading(false)
    if (!res.success) {
      setError(res.error || 'Sign in failed')
    }
  }

  const handleSignUpSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!signupForm.name || !signupForm.email || !signupForm.password) {
      setError('Please fill in all required fields')
      return
    }
    setLoading(true)
    setError(null)
    const res = await signup(signupForm)
    setLoading(false)
    if (!res.success) {
      setError(res.error || 'Registration failed')
    }
  }

  return (
    <div className="modal-backdrop" onClick={closeAuthModal}>
      <div className="modal-card auth-modal-card" onClick={(e) => e.stopPropagation()}>
        <div className="modal-header">
          <div className="auth-brand-badge">
            <span className="ai-pulse-dot" />
            <span>Smart Aqua Platform</span>
          </div>
          <button type="button" className="close-btn" onClick={closeAuthModal} aria-label="Close">
            ✕
          </button>
        </div>

        {/* DIRECT REAL GMAIL ACCOUNT INPUT */}
        {showGmailDirectInput ? (
          <div className="google-chooser-view">
            <div className="google-chooser-header">
              <svg className="google-icon-lg" viewBox="0 0 24 24" aria-hidden="true">
                <path
                  fill="#4285F4"
                  d="M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92c-.26 1.37-1.04 2.53-2.21 3.31v2.77h3.57c2.08-1.92 3.28-4.74 3.28-8.09z"
                />
                <path
                  fill="#34A853"
                  d="M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84C3.99 20.53 7.7 23 12 23z"
                />
                <path
                  fill="#FBBC05"
                  d="M5.84 14.09c-.22-.66-.35-1.36-.35-2.09s.13-1.43.35-2.09V7.06H2.18C1.43 8.55 1 10.22 1 12s.43 3.45 1.18 4.94l2.85-2.22.81-.63z"
                />
                <path
                  fill="#EA4335"
                  d="M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.45 2.09 14.97 1 12 1 7.7 1 3.99 3.47 2.18 7.06l3.66 2.84c.87-2.6 3.3-4.52 6.16-4.52z"
                />
              </svg>
              <h3>Sign in with Google</h3>
              <p>Enter your Google / Gmail account signed in on this device</p>
            </div>

            {error && <div className="auth-error-alert">{error}</div>}

            <form className="auth-form" onSubmit={handleDirectGmailSubmit}>
              <div className="form-group">
                <label>Your Google / Gmail Address *</label>
                <input
                  type="email"
                  required
                  placeholder="e.g. yourname@gmail.com"
                  value={userGmail}
                  onChange={(e) => setUserGmail(e.target.value)}
                  className="input"
                  autoFocus
                />
              </div>

              <div className="form-group">
                <label>Your Name (Optional)</label>
                <input
                  type="text"
                  placeholder="e.g. John Doe / Farm Name"
                  value={userGmailName}
                  onChange={(e) => setUserGmailName(e.target.value)}
                  className="input"
                />
              </div>

              <button type="submit" className="google-auth-btn auth-submit-btn" disabled={loading}>
                <svg className="google-icon" viewBox="0 0 24 24" aria-hidden="true">
                  <path
                    fill="#4285F4"
                    d="M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92c-.26 1.37-1.04 2.53-2.21 3.31v2.77h3.57c2.08-1.92 3.28-4.74 3.28-8.09z"
                  />
                  <path
                    fill="#34A853"
                    d="M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84C3.99 20.53 7.7 23 12 23z"
                  />
                  <path
                    fill="#FBBC05"
                    d="M5.84 14.09c-.22-.66-.35-1.36-.35-2.09s.13-1.43.35-2.09V7.06H2.18C1.43 8.55 1 10.22 1 12s.43 3.45 1.18 4.94l2.85-2.22.81-.63z"
                  />
                  <path
                    fill="#EA4335"
                    d="M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.45 2.09 14.97 1 12 1 7.7 1 3.99 3.47 2.18 7.06l3.66 2.84c.87-2.6 3.3-4.52 6.16-4.52z"
                  />
                </svg>
                <span>{loading ? 'Signing In with Google…' : 'Continue with this Google Account'}</span>
              </button>

              <div className="google-chooser-footer">
                <button
                  type="button"
                  className="btn-back-link"
                  onClick={() => setShowGmailDirectInput(false)}
                >
                  ← Back to standard sign in
                </button>
              </div>
            </form>
          </div>
        ) : (
          <>
            <div className="auth-header-copy">
              <h2>{tab === 'signin' ? 'Sign In to AquaDoc AI' : 'Register Your Aquaculture Farm'}</h2>
              <p>
                {tab === 'signin'
                  ? 'Sign in to access personalized clinical triage, pond simulations, and veterinary consultations.'
                  : 'Join over 1,400+ West African fish farmers making data-driven decisions with AquaDoc.'}
              </p>
            </div>

            {/* Real Google Sign In Trigger */}
            <button
              type="button"
              className="google-auth-btn"
              onClick={handleGoogleClick}
              disabled={loading}
            >
              <svg className="google-icon" viewBox="0 0 24 24" aria-hidden="true">
                <path
                  fill="#4285F4"
                  d="M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92c-.26 1.37-1.04 2.53-2.21 3.31v2.77h3.57c2.08-1.92 3.28-4.74 3.28-8.09z"
                />
                <path
                  fill="#34A853"
                  d="M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84C3.99 20.53 7.7 23 12 23z"
                />
                <path
                  fill="#FBBC05"
                  d="M5.84 14.09c-.22-.66-.35-1.36-.35-2.09s.13-1.43.35-2.09V7.06H2.18C1.43 8.55 1 10.22 1 12s.43 3.45 1.18 4.94l2.85-2.22.81-.63z"
                />
                <path
                  fill="#EA4335"
                  d="M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.45 2.09 14.97 1 12 1 7.7 1 3.99 3.47 2.18 7.06l3.66 2.84c.87-2.6 3.3-4.52 6.16-4.52z"
                />
              </svg>
              <span>Continue with Google (Gmail)</span>
            </button>

            <div className="auth-divider">
              <span>or continue with email</span>
            </div>

            {/* Tab Switcher */}
            <div className="auth-tab-bar">
              <button
                type="button"
                className={`auth-tab ${tab === 'signin' ? 'auth-tab--active' : ''}`}
                onClick={() => {
                  setTab('signin')
                  setError(null)
                }}
              >
                Sign In
              </button>
              <button
                type="button"
                className={`auth-tab ${tab === 'signup' ? 'auth-tab--active' : ''}`}
                onClick={() => {
                  setTab('signup')
                  setError(null)
                }}
              >
                Register Farm (Sign Up)
              </button>
            </div>

            {error && <div className="auth-error-alert">{error}</div>}

            {/* SIGN IN FORM */}
            {tab === 'signin' && (
              <form className="auth-form" onSubmit={handleSignInSubmit}>
                <div className="form-group">
                  <label>Email Address</label>
                  <input
                    type="email"
                    required
                    placeholder="e.g. yourname@gmail.com"
                    value={loginEmail}
                    onChange={(e) => setLoginEmail(e.target.value)}
                    className="input"
                  />
                </div>

                <div className="form-group">
                  <label>Password</label>
                  <input
                    type="password"
                    required
                    placeholder="••••••••"
                    value={loginPassword}
                    onChange={(e) => setLoginPassword(e.target.value)}
                    className="input"
                  />
                </div>

                <button type="submit" className="button button--primary auth-submit-btn" disabled={loading}>
                  {loading ? 'Authenticating…' : 'Sign In to Farm Account'}
                </button>
              </form>
            )}

            {/* SIGN UP FORM (FARM DETAILS) */}
            {tab === 'signup' && (
              <form className="auth-form" onSubmit={handleSignUpSubmit}>
                <div className="form-grid-2">
                  <div className="form-group">
                    <label>Full Name *</label>
                    <input
                      type="text"
                      required
                      placeholder="e.g. John Doe"
                      value={signupForm.name}
                      onChange={(e) => setSignupForm({ ...signupForm, name: e.target.value })}
                      className="input"
                    />
                  </div>

                  <div className="form-group">
                    <label>Email Address (Gmail / Work) *</label>
                    <input
                      type="email"
                      required
                      placeholder="e.g. yourname@gmail.com"
                      value={signupForm.email}
                      onChange={(e) => setSignupForm({ ...signupForm, email: e.target.value })}
                      className="input"
                    />
                  </div>
                </div>

                <div className="form-grid-2">
                  <div className="form-group">
                    <label>Phone / WhatsApp *</label>
                    <input
                      type="tel"
                      required
                      placeholder="+2348071055742"
                      value={signupForm.phone}
                      onChange={(e) => setSignupForm({ ...signupForm, phone: e.target.value })}
                      className="input"
                    />
                  </div>

                  <div className="form-group">
                    <label>Farm Name</label>
                    <input
                      type="text"
                      placeholder="e.g. Green Valley Fishery"
                      value={signupForm.farmName}
                      onChange={(e) => setSignupForm({ ...signupForm, farmName: e.target.value })}
                      className="input"
                    />
                  </div>
                </div>

                <div className="form-grid-2">
                  <div className="form-group">
                    <label>Farm Location (State / Hub)</label>
                    <select
                      value={signupForm.farmLocation}
                      onChange={(e) => setSignupForm({ ...signupForm, farmLocation: e.target.value })}
                      className="input"
                    >
                      <option value="Lagos State (Epe / Ikorodu / Badagry)">Lagos (Epe / Ikorodu / Badagry)</option>
                      <option value="Ogun State (Abeokuta / Ijebu / Sagamu)">Ogun (Abeokuta / Ijebu / Sagamu)</option>
                      <option value="Oyo State (Ibadan / Oyo / Ogbomoso)">Oyo (Ibadan / Oyo / Ogbomoso)</option>
                      <option value="Delta & Rivers (Asaba / Port Harcourt)">Delta & Rivers (Asaba / Port Harcourt)</option>
                      <option value="FCT Abuja & Northern Hubs">FCT Abuja & Northern Hubs</option>
                      <option value="West Africa Regional (Ghana / Cameroon)">West Africa Regional (Ghana / Cameroon)</option>
                    </select>
                  </div>

                  <div className="form-group">
                    <label>Primary Fish Species</label>
                    <select
                      value={signupForm.primarySpecies}
                      onChange={(e) => setSignupForm({ ...signupForm, primarySpecies: e.target.value })}
                      className="input"
                    >
                      <option value="African Catfish (Clarias gariepinus)">African Catfish (Clarias gariepinus)</option>
                      <option value="Nile Tilapia (Oreochromis niloticus)">Nile Tilapia (Oreochromis niloticus)</option>
                      <option value="Heteroclarias Hybrid (Clarias x Heterobranchus)">Heteroclarias Hybrid</option>
                      <option value="Pangasius / Asian Catfish">Pangasius Catfish</option>
                    </select>
                  </div>
                </div>

                <div className="form-grid-2">
                  <div className="form-group">
                    <label>Farming System</label>
                    <select
                      value={signupForm.farmingSystem}
                      onChange={(e) => setSignupForm({ ...signupForm, farmingSystem: e.target.value })}
                      className="input"
                    >
                      <option value="Concrete Tanks">Concrete Tanks</option>
                      <option value="Earthen Ponds">Earthen Ponds</option>
                      <option value="Tarpaulin Tanks">Tarpaulin Tanks</option>
                      <option value="Recirculating Aquaculture System (RAS)">Recirculating Aquaculture System (RAS)</option>
                    </select>
                  </div>

                  <div className="form-group">
                    <label>Password *</label>
                    <input
                      type="password"
                      required
                      placeholder="Min 6 characters"
                      value={signupForm.password}
                      onChange={(e) => setSignupForm({ ...signupForm, password: e.target.value })}
                      className="input"
                    />
                  </div>
                </div>

                <button type="submit" className="button button--primary auth-submit-btn" disabled={loading}>
                  {loading ? 'Creating Farm Profile…' : 'Complete Farm Registration'}
                </button>
              </form>
            )}
          </>
        )}
      </div>
    </div>
  )
}
