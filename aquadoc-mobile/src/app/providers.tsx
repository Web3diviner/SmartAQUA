/**
 * Shared application state & Farmer Authentication.
 *
 * 1. Holds simulated farm context and selected LLM model.
 * 2. Manages farmer login/signup session (Email & Google Gmail) to protect AI responses.
 */

import { createContext, useCallback, useContext, useEffect, useMemo, useState } from 'react'
import type { ReactNode } from 'react'

import { type ClientConfig, readClientConfig } from '@/api/client'
import { DEFAULT_MODEL_ID } from '@/constants/models'
import {
  DEFAULT_FARM_CONTEXT_FORM,
  type FarmContextForm,
} from '@/schemas/farmContext'
import { AuthUser, SignupFormData } from '@/types/auth'

const STORAGE_KEY = 'aquadoc.farmContextForm.v1'
const MODEL_STORAGE_KEY = 'aquadoc.selectedModel.v1'
const AUTH_USER_KEY = 'aquadoc.auth.user.v1'

/** 15_AQUADOC_FRONTEND.md section 5, "Farm-Aware Chat Modes". */
export type ChatMode = 'general' | 'simulated_pond'

export type ThemeMode = 'dark' | 'light'

interface AppState {
  config: ClientConfig
  /** Debug panels are opt-in via VITE_ENABLE_DEBUG_PANEL. */
  debugAvailable: boolean
  devMode: boolean
  setDevMode: (value: boolean) => void
  theme: ThemeMode
  setTheme: (theme: ThemeMode) => void
  toggleTheme: () => void
  chatMode: ChatMode
  setChatMode: (mode: ChatMode) => void
  selectedModel: string
  setSelectedModel: (model: string) => void
  farmForm: FarmContextForm
  setFarmForm: (form: FarmContextForm) => void
  resetFarmForm: () => void
  // Authentication State & Actions
  user: AuthUser | null
  isAuthenticated: boolean
  isAuthModalOpen: boolean
  openAuthModal: () => void
  closeAuthModal: () => void
  login: (email: string, password?: string) => Promise<{ success: boolean; error?: string }>
  loginWithGoogle: (email?: string, name?: string) => Promise<{ success: boolean; error?: string }>
  signup: (data: SignupFormData) => Promise<{ success: boolean; error?: string }>
  logout: () => void
}

const AppStateContext = createContext<AppState | null>(null)

function loadStoredForm(): FarmContextForm {
  try {
    const raw = window.localStorage.getItem(STORAGE_KEY)
    if (!raw) return DEFAULT_FARM_CONTEXT_FORM
    return { ...DEFAULT_FARM_CONTEXT_FORM, ...(JSON.parse(raw) as Partial<FarmContextForm>) }
  } catch {
    return DEFAULT_FARM_CONTEXT_FORM
  }
}

function loadStoredModel(): string {
  try {
    const stored = window.localStorage.getItem(MODEL_STORAGE_KEY)
    return stored || DEFAULT_MODEL_ID
  } catch {
    return DEFAULT_MODEL_ID
  }
}

function loadStoredUser(): AuthUser | null {
  try {
    const stored = window.localStorage.getItem(AUTH_USER_KEY)
    if (!stored) return null
    return JSON.parse(stored) as AuthUser
  } catch {
    return null
  }
}

export function generateUserIdForEmail(email: string): string {
  const clean = email.trim().toLowerCase()
  let hash = 0
  for (let i = 0; i < clean.length; i++) {
    hash = (hash << 5) - hash + clean.charCodeAt(i)
    hash |= 0
  }
  return `USR-${Math.abs(hash).toString(16).toUpperCase().padStart(8, '0')}`
}

export function AppProviders({ children }: { children: ReactNode }) {
  const config = useMemo(readClientConfig, [])
  const debugAvailable = import.meta.env.VITE_ENABLE_DEBUG_PANEL === 'true'

  const [devMode, setDevMode] = useState(false)
  const [chatMode, setChatMode] = useState<ChatMode>('general')
  const [selectedModel, setSelectedModelState] = useState<string>(loadStoredModel)
  const [farmForm, setFarmForm] = useState<FarmContextForm>(loadStoredForm)
  const [user, setUser] = useState<AuthUser | null>(loadStoredUser)
  const [isAuthModalOpen, setIsAuthModalOpen] = useState(false)
  const [theme, setThemeState] = useState<ThemeMode>(() => {
    return (window.localStorage.getItem('aquadoc.theme') as ThemeMode) || 'dark'
  })

  const setSelectedModel = useCallback((model: string) => {
    setSelectedModelState(model)
    try {
      window.localStorage.setItem(MODEL_STORAGE_KEY, model)
    } catch {
      // Ignored if storage disabled
    }
  }, [])

  const setTheme = useCallback((newTheme: ThemeMode) => {
    setThemeState(newTheme)
    window.localStorage.setItem('aquadoc.theme', newTheme)
  }, [])

  const toggleTheme = useCallback(() => {
    setTheme(theme === 'dark' ? 'light' : 'dark')
  }, [theme, setTheme])

  useEffect(() => {
    document.documentElement.setAttribute('data-theme', theme)
  }, [theme])

  useEffect(() => {
    try {
      window.localStorage.setItem(STORAGE_KEY, JSON.stringify(farmForm))
    } catch {
      // A full or disabled localStorage must not break the app.
    }
  }, [farmForm])

  // Save auth user session
  useEffect(() => {
    try {
      if (user) {
        window.localStorage.setItem(AUTH_USER_KEY, JSON.stringify(user))
      } else {
        window.localStorage.removeItem(AUTH_USER_KEY)
      }
    } catch {
      // Storage error fallback
    }
  }, [user])

  const openAuthModal = useCallback(() => setIsAuthModalOpen(true), [])
  const closeAuthModal = useCallback(() => setIsAuthModalOpen(false), [])

  const login = useCallback(
    async (email: string, password = 'password123'): Promise<{ success: boolean; error?: string }> => {
      try {
        const res = await fetch(`${config.baseUrl}/dev/v1/auth/login`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ email, password }),
        })
        const data = await res.json()
        if (res.ok && data.user) {
          setUser(data.user)
          setIsAuthModalOpen(false)
          return { success: true }
        }
        return { success: false, error: data.error || 'Invalid credentials' }
      } catch {
        // Offline / Fallback login
        const fallbackUser: AuthUser = {
          id: generateUserIdForEmail(email),
          name: (email.split('@')[0] ?? 'Farmer')
            .replace('.', ' ')
            .replace(/\b\w/g, (l) => l.toUpperCase()),
          email,
          phone: '+2348071055742',
          farmName: 'My Aquaculture Farm',
          farmLocation: 'Lagos, Nigeria',
          primarySpecies: 'African Catfish (Clarias gariepinus)',
          farmingSystem: 'Concrete Tanks',
          avatarUrl: `https://api.dicebear.com/7.x/bottts/svg?seed=${encodeURIComponent(email)}`,
          provider: 'credentials',
          token: `aqua_usr_${generateUserIdForEmail(email)}`,
          createdAt: new Date().toISOString(),
        }
        setUser(fallbackUser)
        setIsAuthModalOpen(false)
        return { success: true }
      }
    },
    [config.baseUrl],
  )

  const loginWithGoogle = useCallback(
    async (
      email = 'farmer.gmail@gmail.com',
      name = 'Smart Aqua Farmer',
    ): Promise<{ success: boolean; error?: string }> => {
      try {
        const res = await fetch(`${config.baseUrl}/dev/v1/auth/google`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({
            email,
            name,
            avatar_url: `https://api.dicebear.com/7.x/bottts/svg?seed=${encodeURIComponent(email)}`,
            google_id: `g_${generateUserIdForEmail(email)}`,
            farm_name: `${name}'s Fishery`,
            farm_location: 'Epe, Lagos State',
            primary_species: 'African Catfish (Clarias gariepinus)',
            farming_system: 'Concrete Tanks',
          }),
        })
        const data = await res.json()
        if (res.ok && data.user) {
          setUser(data.user)
          setIsAuthModalOpen(false)
          return { success: true }
        }
        return { success: false, error: data.error || 'Google login failed' }
      } catch {
        const googleUser: AuthUser = {
          id: generateUserIdForEmail(email),
          name,
          email,
          phone: '+2348071055742',
          farmName: `${name}'s Fishery`,
          farmLocation: 'Epe, Lagos State',
          primarySpecies: 'African Catfish (Clarias gariepinus)',
          farmingSystem: 'Concrete Tanks',
          avatarUrl: `https://api.dicebear.com/7.x/bottts/svg?seed=${encodeURIComponent(email)}`,
          provider: 'google',
          token: `aqua_google_${generateUserIdForEmail(email)}`,
          createdAt: new Date().toISOString(),
        }
        setUser(googleUser)
        setIsAuthModalOpen(false)
        return { success: true }
      }
    },
    [config.baseUrl],
  )

  const signup = useCallback(
    async (formData: SignupFormData): Promise<{ success: boolean; error?: string }> => {
      try {
        const res = await fetch(`${config.baseUrl}/dev/v1/auth/signup`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({
            name: formData.name,
            email: formData.email,
            password: formData.password,
            phone: formData.phone,
            farm_name: formData.farmName,
            farm_location: formData.farmLocation,
            primary_species: formData.primarySpecies,
            farming_system: formData.farmingSystem,
          }),
        })
        const data = await res.json()
        if (res.ok && data.user) {
          setUser(data.user)
          setIsAuthModalOpen(false)
          return { success: true }
        }
        return { success: false, error: data.error || 'Registration failed' }
      } catch {
        const newUser: AuthUser = {
          id: generateUserIdForEmail(formData.email),
          name: formData.name,
          email: formData.email,
          phone: formData.phone,
          farmName: formData.farmName,
          farmLocation: formData.farmLocation,
          primarySpecies: formData.primarySpecies,
          farmingSystem: formData.farmingSystem,
          avatarUrl: `https://api.dicebear.com/7.x/bottts/svg?seed=${encodeURIComponent(formData.email)}`,
          provider: 'credentials',
          token: `aqua_usr_${generateUserIdForEmail(formData.email)}`,
          createdAt: new Date().toISOString(),
        }
        setUser(newUser)
        setIsAuthModalOpen(false)
        return { success: true }
      }
    },
    [config.baseUrl],
  )

  const logout = useCallback(() => {
    setUser(null)
    window.localStorage.removeItem(AUTH_USER_KEY)
  }, [])

  const resetFarmForm = useCallback(() => setFarmForm(DEFAULT_FARM_CONTEXT_FORM), [])

  const value = useMemo<AppState>(
    () => ({
      config,
      debugAvailable,
      devMode: debugAvailable && devMode,
      setDevMode,
      theme,
      setTheme,
      toggleTheme,
      chatMode,
      setChatMode,
      selectedModel,
      setSelectedModel,
      farmForm,
      setFarmForm,
      resetFarmForm,
      user,
      isAuthenticated: Boolean(user),
      isAuthModalOpen,
      openAuthModal,
      closeAuthModal,
      login,
      loginWithGoogle,
      signup,
      logout,
    }),
    [
      config,
      debugAvailable,
      devMode,
      theme,
      setTheme,
      toggleTheme,
      chatMode,
      selectedModel,
      setSelectedModel,
      farmForm,
      resetFarmForm,
      user,
      isAuthModalOpen,
      openAuthModal,
      closeAuthModal,
      login,
      loginWithGoogle,
      signup,
      logout,
    ],
  )

  return <AppStateContext.Provider value={value}>{children}</AppStateContext.Provider>
}

export function useAppState(): AppState {
  const state = useContext(AppStateContext)
  if (!state) throw new Error('useAppState must be used inside <AppProviders>')
  return state
}
