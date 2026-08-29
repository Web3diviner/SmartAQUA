/**
 * Google Identity Services (GIS) & Account Chooser Utility.
 *
 * Facilitates selecting from signed-in Google accounts on the user's browser/device,
 * decoding Google JWT credentials, and provisioning farmer sessions.
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
 * Initialize Google Identity Services with Account Chooser prompt.
 */
export function promptGoogleAccountPicker(
  clientId: string,
  onSuccess: (profile: GoogleProfile) => void,
  onError?: (error: string) => void,
) {
  const google = (window as any).google

  if (!google?.accounts?.id) {
    // If GIS script hasn't loaded or is blocked, fallback to simulated account chooser
    onError?.('Google Identity Services unavailable')
    return false
  }

  try {
    google.accounts.id.initialize({
      client_id: clientId,
      callback: (response: any) => {
        if (response.credential) {
          const payload = parseJwt(response.credential)
          if (payload && payload.email) {
            onSuccess({
              email: payload.email,
              name: payload.name || payload.given_name || payload.email.split('@')[0],
              picture: payload.picture,
              sub: payload.sub,
            })
            return
          }
        }
        onError?.('Google credential verification failed')
      },
      auto_select: false,
      cancel_on_tap_outside: true,
      prompt_parent_id: 'google-onetap-anchor',
    })

    // Display Google One Tap & Account Chooser overlay
    google.accounts.id.prompt((notification: any) => {
      if (notification.isNotDisplayed() || notification.isSkippedMoment()) {
        console.info('Google One Tap not displayed:', notification.getNotDisplayedReason?.())
      }
    })

    return true
  } catch (err: any) {
    console.warn('Google GIS initialize error:', err)
    onError?.(err?.message || 'Failed to initialize Google Account Chooser')
    return false
  }
}
