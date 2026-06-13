import { getRequestURL, setResponseHeaders } from 'h3'

const baseConnectSources = "'self' https://challenges.cloudflare.com"
const devConnectSources = process.env.NODE_ENV === 'production' ? '' : ' ws: wss:'
const cspBeforeConnect = "default-src 'self'; base-uri 'self'; form-action 'self'; frame-ancestors 'none'; object-src 'none'; img-src 'self' data: blob:; font-src 'self' data:; connect-src "
const cspAfterConnect = "; script-src 'self' 'unsafe-inline' https://challenges.cloudflare.com; style-src 'self' 'unsafe-inline'; frame-src https://challenges.cloudflare.com"
const permissionsPolicy =
  'accelerometer=(), autoplay=(), camera=(), display-capture=(), encrypted-media=(), fullscreen=(), geolocation=(), gyroscope=(), magnetometer=(), microphone=(), midi=(), payment=(), picture-in-picture=(), publickey-credentials-get=(), usb=(), xr-spatial-tracking=()'

let cachedApiBaseUrl = ''
let cachedCspApiOrigin = ''
let cachedContentSecurityPolicy = ''

let cachedApiOrigin = ''

export default defineEventHandler((event) => {
  const config = useRuntimeConfig(event)
  const apiOrigin = httpOrigin(String(config.public.apiBaseUrl || ''))
  let contentSecurityPolicy = cachedContentSecurityPolicy
  if (apiOrigin !== cachedCspApiOrigin || contentSecurityPolicy === '') {
    const apiConnectSource = apiOrigin ? ` ${apiOrigin}` : ''
    cachedCspApiOrigin = apiOrigin
    contentSecurityPolicy = `${cspBeforeConnect}${baseConnectSources}${apiConnectSource}${devConnectSources}${cspAfterConnect}`
    cachedContentSecurityPolicy = contentSecurityPolicy
  }
  const requestPath = getRequestURL(event).pathname
  const sensitivePath =
    requestPath === '/auth' ||
    requestPath.startsWith('/auth/') ||
    requestPath === '/dashboard' ||
    requestPath.startsWith('/dashboard/') ||
    requestPath === '/admin' ||
    requestPath.startsWith('/admin/')

  const headers: Record<string, string> = {
    'X-Content-Type-Options': 'nosniff',
    'Referrer-Policy': 'no-referrer',
    'Content-Security-Policy': contentSecurityPolicy,
    'X-Frame-Options': 'DENY',
    'Permissions-Policy': permissionsPolicy
  }

  if (sensitivePath) {
    headers['Cache-Control'] = 'no-store'
  }

  setResponseHeaders(event, headers)
})

function httpOrigin(value: string) {
  if (value === cachedApiBaseUrl) {
    return cachedApiOrigin
  }

  cachedApiBaseUrl = value
  cachedApiOrigin = ''
  try {
    const url = new URL(value)
    if (url.protocol === 'http:' || url.protocol === 'https:') {
      cachedApiOrigin = url.origin
    }
  } catch {
    cachedApiOrigin = ''
  }
  return cachedApiOrigin
}
