<template>
  <div class="tg-card">
    <div class="flex flex-wrap items-start justify-between gap-4">
      <div>
        <p class="tg-eyebrow">Users</p>
        <h2 class="tg-title-lg">用户列表</h2>
      </div>
      <span class="tg-badge tg-badge-coral">{{ props.users.length }} 个</span>
    </div>

    <div class="tg-table-wrap mt-6">
      <table class="tg-table tg-user-table">
        <thead>
          <tr>
            <th>账号</th>
            <th>状态</th>
            <th>权限</th>
            <th>最近登录</th>
            <th>创建时间</th>
            <th class="text-right">操作</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="user in props.users" :key="user.id">
            <td>
              <p class="tg-title-sm">{{ user.displayName || '未设置昵称' }}</p>
              <p class="tg-muted mt-1">{{ user.email }}</p>
              <span v-if="isSelf(user)" class="tg-badge mt-2">当前账号</span>
            </td>
            <td>
              <span class="tg-badge" :class="statusBadgeClass(user.status)">{{ statusText(user.status) }}</span>
            </td>
            <td>
              <span class="tg-badge" :class="user.isAdmin ? 'tg-badge-warning' : ''">{{ user.isAdmin ? '管理员' : '开发者' }}</span>
            </td>
            <td class="tg-muted">{{ formatDateTime(user.lastLoginAt) }}</td>
            <td class="tg-muted">{{ formatDateTime(user.createdAt) }}</td>
            <td>
              <div class="flex flex-wrap justify-end gap-2">
                <button
                  type="button"
                  class="tg-btn"
                  :class="user.status === 'active' ? 'tg-btn-amber' : 'tg-btn-primary'"
                  :disabled="isSelf(user) || isBusy(user)"
                  @click="toggleStatus(user)"
                >
                  {{ user.status === 'active' ? '停用' : '启用' }}
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
  lastLoginAt?: string
  createdAt: string
  updatedAt: string
}

export interface AdminUserPatch {
  status: UserStatus
}

const props = withDefaults(defineProps<{
  users: AdminUser[]
  currentUserId?: string
  busyUserId?: string | null
}>(), {
  currentUserId: '',
  busyUserId: null
})

const emit = defineEmits<{ update: [id: string, patch: AdminUserPatch] }>()

const isSelf = (user: AdminUser) => user.id === props.currentUserId
const isBusy = (user: AdminUser) => props.busyUserId === user.id

const formatDateTime = (value?: string) => value ? value.slice(0, 19).replace('T', ' ') : '未登录'
const statusText = (status: UserStatus) => status === 'active' ? '启用中' : '已停用'
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
