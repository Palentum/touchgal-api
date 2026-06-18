<template>
  <section class="tg-dashboard-stack">
    <div>
      <p class="tg-eyebrow">OVERVIEW</p>
      <h1 class="tg-display-md">概览</h1>
    </div>

    <DashboardRequestSummaryCards :summary="summaryData" />

    <div class="tg-grid-2 tg-dashboard-equal-row">
      <div class="tg-card tg-endpoint-url-card">
        <p class="tg-eyebrow">Endpoint</p>
        <h2 class="tg-title-lg">接口地址</h2>
        <div class="tg-endpoint-url-box mt-6">
          <span class="tg-badge tg-badge-coral tg-endpoint-region">全球</span>
          <code class="tg-endpoint-url font-mono">{{ apiBaseUrl }}</code>
        </div>
      </div>

      <div class="tg-card">
        <p class="tg-eyebrow">Limits</p>
        <h2 class="tg-title-lg">账户限额</h2>
        <p class="tg-muted mt-3">账户限额以账户为主体计算，账户限额会自动重置。</p>
        <div class="tg-actions mt-6">
          <div class="tg-limit-pill">
            <p class="tg-stat-label">每分钟请求</p>
            <p class="tg-limit-value">{{ formatLimit(auth.user?.minuteLimit) }}</p>
          </div>
          <div class="tg-limit-pill">
            <p class="tg-stat-label">每日请求</p>
            <p class="tg-limit-value">{{ formatLimit(auth.user?.dailyLimit) }}</p>
          </div>
        </div>
      </div>
    </div>

    <ApplicationApplicationStatus :applications="apps" />
  </section>
</template>

<script setup lang="ts">
import type { StatsDashboard, StatsSummary } from '~/composables/useDashboard'

definePageMeta({ layout: 'dashboard', middleware: 'auth' })
const auth = useAuthStore()
const { apiData, baseURL } = useApi()
// 公开 v1 接口根地址：baseURL 已去掉尾部 /，追加 /v1/ 作为调用基础地址
const apiBaseUrl = computed(() => `${baseURL}/v1/`)
const access = useApplicationAccess()
const apps = access.applications
const emptySummary: StatsSummary = { totalRequests: 0, successRequests: 0, errorRequests: 0, avgLatencyMs: 0, uniqueOrigins: 0, uniqueIPs: 0 }
const { data: statsResponse } = await apiData<StatsDashboard>('dashboard:index:stats:7', '/dashboard/stats', { query: { days: 7 } })
const summaryData = computed(() => statsResponse.value?.success ? statsResponse.value.data.summary : emptySummary)
const formatLimit = (value?: number) => typeof value === 'number' && Number.isFinite(value) ? value.toLocaleString('zh-CN') : '—'
</script>

<style scoped>
.tg-dashboard-equal-row {
  align-items: stretch;
}

.tg-endpoint-url-card {
  display: flex;
  flex-direction: column;
}

.tg-endpoint-url-box {
  display: flex;
  align-items: center;
  gap: 12px;
  border: 1px solid var(--tg-hairline);
  border-radius: 10px;
  background: var(--tg-canvas);
  padding: 12px 14px;
}

.tg-endpoint-region {
  flex-shrink: 0;
}

.tg-endpoint-url {
  flex: 1;
  min-width: 0;
  color: var(--tg-body-strong);
  font-size: 14px;
  overflow-wrap: anywhere;
}

.tg-limit-pill {
  min-width: 132px;
  border: 1px solid var(--tg-hairline);
  border-radius: 10px;
  background: var(--tg-canvas);
  padding: 14px 16px;
}

.tg-limit-value {
  margin: 6px 0 0;
  color: var(--tg-body-strong);
  font-family: var(--tg-font-display);
  font-size: 28px;
  font-weight: 400;
  letter-spacing: -0.02em;
  line-height: 1.1;
}
</style>
