<template>
  <section class="tg-dashboard-stack">
    <div>
      <p class="tg-eyebrow">OVERVIEW</p>
      <h1 class="tg-display-md">概览</h1>
    </div>

    <DashboardRequestSummaryCards :summary="summaryData" />

    <div class="tg-grid-2 tg-dashboard-equal-row">
      <div class="tg-card tg-quick-entry-card">
        <p class="tg-eyebrow">快捷入口</p>
        <h2 class="tg-title-lg">快捷入口</h2>
        <p class="tg-muted mt-3">从这里进入 token、统计与 API 调试台。</p>
        <div class="tg-actions tg-quick-entry-actions">
          <NuxtLink class="tg-btn tg-btn-primary" to="/dashboard/tokens">生成 token</NuxtLink>
          <NuxtLink class="tg-btn tg-btn-secondary" to="/dashboard/stats">查看统计</NuxtLink>
          <NuxtLink class="tg-btn tg-btn-secondary" to="/dashboard/console">API 调试台</NuxtLink>
        </div>
      </div>

      <div class="tg-card">
        <p class="tg-eyebrow">账户限额</p>
        <h2 class="tg-title-lg">账户限额</h2>
        <p class="tg-muted mt-3">账号级上限会作为 API token 的有效限流上限。</p>
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
definePageMeta({ layout: 'dashboard', middleware: 'auth' })
const auth = useAuthStore()
const dash = useDashboard()
const access = useApplicationAccess()
const apps = access.applications
const summaryData = ref({ totalRequests: 0, successRequests: 0, errorRequests: 0, avgLatencyMs: 0, uniqueOrigins: 0, uniqueIPs: 0 })
const formatLimit = (value?: number) => typeof value === 'number' && Number.isFinite(value) ? value.toLocaleString('zh-CN') : '—'

onMounted(async () => {
  const [, statsRes] = await Promise.all([access.fetchApplications(auth.user?.id), dash.stats(7)])
  if (statsRes.success) summaryData.value = statsRes.data.summary
})
</script>

<style scoped>
.tg-dashboard-equal-row {
  align-items: stretch;
}

.tg-quick-entry-card {
  display: flex;
  flex-direction: column;
}

.tg-quick-entry-actions {
  margin-top: auto;
  padding-top: 24px;
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
