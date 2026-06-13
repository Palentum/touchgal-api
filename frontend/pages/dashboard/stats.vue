<template>
  <section class="tg-dashboard-stack">
    <div class="flex flex-wrap items-end justify-between gap-4">
      <div>
        <p class="tg-eyebrow">Analysis</p>
        <h1 class="tg-display-md">请求统计</h1>
        <p class="tg-lead">按时间范围和 Token 查询请求趋势与来源。</p>
      </div>

      <div class="tg-actions">
        <label class="tg-label">
          时间范围
          <select v-model.number="days" class="tg-select">
            <option :value="7">7 天</option>
            <option :value="14">14 天</option>
            <option :value="30">30 天</option>
            <option :value="90">90 天</option>
          </select>
        </label>
        <label class="tg-label">
          Token
          <select v-model="tokenId" class="tg-select">
            <option value="">全部 Token</option>
            <option v-for="token in tokens" :key="token.id" :value="token.id">{{ token.name }}</option>
          </select>
        </label>
      </div>
    </div>

    <DashboardRequestSummaryCards :summary="summaryData" />

    <div class="tg-grid-2">
      <div class="tg-card">
        <h2 class="tg-title-lg">请求趋势</h2>
        <div ref="trendChartHost" class="tg-chart-box mt-4">
          <LazyDashboardRequestTrendChart v-if="trendChartVisible" :data="trendData" />
          <div v-else class="flex min-h-[320px] items-center justify-center text-sm text-[var(--tg-muted)]">
            图表将在进入视口后加载
          </div>
        </div>
      </div>
      <div class="tg-card">
        <h2 class="tg-title-lg">请求来源</h2>
        <div ref="sourceChartHost" class="tg-chart-box mt-4">
          <LazyDashboardSourcePieChart v-if="sourceChartVisible" :data="sourceData" />
          <div v-else class="flex min-h-[320px] items-center justify-center text-sm text-[var(--tg-muted)]">
            图表将在进入视口后加载
          </div>
        </div>
      </div>
    </div>

    <DashboardEndpointTable :data="endpointData" />
  </section>
</template>

<script setup lang="ts">
import type { EndpointItem, SourceItem, StatsDashboard, StatsSummary, TokenItem, TrendItem } from '~/composables/useDashboard'

definePageMeta({ layout: 'dashboard', middleware: 'auth' })
const { apiData } = useApi()
const days = ref(30)
const tokenId = ref('')
const emptySummary: StatsSummary = { totalRequests: 0, successRequests: 0, errorRequests: 0, avgLatencyMs: 0, uniqueOrigins: 0, uniqueIPs: 0 }
const statsQuery = computed(() => ({ days: days.value, tokenId: tokenId.value || undefined }))
const [{ data: tokensResponse }, { data: statsResponse, refresh: refreshStats }] = await Promise.all([
  apiData<TokenItem[]>('dashboard:stats:tokens', '/tokens'),
  apiData<StatsDashboard>('dashboard:stats', '/dashboard/stats', {
    query: statsQuery,
    dedupe: 'cancel'
  })
])
const tokens = computed(() => tokensResponse.value?.success ? tokensResponse.value.data : [])
const statsData = computed(() => statsResponse.value?.success ? statsResponse.value.data : null)
const summaryData = computed<StatsSummary>(() => statsData.value?.summary ?? emptySummary)
const trendData = computed<TrendItem[]>(() => statsData.value?.trend ?? [])
const sourceData = computed<SourceItem[]>(() => statsData.value?.sources ?? [])
const endpointData = computed<EndpointItem[]>(() => statsData.value?.endpoints ?? [])

let statsDebounceTimer: ReturnType<typeof setTimeout> | null = null

const trendChartHost = ref<HTMLElement | null>(null)
const sourceChartHost = ref<HTMLElement | null>(null)
const trendChartVisible = ref(false)
const sourceChartVisible = ref(false)

let chartObserver: IntersectionObserver | null = null

const revealAllCharts = () => {
  trendChartVisible.value = true
  sourceChartVisible.value = true
}

const revealChart = (target: Element) => {
  if (target === trendChartHost.value) trendChartVisible.value = true
  if (target === sourceChartHost.value) sourceChartVisible.value = true
}

const setupChartObserver = () => {
  if (!('IntersectionObserver' in window)) {
    revealAllCharts()
    return
  }

  chartObserver = new IntersectionObserver((entries) => {
    for (const entry of entries) {
      if (!entry.isIntersecting) continue
      revealChart(entry.target)
      chartObserver?.unobserve(entry.target)
    }
  }, { rootMargin: '240px 0px', threshold: 0.01 })

  if (trendChartHost.value) chartObserver.observe(trendChartHost.value)
  if (sourceChartHost.value) chartObserver.observe(sourceChartHost.value)
}

const loadStats = async () => {
  await refreshStats()
}

const scheduleStatsLoad = () => {
  if (statsDebounceTimer) clearTimeout(statsDebounceTimer)
  statsDebounceTimer = setTimeout(() => {
    statsDebounceTimer = null
    void loadStats()
  }, 200)
}

watch([days, tokenId], scheduleStatsLoad)
onMounted(() => {
  setupChartObserver()
})
onBeforeUnmount(() => {
  if (statsDebounceTimer) clearTimeout(statsDebounceTimer)
  chartObserver?.disconnect()
  chartObserver = null
})
</script>
