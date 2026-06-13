export const DEFAULT_POST_LOGIN_REDIRECT = '/dashboard'

const allowedPostLoginRedirectPrefixes = ['/dashboard', '/admin'] as const

const hasAllowedRedirectBoundary = (path: string, prefix: string) => {
  if (path === prefix) {
    return true
  }

  const next = path.charCodeAt(prefix.length)
  return next === 47 || next === 63 || next === 35
}

const hasUnsafeWhitespaceOrControl = (value: string) => {
  for (let index = 0; index < value.length; index += 1) {
    const code = value.charCodeAt(index)
    if (code <= 32 || code === 127) {
      return true
    }
  }

  return false
}

export const sanitizePostLoginRedirect = (value: unknown) => {
  if (typeof value !== 'string') {
    return DEFAULT_POST_LOGIN_REDIRECT
  }

  if (
    value.length === 0 ||
    value.charCodeAt(0) !== 47 ||
    value.startsWith('//') ||
    hasUnsafeWhitespaceOrControl(value)
  ) {
    return DEFAULT_POST_LOGIN_REDIRECT
  }

  for (const prefix of allowedPostLoginRedirectPrefixes) {
    if (hasAllowedRedirectBoundary(value, prefix)) {
      return value
    }
  }

  return DEFAULT_POST_LOGIN_REDIRECT
}
