<template>
  <div :class="props.canEdit ? 'tg-card' : 'tg-card-dark'">
    <div class="flex flex-wrap items-start justify-between gap-4">
      <div>
        <p class="tg-eyebrow">Token List</p>
        <h2 class="tg-title-lg">Token 列表</h2>
      </div>
      <span class="tg-badge tg-badge-coral">{{ props.tokens.length }} 个</span>
    </div>

    <div class="tg-table-wrap mt-6" data-mobile-cards="true">
      <table class="tg-table" :class="{ 'tg-token-table-strong': props.canEdit, 'tg-token-table-owner': props.showOwner }">
        <thead>
          <tr>
            <th>名称</th>
            <th v-if="props.showOwner">所属账号</th>
            <template v-if="props.canEdit">
              <th>Tokens</th>
              <th>创建时间</th>
              <th>最新使用时间</th>
            </template>
            <template v-else>
              <th class="text-center">状态</th>
              <th>限流</th>
              <th>创建日期</th>
              <th>上次使用</th>
            </template>
            <th class="text-right">操作</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="token in props.tokens" :key="token.id">
            <td class="font-semibold" data-label="名称">{{ token.name }}</td>
            <td v-if="props.showOwner" data-label="所属账号">
              <p class="tg-title-sm tg-token-owner-name">{{ ownerDisplayName(token) }}</p>
              <p class="tg-muted mt-1">{{ ownerAccount(token) }}</p>
            </td>
            <template v-if="props.canEdit">
              <td data-label="Token">
                <code class="rounded-full border border-[var(--tg-hairline)] bg-[var(--tg-surface-soft)] px-3 py-1 text-xs font-semibold text-[var(--tg-body-strong)]">{{ maskedToken(token.tokenPrefix) }}</code>
              </td>
              <td data-label="创建时间">{{ formatDateTime(token.createdAt) }}</td>
              <td data-label="最新使用时间">{{ token.lastUsedAt ? formatDateTime(token.lastUsedAt) : '未使用' }}</td>
            </template>
            <template v-else>
              <td class="text-center" data-label="状态"><span class="tg-badge" :class="token.status === 'active' ? 'tg-badge-success' : 'tg-badge-error'">{{ statusText(token.status) }}</span></td>
              <td data-label="限流">{{ token.minuteLimit }}/{{ token.dailyLimit }}</td>
              <td class="tg-muted" data-label="创建日期">{{ formatDateTime(token.createdAt) }}</td>
              <td class="tg-muted" data-label="上次使用">{{ token.lastUsedAt ? formatDateTime(token.lastUsedAt) : '未使用' }}</td>
            </template>
            <td data-label="操作">
              <div class="flex justify-end gap-1 sm:gap-2">
                <button
                  v-if="props.canEdit"
                  type="button"
                  class="tg-icon-btn"
                  :aria-label="`编辑 ${token.name}`"
                  title="编辑"
                  @click="$emit('edit', token)"
                >
                  <svg viewBox="0 0 24 24" aria-hidden="true" class="h-4 w-4" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round">
                    <path d="M12 20h9" />
                    <path d="M16.5 3.5a2.1 2.1 0 0 1 3 3L7 19l-4 1 1-4Z" />
                  </svg>
                </button>
                <button
                  type="button"
                  class="tg-icon-btn text-red-700"
                  :aria-label="`删除 ${token.name}`"
                  title="删除"
                  @click="$emit('delete', token.id)"
                >
                  <svg viewBox="0 0 24 24" aria-hidden="true" class="h-4 w-4" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round">
                    <path d="M3 6h18" />
                    <path d="M8 6V4h8v2" />
                    <path d="m6 6 1 18h10l1-18" />
                    <path d="M10 11v6" />
                    <path d="M14 11v6" />
                  </svg>
                </button>
              </div>
            </td>
          </tr>
        </tbody>
      </table>
      <p v-if="props.tokens.length === 0" class="tg-empty">暂无 token。</p>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { TokenItem } from '~/composables/useDashboard'

const formatDateTime = (value: string) => value.slice(0, 19).replace('T', ' ')
const statusText = (status: string) => status === 'active' ? '启用中' : '不可用'

const maskedToken = (tokenPrefix: string) => {
  const visibleHead = tokenPrefix.slice(0, 18)
  const visibleTail = tokenPrefix.slice(18)
  const maskLength = Math.max(12, 53 - tokenPrefix.length)
  return `${visibleHead}${'*'.repeat(maskLength)}${visibleTail}`
}
const ownerDisplayName = (token: TokenItem) => token.owner?.displayName || '未设置昵称'
const ownerAccount = (token: TokenItem) => token.owner?.email || token.userId

const props = withDefaults(defineProps<{ tokens: TokenItem[]; canEdit?: boolean; showOwner?: boolean }>(), {
  canEdit: false,
  showOwner: false
})
defineEmits<{ edit: [token: TokenItem]; delete: [id: string] }>()
</script>

<style scoped>
.tg-token-table-strong,
.tg-token-table-strong thead {
  color: var(--tg-body-strong);
}

.tg-token-table-strong :deep(th),
.tg-token-table-strong :deep(td),
.tg-token-table-owner :deep(th),
.tg-token-table-owner :deep(td) {
  vertical-align: middle;
}

.tg-token-owner-name.tg-title-sm {
  color: var(--tg-body-strong);
}
</style>
