<template>
  <div class="rounded-3xl border border-white/10 bg-white/[0.06] p-6">
    <h3 class="text-xl font-black">同步记录</h3>
    <table class="mt-4 w-full text-left text-sm">
      <thead class="text-slate-400"><tr><th class="py-2">模式</th><th>状态</th><th>Upsert</th><th>Deleted</th><th>错误</th><th>时间</th></tr></thead>
      <tbody><tr v-for="run in runs" :key="run.id" class="border-t border-white/10"><td class="py-3">{{ run.mode }}</td><td>{{ run.status }}</td><td>{{ run.gamesUpserted }}</td><td>{{ run.gamesDeleted }}</td><td class="max-w-md truncate text-rose-300">{{ run.errorMessage }}</td><td>{{ run.startedAt }}</td></tr></tbody>
    </table>
  </div>
</template>
<script setup lang="ts">
export interface SyncRun { id: string; mode: string; status: string; gamesUpserted: number; gamesDeleted: number; errorMessage: string; startedAt: string }
defineProps<{ runs: SyncRun[] }>()
</script>
