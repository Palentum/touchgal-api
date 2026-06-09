<template>
  <section class="tg-dashboard-stack">
    <div>
      <p class="tg-eyebrow">开发者门户</p>
      <h1 class="tg-display-md">开发者后台</h1>
      <p class="tg-lead">当前用户：{{ auth.user?.email }}</p>
    </div>

    <DashboardRequestSummaryCards :summary="summaryData" />

    <div class="tg-grid-2">
      <div class="tg-card">
        <p class="tg-eyebrow">快捷入口</p>
        <h2 class="tg-title-lg">快捷入口</h2>
        <p class="tg-muted mt-3">从这里进入 token、统计与 API 调试台。</p>
        <div class="tg-actions mt-6">
          <NuxtLink class="tg-btn tg-btn-primary" to="/dashboard/tokens">生成 token</NuxtLink>
          <NuxtLink class="tg-btn tg-btn-secondary" to="/dashboard/stats">查看统计</NuxtLink>
          <NuxtLink class="tg-btn tg-btn-secondary" to="/dashboard/console">API 调试台</NuxtLink>
        </div>
      </div>

      <ApplicationApplicationStatus :applications="apps" />
    </div>
  </section>
</template>

<script setup lang="ts">
definePageMeta({ layout: 'dashboard', middleware: 'auth' })
const auth = useAuthStore()
const dash = useDashboard()
const access = useApplicationAccess()
const apps = access.applications
const summaryData = ref({ totalRequests: 0, successRequests: 0, errorRequests: 0, avgLatencyMs: 0, uniqueOrigins: 0, uniqueIPs: 0 })

onMounted(async () => {
  const [, summaryRes] = await Promise.all([access.fetchApplications(auth.user?.id), dash.summary(7)])
  if (summaryRes.success) summaryData.value = summaryRes.data
})
</script>
