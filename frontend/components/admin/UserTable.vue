<template>
  <div class="tg-card">
    <div class="flex flex-wrap items-start justify-between gap-4">
      <div>
        <p class="tg-eyebrow">User List</p>
        <h2 class="tg-title-lg">用户列表</h2>
      </div>
      <span class="tg-badge tg-badge-coral">{{ props.users.length }} 个</span>
    </div>

    <div class="tg-table-wrap mt-6">
      <table class="tg-table tg-user-table">
        <thead>
          <tr>
            <th>账号</th>
            <th class="text-center">状态</th>
            <th class="text-center">权限</th>
            <th>限流</th>
            <th>最近登录</th>
            <th>创建时间</th>
            <th class="text-right" aria-label="操作"></th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="user in props.users" :key="user.id">
            <td>
              <p class="tg-title-sm">{{ user.displayName || '未设置昵称' }}</p>
              <p class="tg-muted mt-1">{{ user.email }}</p>
            </td>
            <td class="text-center">
              <span class="tg-badge" :class="statusBadgeClass(user.status)">{{ statusText(user.status) }}</span>
            </td>
            <td class="text-center">
              <span class="tg-badge" :class="user.isAdmin ? 'tg-badge-warning' : ''">{{ user.isAdmin ? '管理员' : '开发者' }}</span>
            </td>
            <td class="tg-muted">{{ user.minuteLimit }}/{{ user.dailyLimit }}</td>
            <td class="tg-muted">{{ formatDateTime(user.lastLoginAt) }}</td>
            <td class="tg-muted">{{ formatDateTime(user.createdAt) }}</td>
            <td>
              <div class="flex flex-wrap justify-end gap-2">
                <button
                  type="button"
                  class="tg-icon-btn"
                  :aria-label="`编辑 ${user.displayName || user.email}`"
                  title="编辑"
                  :disabled="isBusy(user)"
                  @click="emit('edit', user)"
                >
                  <svg viewBox="0 0 24 24" aria-hidden="true" class="h-4 w-4" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round">
                    <path d="M12 20h9" />
                    <path d="M16.5 3.5a2.1 2.1 0 0 1 3 3L7 19l-4 1 1-4Z" />
                  </svg>
                </button>
                <button
                  type="button"
                  :class="user.status === 'active' ? 'tg-icon-btn text-red-700' : 'tg-icon-btn text-[var(--tg-success)]'"
                  :aria-label="`${user.status === 'active' ? '停用' : '启用'} ${user.displayName || user.email}`"
                  :title="user.status === 'active' ? '停用' : '启用'"
                  :disabled="isSelf(user) || isBusy(user)"
                  @click="toggleStatus(user)"
                >
                  <template v-if="user.status === 'active'">
                    <svg viewBox="0 0 24 24" aria-hidden="true" class="h-4 w-4" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round">
                      <circle cx="12" cy="12" r="8.5" />
                      <path d="m6.5 6.5 11 11" />
                    </svg>
                  </template>
                  <template v-else>
                    <svg viewBox="0 0 24 24" aria-hidden="true" class="h-4 w-4" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round">
                      <circle cx="12" cy="12" r="8.5" />
                      <path d="m8 12.5 3 3 5.5-7" />
                    </svg>
                  </template>
                </button>
                <button
                  type="button"
                  class="tg-icon-btn text-red-700"
                  :aria-label="`删除 ${user.displayName || user.email}`"
                  title="删除"
                  :disabled="isSelf(user) || isBusy(user)"
                  @click="emit('delete', user)"
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
      <p v-if="props.users.length === 0" class="tg-empty">暂无符合条件的用户。</p>
    </div>
  </div>
</template>

<script setup lang="ts">
export type UserStatus = 'active' | 'disabled'

export interface AdminUser {
  id: string
  email: string
  displayName: string
  status: UserStatus
  isAdmin: boolean
  minuteLimit: number
  dailyLimit: number
  lastLoginAt?: string
  createdAt: string
  updatedAt: string
}

export interface AdminUserPatch {
  email?: string
  displayName?: string
  status?: UserStatus
  minuteLimit?: number
  dailyLimit?: number
}

const props = withDefaults(defineProps<{
  users: AdminUser[]
  currentUserId?: string
  busyUserId?: string | null
}>(), {
  currentUserId: '',
  busyUserId: null
})

const emit = defineEmits<{ update: [id: string, patch: AdminUserPatch]; edit: [user: AdminUser]; delete: [user: AdminUser] }>()

const isSelf = (user: AdminUser) => user.id === props.currentUserId
const isBusy = (user: AdminUser) => props.busyUserId === user.id

const formatDateTime = (value?: string) => value ? value.slice(0, 19).replace('T', ' ') : '未登录'
const statusText = (status: UserStatus) => status === 'active' ? '正常' : '停用'
const statusBadgeClass = (status: UserStatus) => status === 'active' ? 'tg-badge-success' : 'tg-badge-error'

const toggleStatus = (user: AdminUser) => {
  emit('update', user.id, { status: user.status === 'active' ? 'disabled' : 'active' })
}

</script>

<style scoped>
.tg-user-table :deep(th),
.tg-user-table :deep(td) {
  vertical-align: middle;
}
</style>
