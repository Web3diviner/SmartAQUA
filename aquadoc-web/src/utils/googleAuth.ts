/**
 * Real Google Identity Services (GIS) & OAuth 2.0 Client.
 *
 * Opens the native Google Account Chooser popup on the user's browser/device,
 * prompts account selection from real signed-in Google profiles, and fetches
 * verified user details from Google's UserInfo API.
 */

export interface GoogleProfile {
  email: string
  name: string
  picture?: string
  sub?: string
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
 * Launch the authentic Google OAuth 2.0 Account Chooser Popup.
 * Opens Google's official account selection window where the user selects
 * their real Google/Gmail accounts logged into this browser or device.
 */
export async function launchGoogleOAuthPopup(
  clientId: string,
  onSuccess: (profile: GoogleProfile) => void,
  onError: (error: string) => void,
) {
  const google = (window as any).google

  // If GIS is available and a Client ID is present, use Google Token Client
  if (google?.accounts?.oauth2 && clientId && !clientId.includes('YOUR_')) {
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
              // Fetch real verified user info from Google's UserInfo endpoint
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

      // Prompt account selection
      tokenClient.requestAccessToken({ prompt: 'select_account' })
      return true
    } catch (err: any) {
      console.warn('Google TokenClient init failed:', err)
      onError(err?.message || 'Failed to open Google Account Chooser')
      return false
    }
  }

  // If GIS ID button is available, try id.prompt
  if (google?.accounts?.id && clientId && !clientId.includes('YOUR_')) {
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
        prompt_parent_id: 'google-btn-container',
      })

      google.accounts.id.prompt((notification: any) => {
        if (notification.isNotDisplayed() || notification.isSkippedMoment()) {
          console.info('Google One-Tap dismissed/unavailable:', notification.getNotDisplayedReason?.())
        }
      })
      return true
    } catch (err: any) {
      console.warn('Google One-Tap failed:', err)
      onError(err?.message || 'Google Sign-In failed')
      return false
    }
  }

  return false
}
