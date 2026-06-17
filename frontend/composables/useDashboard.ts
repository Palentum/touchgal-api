export interface ApplicationItem {
  id: string
  userId: string
  applicantName: string
  projectName: string
  projectUrl: string
  expectedDailyRequests: number
  usageScenario: string
  status: string
  createdAt: string
  reviewedAt?: string
  defaultMinuteLimit?: number
  defaultDailyLimit?: number
  owner?: ApplicationOwner
}

export interface ApplicationOwner {
  id: string
  email: string
  displayName: string
}

export interface TokenOwner {
  id: string
  email: string
  displayName: string
}

export interface TokenItem {
  id: string
  userId: string
  applicationId: string
  name: string
  tokenPrefix: string
  status: string
  minuteLimit: number
  dailyLimit: number
  lastUsedAt?: string
  createdAt: string
  owner?: TokenOwner
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
export interface StatsDashboard {
  summary: StatsSummary
  trend: TrendItem[]
  sources: SourceItem[]
  endpoints: EndpointItem[]
}


export const useDashboard = () => {
  const { apiFetch } = useApi()
  const tokens = () => apiFetch<TokenItem[]>('/tokens')
  const stats = (days = 30, tokenId?: string) => apiFetch<StatsDashboard>('/dashboard/stats', { query: { days, tokenId } })
  return { tokens, stats }
}
