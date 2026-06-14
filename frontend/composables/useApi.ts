import type { MaybeRefOrGetter, MultiWatchSources } from 'vue'
import { toValue } from 'vue'

export interface ApiSuccess<T> { success: true; data: T }
export interface ApiFailure { success: false; error: { code: string; message: string } }
export type ApiResponse<T> = ApiSuccess<T> | ApiFailure

type ApiQuery = Record<string, string | number | boolean | undefined>

type ApiOptions = {
  method?: 'GET' | 'POST' | 'PATCH' | 'DELETE'
  body?: any
  query?: ApiQuery
  headers?: Record<string, string>
  signal?: AbortSignal
}

type ApiDataOptions<T> = {
  query?: MaybeRefOrGetter<ApiQuery>
  headers?: Record<string, string>
  server?: boolean
  lazy?: boolean
  immediate?: boolean
  deep?: boolean
  watch?: MultiWatchSources
  dedupe?: 'cancel' | 'defer'
  default?: () => ApiResponse<T> | null
}

const toApiFailure = (err: unknown): ApiFailure => {
  const data = (err as { data?: ApiResponse<never> }).data
  if (data && data.success === false) {
    return data
  }

  const message = (err as { message?: string }).message || '请求失败'
  return { success: false, error: { code: 'FETCH_ERROR', message } }
}

const normalizeBaseURL = (value: unknown) => String(value || '').replace(/\/$/, '')

// Server-side $fetch treats relative URLs as Nitro-local routes. Relative SSR API bases must be backed by trusted config, not request Host.
const resolveFetchBaseURL = (publicBaseURL: string, serverBaseURL: string) => {
  const configuredBaseURL = import.meta.server && serverBaseURL ? serverBaseURL : publicBaseURL
  if (import.meta.server && configuredBaseURL.startsWith('/')) {
    throw new Error('NUXT_API_BASE_URL is required when NUXT_PUBLIC_API_BASE_URL is relative')
  }

  return configuredBaseURL
}

export const useApi = () => {
  const config = useRuntimeConfig()
  const baseURL = normalizeBaseURL(config.public.apiBaseUrl)
  const fetchBaseURL = resolveFetchBaseURL(baseURL, normalizeBaseURL(config.apiBaseUrl))

  const headersWithRequestCookie = (input?: Record<string, string>) => {
    const headers: Record<string, string> = { ...(input || {}) }
    if (import.meta.server) {
      const requestHeaders = useRequestHeaders(['cookie'])
      if (requestHeaders.cookie && !headers.cookie) {
        headers.cookie = requestHeaders.cookie
      }
    }
    return headers
  }

  const apiFetchWithHeaders = async <T>(path: string, options: ApiOptions, headers: Record<string, string>) => {
    return await $fetch<ApiResponse<T>>(`${fetchBaseURL}${path}`, {
      method: options.method || 'GET',
      body: options.body,
      query: options.query,
      headers,
      credentials: 'include',
      signal: options.signal
    })
  }

  const apiFetch = async <T>(path: string, options: ApiOptions = {}) => {
    return await apiFetchWithHeaders<T>(path, options, headersWithRequestCookie(options.headers))
  }

  const apiData = <T>(key: string, path: string, options: ApiDataOptions<T> = {}) => {
    const headers = headersWithRequestCookie(options.headers)
    return useAsyncData<ApiResponse<T>, unknown, ApiResponse<T>, [], ApiResponse<T> | null>(
      key,
      async (_nuxtApp, { signal }) => {
        try {
          return await apiFetchWithHeaders<T>(path, {
            query: toValue(options.query),
            signal
          }, headers)
        } catch (err) {
          return toApiFailure(err)
        }
      },
      {
        server: options.server ?? true,
        lazy: options.lazy ?? false,
        immediate: options.immediate ?? true,
        deep: options.deep ?? false,
        watch: options.watch,
        dedupe: options.dedupe ?? 'defer',
        default: options.default
      }
    )
  }

  return { apiFetch, apiData, baseURL }
}
