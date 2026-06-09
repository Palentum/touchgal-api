<template>
  <div class="tg-card-dark">
    <div class="flex flex-wrap items-start justify-between gap-4">
      <div>
        <p class="tg-eyebrow">Token 列表</p>
        <h2 class="tg-title-lg">Token 列表</h2>
      </div>
      <span class="tg-badge tg-badge-coral">{{ tokens.length }} 个</span>
    </div>

    <div class="tg-table-wrap mt-6">
      <table class="tg-table">
        <thead>
          <tr>
            <th>名称</th>
            <th>状态</th>
            <th>限流</th>
            <th>创建日期</th>
            <th>上次使用</th>
            <th>操作</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="token in tokens" :key="token.id">
            <td class="font-semibold">{{ token.name }}</td>
            <td><span class="tg-badge" :class="token.status === 'active' ? 'tg-badge-success' : 'tg-badge-error'">{{ statusText(token.status) }}</span></td>
            <td>{{ token.minuteLimit }}/min · {{ token.dailyLimit }}/day</td>
            <td class="tg-muted">{{ formatDateTime(token.createdAt) }}</td>
            <td class="tg-muted">{{ token.lastUsedAt ? formatDateTime(token.lastUsedAt) : '未使用' }}</td>
            <td>
              <button v-if="token.status === 'active'" class="tg-btn tg-btn-danger" @click="$emit('revoke', token.id)">失效</button>
            </td>
          </tr>
        </tbody>
      </table>
      <p v-if="tokens.length === 0" class="tg-empty">暂无 token。</p>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { TokenItem } from '~/composables/useDashboard'

const formatDateTime = (value: string) => value.slice(0, 19).replace('T', ' ')

const statusText = (status: string) => status === 'active' ? '启用中' : '已失效'
defineProps<{ tokens: TokenItem[] }>()
defineEmits<{ revoke: [id: string] }>()
</script>
