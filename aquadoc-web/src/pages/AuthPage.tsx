import React, { useState, useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { useAppState } from '@/app/providers'
import { SignupFormData } from '@/types/auth'
import { launchGoogleOAuthPopup } from '@/utils/googleAuth'

export const AuthPage: React.FC = () => {
  const { user, isAuthenticated, login, loginWithGoogle, signup } = useAppState()
  const navigate = useNavigate()
  const [authMode, setAuthMode] = useState<'signup' | 'signin'>('signup')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  // Direct Gmail Input (if popup blocked or offline)
  const [showDirectGmail, setShowDirectGmail] = useState(false)
  const [directGmail, setDirectGmail] = useState('')

  // Sign In State
  const [loginEmail, setLoginEmail] = useState('')
  const [loginPassword, setLoginPassword] = useState('')

  // Manual Sign Up Form State
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

  const googleClientId = import.meta.env.VITE_GOOGLE_CLIENT_ID || ''

  // If already authenticated, redirect to chat
  useEffect(() => {
    if (isAuthenticated) {
      navigate('/chat', { replace: true })
    }
  }, [isAuthenticated, navigate])

  const handleGoogleClick = async () => {
    setLoading(true)
    setError(null)

    const opened = await launchGoogleOAuthPopup(
      googleClientId,
      async (profile) => {
        const res = await loginWithGoogle(profile.email, profile.name)
        setLoading(false)
        if (res.success) {
          navigate('/chat', { replace: true })
        } else {
          setError(res.error || 'Google sign-in failed')
        }
      },
      (err) => {
        setLoading(false)
        console.info('Google popup fallback:', err)
        setShowDirectGmail(true)
      },
    )

    if (!opened) {
      setLoading(false)
      setShowDirectGmail(true)
    }
  }

  const handleDirectGmailSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!directGmail.trim()) return
    const formattedEmail = directGmail.includes('@') ? directGmail.trim() : `${directGmail.trim()}@gmail.com`
    const displayName = formattedEmail
      .split('@')[0]!
      .replace('.', ' ')
      .replace(/\b\w/g, (l) => l.toUpperCase())

    setLoading(true)
    setError(null)
    const res = await loginWithGoogle(formattedEmail, displayName)
    setLoading(false)
    if (res.success) {
      navigate('/chat', { replace: true })
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
    if (res.success) {
      navigate('/chat', { replace: true })
    } else {
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
    if (res.success) {
      navigate('/chat', { replace: true })
    } else {
      setError(res.error || 'Registration failed')
    }
  }

  if (user) return null

  return (
    <div className="auth-page-container">
      <div className="auth-page-card">
        <div className="auth-brand-badge">
          <span className="ai-pulse-dot" />
          <span>Smart Aqua Ecosystem</span>
        </div>

        <div className="auth-header-copy">
          <h2>Get Started with AquaDoc AI</h2>
          <p>Choose Google sign-in or create your manual farm account to continue.</p>
        </div>

        {error && <div className="auth-error-alert">{error}</div>}

        {/* 1. GOOGLE SIGN-IN OPTION */}
        <div className="google-auth-section">
          {/* Google SSO Action Button */}
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
            <span>Continue with Google</span>
          </button>
        </div>

        {/* DIRECT GMAIL FORM IF PROMPTED */}
        {showDirectGmail && (
          <form className="custom-gmail-form" onSubmit={handleDirectGmailSubmit}>
            <div className="form-group">
              <label>Enter your Google / Gmail account *</label>
              <input
                type="email"
                required
                placeholder="yourname@gmail.com"
                value={directGmail}
                onChange={(e) => setDirectGmail(e.target.value)}
                className="input"
                autoFocus
              />
            </div>
            <button type="submit" className="button button--primary auth-submit-btn" disabled={loading}>
              {loading ? 'Signing in with Google…' : 'Sign in with this Gmail'}
            </button>
          </form>
        )}

        <div className="auth-divider">
          <span>or create an account manually</span>
        </div>

        {/* 2. MANUAL SIGN UP / SIGN IN TABS */}
        <div className="auth-tab-bar">
          <button
            type="button"
            className={`auth-tab ${authMode === 'signup' ? 'auth-tab--active' : ''}`}
            onClick={() => {
              setAuthMode('signup')
              setError(null)
            }}
          >
            Sign Up (Manual Registration)
          </button>
          <button
            type="button"
            className={`auth-tab ${authMode === 'signin' ? 'auth-tab--active' : ''}`}
            onClick={() => {
              setAuthMode('signin')
              setError(null)
            }}
          >
            Sign In with Email
          </button>
        </div>

        {/* MANUAL SIGN UP FORM */}
        {authMode === 'signup' && (
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
                <label>Email Address *</label>
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
                <label>Farm Location (State / Region)</label>
                <select
                  value={signupForm.farmLocation}
                  onChange={(e) => setSignupForm({ ...signupForm, farmLocation: e.target.value })}
                  className="input"
                >
                  <option value="Lagos State (Epe / Ikorodu / Badagry)">Lagos (Epe, Ikorodu, Badagry)</option>
                  <option value="Ogun State (Abeokuta / Ijebu / Sagamu)">Ogun (Abeokuta, Sagamu, Ijebu)</option>
                  <option value="Oyo State (Ibadan / Oyo / Ogbomoso)">Oyo (Ibadan, Ogbomoso)</option>
                  <option value="Delta & Rivers (Asaba / Port Harcourt)">Delta & Rivers (Port Harcourt, Asaba)</option>
                  <option value="FCT Abuja & Northern Hubs">Abuja & Northern Hubs</option>
                  <option value="West Africa Regional (Ghana / Cameroon)">West Africa (Ghana, Cameroon)</option>
                </select>
              </div>

              <div className="form-group">
                <label>Primary Fish Species</label>
                <select
                  value={signupForm.primarySpecies}
                  onChange={(e) => setSignupForm({ ...signupForm, primarySpecies: e.target.value })}
                  className="input"
                >
                  <option value="African Catfish (Clarias gariepinus)">African Catfish (Clarias)</option>
                  <option value="Nile Tilapia (Oreochromis niloticus)">Nile Tilapia (Oreochromis)</option>
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
                  <option value="Recirculating Aquaculture System (RAS)">RAS (Recirculating System)</option>
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
              {loading ? 'Registering Farm Profile…' : 'Create Farm Account (Sign Up)'}
            </button>
          </form>
        )}

        {/* MANUAL SIGN IN FORM */}
        {authMode === 'signin' && (
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
              {loading ? 'Signing In…' : 'Sign In to Account'}
            </button>
          </form>
        )}
      </div>
    </div>
  )
}
