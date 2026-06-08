<template>
  <section class="grid gap-6">
    <div>
      <h1 class="text-4xl font-black">开发者后台</h1>
      <p class="mt-2 text-slate-400">当前用户：{{ auth.user?.email }}</p>
    </div>
    <DashboardRequestSummaryCards :summary="summaryData" />
    <div class="grid gap-6 lg:grid-cols-2">
      <ApplicationApplicationStatus :applications="apps" />
      <div class="rounded-3xl border border-white/10 bg-white/[0.06] p-6">
        <h3 class="text-xl font-black">快捷入口</h3>
        <div class="mt-5 grid gap-3"><NuxtLink class="rounded-2xl bg-slate-950 px-4 py-3 text-emerald-200" to="/dashboard/tokens">生成 token</NuxtLink><NuxtLink class="rounded-2xl bg-slate-950 px-4 py-3 text-emerald-200" to="/dashboard/stats">查看统计</NuxtLink><NuxtLink class="rounded-2xl bg-slate-950 px-4 py-3 text-emerald-200" to="/dashboard/console">API 调试台</NuxtLink></div>
      </div>
    </div>
  </section>
</template>
<script setup lang="ts">
import type { ApplicationItem } from '~/composables/useDashboard'
definePageMeta({ layout: 'dashboard', middleware: 'auth' })
const auth = useAuthStore()
const dash = useDashboard()
const apps = ref<ApplicationItem[]>([])
const summaryData = ref({ totalRequests: 0, successRequests: 0, errorRequests: 0, avgLatencyMs: 0, uniqueOrigins: 0, uniqueIPs: 0 })
onMounted(async () => {
  const [appRes, summaryRes] = await Promise.all([dash.applications(), dash.summary(7)])
  if (appRes.success) apps.value = appRes.data
  if (summaryRes.success) summaryData.value = summaryRes.data
})
</script>
