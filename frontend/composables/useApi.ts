export interface ApiSuccess<T> { success: true; data: T }
export interface ApiFailure { success: false; error: { code: string; message: string } }
export type ApiResponse<T> = ApiSuccess<T> | ApiFailure

type ApiOptions = {
  method?: 'GET' | 'POST'
  body?: any
  query?: Record<string, string | number | boolean | undefined>
  headers?: Record<string, string>
}

export const useApi = () => {
  const config = useRuntimeConfig()
  const baseURL = String(config.public.apiBaseUrl).replace(/\/$/, '')

  const apiFetch = async <T>(path: string, options: ApiOptions = {}) => {
    const headers: Record<string, string> = { ...(options.headers || {}) }
    if (import.meta.server) {
      const requestHeaders = useRequestHeaders(['cookie'])
      if (requestHeaders.cookie && !headers.cookie) {
        headers.cookie = requestHeaders.cookie
      }
    }

    return await $fetch<ApiResponse<T>>(`${baseURL}${path}`, {
      method: options.method || 'GET',
      body: options.body,
      query: options.query,
      headers,
      credentials: 'include'
    })
  }

  return { apiFetch, baseURL }
}
