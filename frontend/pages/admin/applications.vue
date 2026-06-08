<template>
  <section class="grid gap-6">
    <h1 class="text-4xl font-black">申请审核</h1>
    <AdminApplicationReviewTable :applications="applications" @review="review" />
  </section>
</template>
<script setup lang="ts">
import type { ApplicationItem } from '~/composables/useDashboard'
definePageMeta({ layout: 'dashboard', middleware: 'admin' })
const { apiFetch } = useApi()
const applications = ref<ApplicationItem[]>([])
const load = async () => { const res = await apiFetch<ApplicationItem[]>('/admin/applications', { query: { page: 1, limit: 50 } }); if (res.success) applications.value = res.data }
const review = async (id: string, action: 'approve' | 'reject' | 'revoke') => { await apiFetch(`/admin/applications/${id}/${action}`, { method: 'POST', body: { minuteLimit: 60, dailyLimit: 5000, reviewNote: action === 'approve' ? 'Approved' : 'Reviewed' } }); await load() }
onMounted(load)
</script>
