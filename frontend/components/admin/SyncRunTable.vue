<template>
  <div class="tg-card">
    <div class="mb-4 flex flex-wrap items-start justify-between gap-4">
      <div>
        <p class="tg-eyebrow">Sync Record</p>
        <h3 class="tg-title-lg">同步记录</h3>
      </div>
      <span class="tg-badge tg-badge-coral">{{ runs.length }} 次运行</span>
    </div>
    <div class="tg-table-wrap">
      <table class="tg-table">
        <thead>
          <tr>
            <th>模式</th>
            <th class="text-center">状态</th>
            <th>写入/更新</th>
            <th>删除</th>
            <th>错误</th>
            <th>时间</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="run in runs" :key="run.id">
            <td>{{ modeText(run.mode) }}</td>
            <td class="text-center">
              <span class="tg-badge" :class="statusBadgeClass(run.status)">{{ run.status }}</span>
            </td>
            <td>{{ run.gamesUpserted }}</td>
            <td>{{ run.gamesDeleted }}</td>
            <td class="max-w-md truncate">{{ run.errorMessage || '—' }}</td>
            <td>{{ formatDateTime(run.startedAt) }}</td>
          </tr>
          <tr v-if="runs.length === 0">
            <td class="tg-empty" colspan="6">暂无同步记录。</td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>
<script setup lang="ts">
export interface SyncRun { id: string; mode: string; status: string; gamesUpserted: number; gamesDeleted: number; errorMessage: string; startedAt: string }
defineProps<{ runs: SyncRun[] }>()

const modeText = (mode: string) => {
  if (mode === 'full') return '全量'
  if (mode === 'incremental') return '增量'
  return mode
}

const formatDateTime = (value: string) => value.slice(0, 19).replace('T', ' ')

const statusBadgeClass = (status: string) => {
  if (status === 'success' || status === 'completed') return 'tg-badge-success'
  if (status === 'failed' || status === 'error') return 'tg-badge-error'
  return 'tg-badge-warning'
}
</script>
