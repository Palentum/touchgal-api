<template>
  <section class="tg-dashboard-stack">
    <div class="tg-card-dark flex flex-wrap items-center justify-between gap-4">
      <div>
        <p class="tg-eyebrow">Sync Status</p>
        <h1 class="tg-display-md">同步状态</h1>
        <p class="tg-muted mt-3">触发 clean DB 同步并查看最近运行记录。</p>
      </div>
      <div>
        <div class="tg-actions">
          <button class="tg-btn tg-btn-cream" :disabled="pendingMode !== null" @click="run('incremental')">
            {{ pendingMode === 'incremental' ? '正在触发增量同步...' : '运行增量同步' }}
          </button>
          <button class="tg-btn tg-btn-secondary-dark" :disabled="pendingMode !== null" @click="run('full')">
            {{ pendingMode === 'full' ? '正在触发全量同步...' : '运行全量同步' }}
          </button>
        </div>
        <p v-if="errorMessage" class="tg-message-error mt-3">{{ errorMessage }}</p>
      </div>
    </div>
    <AdminSyncRunTable :runs="runs" />
  </section>
</template>
<script setup lang="ts">
import type { SyncRun } from '~/components/admin/SyncRunTable.vue'
definePageMeta({ layout: 'admin', middleware: 'admin' })
const { apiFetch, apiData } = useApi()
type SyncMode = 'incremental' | 'full'

const { data: runsResponse, refresh: refreshRuns } = await apiData<SyncRun[]>('admin:sync:runs', '/admin/sync/runs', { dedupe: 'cancel' })
const runs = ref<SyncRun[]>(runsResponse.value?.success ? runsResponse.value.data : [])
const pendingMode = ref<SyncMode | null>(null)
const errorMessage = ref(runsResponse.value && !runsResponse.value.success ? runsResponse.value.error.message : '')
let pollTimer: ReturnType<typeof setTimeout> | null = null
let disposed = false

const hasRunningRun = computed(() => runs.value.some(run => run.status === 'running'))

const clearPoll = () => {
  if (pollTimer) {
    clearTimeout(pollTimer)
    pollTimer = null
  }
}

const queuePoll = () => {
  if (disposed || !hasRunningRun.value || pollTimer) return
  pollTimer = setTimeout(() => {
    pollTimer = null
    void load()
  }, 3000)
}

const upsertRun = (run: SyncRun) => {
  runs.value = [run, ...runs.value.filter(item => item.id !== run.id)].slice(0, 50)
}

const load = async () => {
  if (disposed) return
  try {
    await refreshRuns()
    if (disposed) return
    const res = runsResponse.value
    if (res?.success) {
      runs.value = res.data
      if (hasRunningRun.value) queuePoll()
      else clearPoll()
    } else if (res) {
      errorMessage.value = res.error.message
    }
  } catch {
    if (!disposed) {
      errorMessage.value = '刷新同步记录失败'
      if (hasRunningRun.value) queuePoll()
    }
  }
}

const run = async (mode: SyncMode) => {
  if (pendingMode.value) return
  pendingMode.value = mode
  errorMessage.value = ''
  try {
    const res = await apiFetch<SyncRun>('/admin/sync/run', { method: 'POST', body: { mode } })
    if (disposed) return
    if (res.success) {
      upsertRun(res.data)
      queuePoll()
      void load()
    } else {
      errorMessage.value = res.error.message
    }
  } catch (error: any) {
    if (!disposed) errorMessage.value = error?.data?.error?.message || '触发同步失败'
  } finally {
    if (!disposed) pendingMode.value = null
  }
}

onMounted(() => {
  disposed = false
  if (hasRunningRun.value) queuePoll()
})
onUnmounted(() => {
  disposed = true
  clearPoll()
})
</script>
