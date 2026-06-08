<template>
  <div class="overflow-hidden rounded-3xl border border-white/10 bg-white/[0.06]">
    <table class="w-full text-left text-sm">
      <thead class="bg-white/5 text-slate-300"><tr><th class="p-4">项目</th><th class="p-4">状态</th><th class="p-4">请求量</th><th class="p-4">审核</th></tr></thead>
      <tbody>
        <tr v-for="app in applications" :key="app.id" class="border-t border-white/10">
          <td class="p-4"><p class="font-bold">{{ app.projectName || app.applicantName }}</p><p class="text-slate-400">{{ app.projectUrl }}</p></td>
          <td class="p-4">{{ app.status }}</td>
          <td class="p-4">{{ app.expectedDailyRequests }}</td>
          <td class="p-4 flex gap-2"><button class="rounded-lg px-3 py-2 text-xs font-bold text-slate-950 bg-emerald-400" @click="$emit('review', app.id, 'approve')">批准</button><button class="rounded-lg px-3 py-2 text-xs font-bold text-slate-950 bg-amber-300" @click="$emit('review', app.id, 'reject')">拒绝</button><button class="rounded-lg px-3 py-2 text-xs font-bold text-slate-950 bg-rose-400" @click="$emit('review', app.id, 'revoke')">撤销</button></td>
        </tr>
      </tbody>
    </table>
  </div>
</template>
<script setup lang="ts">
import type { ApplicationItem } from '~/composables/useDashboard'
defineProps<{ applications: ApplicationItem[] }>()
defineEmits<{ review: [id: string, action: 'approve' | 'reject' | 'revoke'] }>()
</script>
