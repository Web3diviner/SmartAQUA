import React, { useState, useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { useAppState } from '@/app/providers'
import { SignupFormData } from '@/types/auth'

export const AuthPage: React.FC = () => {
  const { user, isAuthenticated, login, loginWithGoogle, signup } = useAppState()
  const navigate = useNavigate()
  const [tab, setTab] = useState<'signin' | 'signup'>('signup')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

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

  // If already authenticated, redirect to chat
  useEffect(() => {
    if (isAuthenticated) {
      navigate('/chat', { replace: true })
    }
  }, [isAuthenticated, navigate])

  const handleGoogleAuth = async () => {
    setLoading(true)
    setError(null)
    const res = await loginWithGoogle()
    setLoading(false)
    if (res.success) {
      navigate('/chat', { replace: true })
    } else {
      setError(res.error || 'Google sign in failed')
    }
  }

  const handleSignInSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
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
          <h2>{tab === 'signin' ? 'Welcome Back, Farmer' : 'Register Your Aquaculture Farm'}</h2>
          <p>
            {tab === 'signin'
              ? 'Sign in to access AI clinical triage, farm simulations, and veterinary consultations.'
              : 'Create an account with your farm details to start receiving grounded advisory and predictive simulations.'}
          </p>
        </div>

        {/* 1-Click Google Sign In */}
        <button
          type="button"
          className="google-auth-btn"
          onClick={handleGoogleAuth}
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
          <span>or continue with email & farm details</span>
        </div>

        {/* Tab Switcher */}
        <div className="auth-tab-bar">
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
        </div>

        {error && <div className="auth-error-alert">{error}</div>}

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
                <label>Farm Location (State / Region)</label>
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
              {loading ? 'Registering Farm Profile…' : 'Complete Farm Registration & Enter'}
            </button>
          </form>
        )}

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
              {loading ? 'Signing In…' : 'Sign In to Account'}
            </button>
          </form>
        )}
      </div>
    </div>
  )
}
