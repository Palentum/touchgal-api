<template>
  <section class="grid gap-6">
    <div class="flex items-center justify-between"><h1 class="text-4xl font-black">同步状态</h1><div class="flex gap-3"><button class="rounded-xl bg-emerald-400 px-4 py-3 font-bold text-slate-950" @click="run('incremental')">运行 incremental</button><button class="rounded-xl bg-emerald-400 px-4 py-3 font-bold text-slate-950" @click="run('full')">运行 full</button></div></div>
    <AdminSyncRunTable :runs="runs" />
  </section>
</template>
<script setup lang="ts">
import type { SyncRun } from '~/components/admin/SyncRunTable.vue'
definePageMeta({ layout: 'dashboard', middleware: 'admin' })
const { apiFetch } = useApi()
const runs = ref<SyncRun[]>([])
const load = async () => { const res = await apiFetch<SyncRun[]>('/admin/sync/runs'); if (res.success) runs.value = res.data }
const run = async (mode: 'incremental' | 'full') => { await apiFetch('/admin/sync/run', { method: 'POST', body: { mode } }); await load() }
onMounted(load)
</script>
