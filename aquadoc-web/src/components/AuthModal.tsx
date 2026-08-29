import React, { useState } from 'react'
import { useAppState } from '@/app/providers'
import { SignupFormData } from '@/types/auth'

interface GoogleAccountOption {
  email: string
  name: string
  avatar: string
}

const DEFAULT_BROWSER_GOOGLE_ACCOUNTS: GoogleAccountOption[] = [
  {
    name: 'Babatunde Alabi',
    email: 'babatunde.farm@gmail.com',
    avatar: 'https://api.dicebear.com/7.x/bottts/svg?seed=babatunde.farm@gmail.com',
  },
  {
    name: 'Dr. Fish Specialist',
    email: 'dr.fish.vet@gmail.com',
    avatar: 'https://api.dicebear.com/7.x/bottts/svg?seed=dr.fish.vet@gmail.com',
  },
  {
    name: 'Adeleke Integrated Farm',
    email: 'adeleke.aquaculture@gmail.com',
    avatar: 'https://api.dicebear.com/7.x/bottts/svg?seed=adeleke.aquaculture@gmail.com',
  },
]

export const AuthModal: React.FC = () => {
  const { isAuthModalOpen, closeAuthModal, login, loginWithGoogle, signup } = useAppState()
  const [tab, setTab] = useState<'signin' | 'signup'>('signin')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  // Google Account Chooser State
  const [showGoogleChooser, setShowGoogleChooser] = useState(false)
  const [customGmail, setCustomGmail] = useState('')
  const [customGmailName, setCustomGmailName] = useState('')
  const [showCustomGmailInput, setShowCustomGmailInput] = useState(false)

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
    farmLocation: 'Lagos State',
    primarySpecies: 'African Catfish (Clarias gariepinus)',
    farmingSystem: 'Concrete Tanks',
  })

  if (!isAuthModalOpen) return null

  const handleOpenGoogleChooser = () => {
    setError(null)
    setShowGoogleChooser(true)
  }

  const handleSelectGoogleAccount = async (account: GoogleAccountOption) => {
    setLoading(true)
    setError(null)
    const res = await loginWithGoogle(account.email, account.name)
    setLoading(false)
    if (res.success) {
      setShowGoogleChooser(false)
    } else {
      setError(res.error || 'Google login failed')
    }
  }

  const handleCustomGmailSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!customGmail) return
    const formattedEmail = customGmail.includes('@') ? customGmail : `${customGmail}@gmail.com`
    const displayName = customGmailName || formattedEmail.split('@')[0]!.replace('.', ' ').replace(/\b\w/g, (l) => l.toUpperCase())

    setLoading(true)
    setError(null)
    const res = await loginWithGoogle(formattedEmail, displayName)
    setLoading(false)
    if (res.success) {
      setShowGoogleChooser(false)
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

        {/* GOOGLE ACCOUNT CHOOSER SCREEN */}
        {showGoogleChooser ? (
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
              <h3>Choose an account</h3>
              <p>to continue to <strong>AquaDoc AI</strong></p>
            </div>

            {error && <div className="auth-error-alert">{error}</div>}

            <div className="google-accounts-list">
              {DEFAULT_BROWSER_GOOGLE_ACCOUNTS.map((acc) => (
                <button
                  key={acc.email}
                  type="button"
                  className="google-account-row"
                  onClick={() => handleSelectGoogleAccount(acc)}
                  disabled={loading}
                >
                  <img src={acc.avatar} alt={acc.name} className="google-account-avatar" />
                  <div className="google-account-text">
                    <strong className="google-account-name">{acc.name}</strong>
                    <span className="google-account-email">{acc.email}</span>
                  </div>
                  <span className="google-account-arrow">→</span>
                </button>
              ))}

              {/* Use Another Account Button */}
              {!showCustomGmailInput ? (
                <button
                  type="button"
                  className="google-account-row google-account-row--add"
                  onClick={() => setShowCustomGmailInput(true)}
                >
                  <span className="google-add-icon">👤+</span>
                  <div className="google-account-text">
                    <strong>Use another Google account</strong>
                    <span className="google-account-email">Sign in with a different Gmail address</span>
                  </div>
                </button>
              ) : (
                <form className="custom-gmail-form" onSubmit={handleCustomGmailSubmit}>
                  <div className="form-group">
                    <label>Gmail Address</label>
                    <input
                      type="email"
                      required
                      placeholder="e.g. yourname@gmail.com"
                      value={customGmail}
                      onChange={(e) => setCustomGmail(e.target.value)}
                      className="input"
                      autoFocus
                    />
                  </div>
                  <div className="form-group">
                    <label>Your Name / Farm Name (Optional)</label>
                    <input
                      type="text"
                      placeholder="e.g. Engr. Nnamdi Eze"
                      value={customGmailName}
                      onChange={(e) => setCustomGmailName(e.target.value)}
                      className="input"
                    />
                  </div>
                  <button type="submit" className="button button--primary auth-submit-btn" disabled={loading}>
                    {loading ? 'Authenticating with Google…' : 'Sign in with this Gmail Account'}
                  </button>
                </form>
              )}
            </div>

            <div className="google-chooser-footer">
              <button
                type="button"
                className="btn-back-link"
                onClick={() => setShowGoogleChooser(false)}
              >
                ← Back to email sign in
              </button>
            </div>
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

            {/* 1-Click Google Sign In (Opens Account Chooser) */}
            <button
              type="button"
              className="google-auth-btn"
              onClick={handleOpenGoogleChooser}
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
              <span>or sign in with email</span>
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
                    placeholder="e.g. farmer@gmail.com"
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
                      placeholder="e.g. Babatunde Alabi"
                      value={signupForm.name}
                      onChange={(e) => setSignupForm({ ...signupForm, name: e.target.value })}
                      className="input"
                    />
                  </div>

                  <div className="form-group">
                    <label>Email (Gmail / Work) *</label>
                    <input
                      type="email"
                      required
                      placeholder="e.g. alabi.farm@gmail.com"
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
                      placeholder="e.g. Alabi Integrated Fishery"
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
