/**
 * Google Identity Services (GIS) & OAuth 2.0 Integration.
 *
 * Checks for a verified Google OAuth Client ID before triggering Google's popup.
 * If not configured, gracefully falls back to direct Gmail identity sign-in
 * to prevent "Access blocked: Authorization Error" from Google's servers.
 */

export interface GoogleProfile {
  email: string
  name: string
  picture?: string
  sub?: string
}

export function isValidGoogleClientId(clientId?: string): boolean {
  if (!clientId) return false
  if (
    clientId.includes('YOUR_') ||
    clientId.includes('placeholder') ||
    clientId.includes('smart-aqua-aquaculture') ||
    !clientId.includes('.apps.googleusercontent.com')
  ) {
    return false
  }
  return true
}

export function parseJwt(token: string): any {
  try {
    const base64Url = token.split('.')[1]
    if (!base64Url) return null
    const base64 = base64Url.replace(/-/g, '+').replace(/_/g, '/')
    const jsonPayload = decodeURIComponent(
      window
        .atob(base64)
        .split('')
        .map((c) => `%${`00${c.charCodeAt(0).toString(16)}`.slice(-2)}`)
        .join(''),
    )
    return JSON.parse(jsonPayload)
  } catch (err) {
    console.warn('Failed to parse Google JWT:', err)
    return null
  }
}

/**
 * Launch Google OAuth popup ONLY if a verified Google Cloud Client ID is configured.
 */
export async function launchGoogleOAuthPopup(
  clientId: string,
  onSuccess: (profile: GoogleProfile) => void,
  onError: (error: string) => void,
): Promise<boolean> {
  // Guard against unconfigured or dummy client IDs that trigger Google's Authorization Error
  if (!isValidGoogleClientId(clientId)) {
    return false
  }

  const google = (window as any).google

  // Try Google Token Client if available
  if (google?.accounts?.oauth2) {
    try {
      const tokenClient = google.accounts.oauth2.initTokenClient({
        client_id: clientId,
        scope: 'email profile openid',
        prompt: 'select_account',
        callback: async (tokenResponse: any) => {
          if (tokenResponse.error) {
            onError(tokenResponse.error_description || tokenResponse.error || 'Google authentication cancelled')
            return
          }

          if (tokenResponse.access_token) {
            try {
              const res = await fetch('https://www.googleapis.com/oauth2/v3/userinfo', {
                headers: { Authorization: `Bearer ${tokenResponse.access_token}` },
              })
              if (!res.ok) {
                throw new Error(`Google UserInfo returned ${res.status}`)
              }
              const info = await res.json()
              onSuccess({
                email: info.email,
                name: info.name || info.given_name || info.email.split('@')[0],
                picture: info.picture,
                sub: info.sub,
              })
            } catch (fetchErr: any) {
              console.error('Failed to fetch Google profile info:', fetchErr)
              onError('Could not retrieve Google profile details.')
            }
          }
        },
      })

      tokenClient.requestAccessToken({ prompt: 'select_account' })
      return true
    } catch (err: any) {
      console.warn('Google TokenClient init failed:', err)
      return false
    }
  }

  // Fallback to id.prompt
  if (google?.accounts?.id) {
    try {
      google.accounts.id.initialize({
        client_id: clientId,
        callback: (response: any) => {
          if (response.credential) {
            const payload = parseJwt(response.credential)
            if (payload?.email) {
              onSuccess({
                email: payload.email,
                name: payload.name || payload.given_name || payload.email.split('@')[0],
                picture: payload.picture,
                sub: payload.sub,
              })
              return
            }
          }
          onError('Google credential verification failed')
        },
        auto_select: false,
      })

      google.accounts.id.prompt()
      return true
    } catch (err: any) {
      console.warn('Google One-Tap prompt failed:', err)
    }
  }

  return false
}
