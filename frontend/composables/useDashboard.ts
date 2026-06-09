export interface ApplicationItem {
  id: string
  applicantName: string
  projectName: string
  projectUrl: string
  expectedDailyRequests: number
  usageScenario: string
  status: string
  reviewNote: string
  createdAt: string
  reviewedAt?: string
  defaultMinuteLimit?: number
  defaultDailyLimit?: number
}

export interface TokenItem {
  id: string
  applicationId: string
  name: string
  tokenPrefix: string
  status: string
  minuteLimit: number
  dailyLimit: number
  lastUsedAt?: string
  createdAt: string
}

export interface StatsSummary {
  totalRequests: number
  successRequests: number
  errorRequests: number
  avgLatencyMs: number
  uniqueOrigins: number
  uniqueIPs: number
}

export interface TrendItem { date: string; totalRequests: number; successRequests: number; errorRequests: number }
export interface SourceItem { origin: string; refererHost: string; requests: number }
export interface EndpointItem { route: string; requests: number; avgLatencyMs: number; errorRate: number }

export const useDashboard = () => {
  const { apiFetch } = useApi()
  const applications = () => apiFetch<ApplicationItem[]>('/applications')
  const tokens = () => apiFetch<TokenItem[]>('/tokens')
  const summary = (days = 30, tokenId?: string) => apiFetch<StatsSummary>('/dashboard/stats/summary', { query: { days, tokenId } })
  const trend = (days = 30, tokenId?: string) => apiFetch<TrendItem[]>('/dashboard/stats/trend', { query: { days, tokenId } })
  const sources = (days = 30, tokenId?: string) => apiFetch<SourceItem[]>('/dashboard/stats/sources', { query: { days, tokenId } })
  const endpoints = (days = 30, tokenId?: string) => apiFetch<EndpointItem[]>('/dashboard/stats/endpoints', { query: { days, tokenId } })
  return { applications, tokens, summary, trend, sources, endpoints }
}
