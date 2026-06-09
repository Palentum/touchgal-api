<template>
  <section class="tg-dashboard-stack">
    <div>
      <p class="tg-eyebrow">申请审核</p>
      <h1 class="tg-display-md">申请审核</h1>
      <p class="tg-lead">按项目查看开发者 API 访问申请，并执行管理员审核动作。</p>
    </div>
    <AdminApplicationReviewTable :applications="applications" @review="review" />
  </section>
</template>
<script setup lang="ts">
import type { ApplicationItem } from '~/composables/useDashboard'
definePageMeta({ layout: 'admin', middleware: 'admin' })
const { apiFetch } = useApi()
const applications = ref<ApplicationItem[]>([])
const load = async () => { const res = await apiFetch<ApplicationItem[]>('/admin/applications', { query: { page: 1, limit: 50 } }); if (res.success) applications.value = res.data }
const review = async (id: string, action: 'approve' | 'reject' | 'revoke') => { await apiFetch(`/admin/applications/${id}/${action}`, { method: 'POST', body: { minuteLimit: 60, dailyLimit: 5000 } }); await load() }
onMounted(load)
</script>
