<template>
  <section class="tg-dashboard-stack">
    <div class="flex flex-wrap items-end justify-between gap-4">
      <div>
        <p class="tg-eyebrow">Analysis</p>
        <h1 class="tg-display-md">请求统计</h1>
        <p class="tg-lead">按时间窗口与 token 查看调用质量、来源与 endpoint 表现。</p>
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
            <option value="">全部 token</option>
            <option v-for="token in tokens" :key="token.id" :value="token.id">{{ token.name }}</option>
          </select>
        </label>
      </div>
    </div>

    <DashboardRequestSummaryCards :summary="summaryData" />

    <div class="tg-grid-2">
      <div class="tg-card">
        <h2 class="tg-title-lg">请求趋势 / 成功失败</h2>
        <div class="tg-chart-box mt-4">
          <DashboardRequestTrendChart :data="trendData" />
        </div>
      </div>
      <div class="tg-card">
        <h2 class="tg-title-lg">请求来源</h2>
        <div class="tg-chart-box mt-4">
          <DashboardSourcePieChart :data="sourceData" />
        </div>
      </div>
    </div>

    <DashboardEndpointTable :data="endpointData" />
  </section>
</template>

<script setup lang="ts">
import type { EndpointItem, SourceItem, StatsSummary, TokenItem, TrendItem } from '~/composables/useDashboard'

definePageMeta({ layout: 'dashboard', middleware: 'auth' })
const dash = useDashboard()
const days = ref(30)
const tokenId = ref('')
const tokens = ref<TokenItem[]>([])
const summaryData = ref<StatsSummary>({ totalRequests: 0, successRequests: 0, errorRequests: 0, avgLatencyMs: 0, uniqueOrigins: 0, uniqueIPs: 0 })
const trendData = ref<TrendItem[]>([])
const sourceData = ref<SourceItem[]>([])
const endpointData = ref<EndpointItem[]>([])

let statsDebounceTimer: ReturnType<typeof setTimeout> | null = null
let statsRequestSeq = 0

const loadTokens = async () => {
  const tokenRes = await dash.tokens()
  if (tokenRes.success) tokens.value = tokenRes.data
}

const loadStats = async () => {
  const requestSeq = ++statsRequestSeq
  const id = tokenId.value || undefined
  const res = await dash.stats(days.value, id)
  if (requestSeq !== statsRequestSeq) return
  if (res.success) {
    summaryData.value = res.data.summary
    trendData.value = res.data.trend
    sourceData.value = res.data.sources
    endpointData.value = res.data.endpoints
  }
}

const scheduleStatsLoad = () => {
  statsRequestSeq++
  if (statsDebounceTimer) clearTimeout(statsDebounceTimer)
  statsDebounceTimer = setTimeout(() => {
    statsDebounceTimer = null
    void loadStats()
  }, 200)
}

watch([days, tokenId], scheduleStatsLoad)
onMounted(() => {
  void loadTokens()
  void loadStats()
})
onBeforeUnmount(() => {
  if (statsDebounceTimer) clearTimeout(statsDebounceTimer)
  statsRequestSeq++
})
</script>
