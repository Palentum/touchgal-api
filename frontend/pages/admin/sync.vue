<template>
  <section class="tg-dashboard-stack">
    <div class="tg-card-dark flex flex-wrap items-center justify-between gap-4">
      <div>
        <p class="tg-eyebrow">Sync Status</p>
        <h1 class="tg-display-md">同步状态</h1>
        <p class="tg-muted mt-3">触发 clean DB 同步并查看最近运行记录。</p>
      </div>
      <div class="tg-actions">
        <button class="tg-btn tg-btn-cream" @click="run('incremental')">运行增量同步</button>
        <button class="tg-btn tg-btn-secondary-dark" @click="run('full')">运行全量同步</button>
      </div>
    </div>
    <AdminSyncRunTable :runs="runs" />
  </section>
</template>
<script setup lang="ts">
import type { SyncRun } from '~/components/admin/SyncRunTable.vue'
definePageMeta({ layout: 'admin', middleware: 'admin' })
const { apiFetch } = useApi()
const runs = ref<SyncRun[]>([])
const load = async () => { const res = await apiFetch<SyncRun[]>('/admin/sync/runs'); if (res.success) runs.value = res.data }
const run = async (mode: 'incremental' | 'full') => { await apiFetch('/admin/sync/run', { method: 'POST', body: { mode } }); await load() }
onMounted(load)
</script>
