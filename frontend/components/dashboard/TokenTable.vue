<template>
  <div class="overflow-hidden rounded-3xl border border-white/10 bg-white/[0.06]">
    <table class="w-full text-left text-sm">
      <thead class="bg-white/5 text-slate-300"><tr><th class="p-4">名称</th><th class="p-4">Prefix</th><th class="p-4">状态</th><th class="p-4">限流</th><th class="p-4">上次使用</th><th class="p-4">操作</th></tr></thead>
      <tbody>
        <tr v-for="token in tokens" :key="token.id" class="border-t border-white/10">
          <td class="p-4 font-semibold">{{ token.name }}</td>
          <td class="p-4 font-mono text-emerald-200">{{ token.tokenPrefix }}</td>
          <td class="p-4">{{ token.status }}</td>
          <td class="p-4">{{ token.minuteLimit }}/min · {{ token.dailyLimit }}/day</td>
          <td class="p-4 text-slate-400">{{ token.lastUsedAt || '未使用' }}</td>
          <td class="p-4"><button v-if="token.status === 'active'" class="rounded-lg bg-rose-400 px-3 py-2 text-slate-950" @click="$emit('revoke', token.id)">失效</button></td>
        </tr>
      </tbody>
    </table>
    <p v-if="tokens.length === 0" class="p-6 text-slate-400">暂无 token。</p>
  </div>
</template>
<script setup lang="ts">
import type { TokenItem } from '~/composables/useDashboard'
defineProps<{ tokens: TokenItem[] }>()
defineEmits<{ revoke: [id: string] }>()
</script>
