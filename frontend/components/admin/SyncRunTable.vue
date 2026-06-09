<template>
  <div class="tg-card">
    <div class="mb-4 flex items-center justify-between gap-4">
      <h3 class="tg-title-lg">同步记录</h3>
      <span class="tg-badge tg-badge-coral">{{ runs.length }} 次运行</span>
    </div>
    <div class="tg-table-wrap">
      <table class="tg-table">
        <thead>
          <tr>
            <th>模式</th>
            <th>状态</th>
            <th>写入/更新</th>
            <th>删除</th>
            <th>错误</th>
            <th>时间</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="run in runs" :key="run.id">
            <td>{{ run.mode }}</td>
            <td>
              <span class="tg-badge" :class="statusBadgeClass(run.status)">{{ run.status }}</span>
            </td>
            <td>{{ run.gamesUpserted }}</td>
            <td>{{ run.gamesDeleted }}</td>
            <td class="max-w-md truncate">{{ run.errorMessage || '—' }}</td>
            <td>{{ run.startedAt }}</td>
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

const statusBadgeClass = (status: string) => {
  if (status === 'success' || status === 'completed') return 'tg-badge-success'
  if (status === 'failed' || status === 'error') return 'tg-badge-error'
  return 'tg-badge-warning'
}
</script>
