export interface AuthUser {
  id: string
  name: string
  email: string
  phone: string
  farmName: string
  farmLocation: string
  primarySpecies: string
  farmingSystem: string
  avatarUrl?: string
  provider: 'google' | 'credentials'
  token: string
  createdAt: string
}

export interface SignupFormData {
  name: string
  email: string
  password: string
  phone: string
  farmName: string
  farmLocation: string
  primarySpecies: string
  farmingSystem: string
}
