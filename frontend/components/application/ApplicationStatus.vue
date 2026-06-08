<template>
  <div class="rounded-3xl border border-white/10 bg-white/[0.06] p-6">
    <h3 class="text-xl font-black">账号申请状态</h3>
    <div class="mt-4 grid gap-3">
      <div v-for="app in applications" :key="app.id" class="rounded-2xl bg-slate-950/70 p-4">
        <div class="flex items-center justify-between gap-4">
          <div><p class="font-bold">{{ app.projectName || app.projectUrl }}</p><p class="text-sm text-slate-400">{{ app.projectUrl }}</p></div>
          <span class="rounded-full px-3 py-1 text-xs" :class="badge(app.status)">{{ app.status }}</span>
        </div>
        <p v-if="app.reviewNote" class="mt-3 text-sm text-slate-300">{{ app.reviewNote }}</p>
      </div>
      <p v-if="applications.length === 0" class="text-slate-400">暂无账号申请。</p>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { ApplicationItem } from '~/composables/useDashboard'
defineProps<{ applications: ApplicationItem[] }>()
const badge = (status: string) => status === 'approved' ? 'bg-emerald-400 text-slate-950' : status === 'pending' ? 'bg-amber-300 text-slate-950' : 'bg-rose-400 text-white'
</script>
