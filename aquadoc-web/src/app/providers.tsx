/**
 * Shared application state.
 *
 * The simulated farm context is held here rather than on the Chat page so the
 * Farm Simulator and the Chat share one pond: editing a measurement changes the
 * next question's context immediately. That is the whole point of the simulator
 * (15_AQUADOC_FRONTEND.md section 4).
 *
 * State persists to localStorage so a page reload does not discard a pond
 * configuration mid-session. Only simulated values are stored — never a token.
 */

import { createContext, useCallback, useContext, useEffect, useMemo, useState } from 'react'
import type { ReactNode } from 'react'

import { type ClientConfig, readClientConfig } from '@/api/client'
import { DEFAULT_MODEL_ID } from '@/constants/models'
import {
  DEFAULT_FARM_CONTEXT_FORM,
  type FarmContextForm,
} from '@/schemas/farmContext'

const STORAGE_KEY = 'aquadoc.farmContextForm.v1'
const MODEL_STORAGE_KEY = 'aquadoc.selectedModel.v1'

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
}

const AppStateContext = createContext<AppState | null>(null)

function loadStoredForm(): FarmContextForm {
  try {
    const raw = window.localStorage.getItem(STORAGE_KEY)
    if (!raw) return DEFAULT_FARM_CONTEXT_FORM
    // Merge over the defaults so a stored form from an older shape still loads.
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

export function AppProviders({ children }: { children: ReactNode }) {
  const config = useMemo(readClientConfig, [])
  const debugAvailable = import.meta.env.VITE_ENABLE_DEBUG_PANEL === 'true'

  const [devMode, setDevMode] = useState(false)
  const [chatMode, setChatMode] = useState<ChatMode>('general')
  const [selectedModel, setSelectedModelState] = useState<string>(loadStoredModel)
  const [farmForm, setFarmForm] = useState<FarmContextForm>(loadStoredForm)
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

  const resetFarmForm = useCallback(() => setFarmForm(DEFAULT_FARM_CONTEXT_FORM), [])

  const value = useMemo<AppState>(
    () => ({
      config,
      debugAvailable,
      // Debug output stays off entirely when the build disables it.
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
    }),
    [config, debugAvailable, devMode, theme, setTheme, toggleTheme, chatMode, selectedModel, setSelectedModel, farmForm, resetFarmForm],
  )

  return <AppStateContext.Provider value={value}>{children}</AppStateContext.Provider>
}

export function useAppState(): AppState {
  const state = useContext(AppStateContext)
  if (!state) throw new Error('useAppState must be used inside <AppProviders>')
  return state
}
