<template>
  <section class="grid gap-6">
    <div class="flex flex-wrap items-center justify-between gap-3">
      <h1 class="text-4xl font-black">请求统计</h1>
      <div class="flex gap-3"><select v-model.number="days" class="rounded-xl border border-white/10 bg-slate-950 px-4 py-3 text-white"><option :value="7">7 天</option><option :value="14">14 天</option><option :value="30">30 天</option><option :value="90">90 天</option></select><select v-model="tokenId" class="rounded-xl border border-white/10 bg-slate-950 px-4 py-3 text-white"><option value="">全部 token</option><option v-for="token in tokens" :key="token.id" :value="token.id">{{ token.name }}</option></select></div>
    </div>
    <DashboardRequestSummaryCards :summary="summaryData" />
    <div class="grid gap-6 xl:grid-cols-2"><div class="rounded-3xl border border-white/10 bg-white/[0.06] p-6"><h3 class="text-xl font-black">请求趋势 / 成功失败</h3><DashboardRequestTrendChart :data="trendData" /></div><div class="rounded-3xl border border-white/10 bg-white/[0.06] p-6"><h3 class="text-xl font-black">请求来源</h3><DashboardSourcePieChart :data="sourceData" /></div></div>
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
