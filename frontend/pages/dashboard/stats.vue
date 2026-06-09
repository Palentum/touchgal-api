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

const load = async () => {
  const id = tokenId.value || undefined
  const [summaryRes, trendRes, sourceRes, endpointRes, tokenRes] = await Promise.all([dash.summary(days.value, id), dash.trend(days.value, id), dash.sources(days.value, id), dash.endpoints(days.value, id), dash.tokens()])
  if (summaryRes.success) summaryData.value = summaryRes.data
  if (trendRes.success) trendData.value = trendRes.data
  if (sourceRes.success) sourceData.value = sourceRes.data
  if (endpointRes.success) endpointData.value = endpointRes.data
  if (tokenRes.success) tokens.value = tokenRes.data
}

watch([days, tokenId], load)
onMounted(load)
</script>
