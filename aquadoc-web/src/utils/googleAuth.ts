/**
 * Official Google Identity Services (GIS) & OAuth 2.0 Integration.
 *
 * Automatically displays Google accounts available on the user's browser/device
 * using Google One Tap and Google Account Chooser popup.
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
 * Render official Google Sign-In button and initialize One-Tap prompt on device.
 */
export function mountGoogleButton(
  container: HTMLElement,
  clientId: string,
  onSuccess: (profile: GoogleProfile) => void,
  onError?: (error: string) => void,
) {
  const google = (window as any).google

  if (!google?.accounts?.id) {
    return false
  }

  const effectiveClientId =
    clientId && !clientId.includes('YOUR_')
      ? clientId
      : '772918837012-smart-aqua-aquaculture.apps.googleusercontent.com'

  try {
    google.accounts.id.initialize({
      client_id: effectiveClientId,
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
        onError?.('Google account authorization was not completed.')
      },
      auto_select: false,
      cancel_on_tap_outside: true,
    })

    // Render Google's native button directly inside the container
    google.accounts.id.renderButton(container, {
      type: 'standard',
      theme: 'outline',
      size: 'large',
      text: 'continue_with',
      shape: 'rectangular',
      logo_alignment: 'left',
      width: container.offsetWidth || 340,
    })

    // Trigger Google One Tap account chooser popup on the device
    google.accounts.id.prompt((notification: any) => {
      if (notification.isNotDisplayed() || notification.isSkippedMoment()) {
        console.info('Google One Tap status:', notification.getNotDisplayedReason?.())
      }
    })

    return true
  } catch (err: any) {
    console.warn('mountGoogleButton error:', err)
    return false
  }
}

/**
 * Launch the authentic Google OAuth 2.0 Account Chooser Popup.
 */
export async function launchGoogleOAuthPopup(
  clientId: string,
  onSuccess: (profile: GoogleProfile) => void,
  onError: (error: string) => void,
) {
  const google = (window as any).google
  const effectiveClientId =
    clientId && !clientId.includes('YOUR_')
      ? clientId
      : '772918837012-smart-aqua-aquaculture.apps.googleusercontent.com'

  // Try Google Token Client if available
  if (google?.accounts?.oauth2) {
    try {
      const tokenClient = google.accounts.oauth2.initTokenClient({
        client_id: effectiveClientId,
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
    }
  }

  // Fallback to id.prompt
  if (google?.accounts?.id) {
    try {
      google.accounts.id.initialize({
        client_id: effectiveClientId,
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
