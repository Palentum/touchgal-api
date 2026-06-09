<template>
  <section class="tg-dashboard-stack">
    <div class="tg-card-dark flex flex-wrap items-end justify-between gap-6">
      <div>
        <p class="tg-eyebrow">用户管理</p>
        <h1 class="tg-display-md">用户管理</h1>
        <p class="tg-muted mt-3">检索开发者账号，并停用或恢复门户登录。</p>
      </div>

      <form class="flex flex-wrap items-end gap-3" @submit.prevent="applyFilters">
        <label class="tg-label tg-label-on-dark min-w-[240px]">
          搜索用户
          <input v-model="search" class="tg-input" type="search" name="q" placeholder="邮箱或昵称">
        </label>
        <label class="tg-label tg-label-on-dark min-w-[160px]">
          状态
          <select v-model="status" class="tg-select" name="status">
            <option value="all">全部</option>
            <option value="active">启用中</option>
            <option value="disabled">已停用</option>
          </select>
        </label>
        <button class="tg-btn tg-btn-cream" type="submit" :disabled="loading">筛选</button>
      </form>
    </div>

    <p v-if="error" class="tg-message-error">{{ error }}</p>

    <AdminUserTable
      :users="users"
      :current-user-id="auth.user?.id"
      :busy-user-id="busyUserId"
      @update="updateUser"
    />

    <div class="flex flex-wrap items-center justify-between gap-3">
      <p class="tg-muted">第 {{ page }} 页 · 每页 {{ limit }} 个</p>
      <div class="tg-actions">
        <button class="tg-btn tg-btn-secondary" type="button" :disabled="loading || page <= 1" @click="previousPage">上一页</button>
        <button class="tg-btn tg-btn-primary" type="button" :disabled="loading || users.length < limit" @click="nextPage">下一页</button>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import type { AdminUser, AdminUserPatch, UserStatus } from '~/components/admin/UserTable.vue'

definePageMeta({ layout: 'admin', middleware: 'admin' })

const { apiFetch } = useApi()
const auth = useAuthStore()
const users = ref<AdminUser[]>([])
const status = ref<UserStatus | 'all'>('all')
const search = ref('')
const page = ref(1)
const limit = 50
const loading = ref(false)
const busyUserId = ref<string | null>(null)
const error = ref('')

const query = computed(() => ({
  page: page.value,
  limit,
  status: status.value === 'all' ? undefined : status.value,
  q: search.value.trim() || undefined
}))

const load = async () => {
  loading.value = true
  error.value = ''
  try {
    const res = await apiFetch<AdminUser[]>('/admin/users', { query: query.value })
    if (res.success) {
      users.value = res.data
    } else {
      error.value = res.error.message
    }
  } catch {
    error.value = '加载用户失败'
  } finally {
    loading.value = false
  }
}

const applyFilters = async () => {
  page.value = 1
  await load()
}

const previousPage = async () => {
  if (page.value <= 1) return
  page.value -= 1
  await load()
}

const nextPage = async () => {
  if (users.value.length < limit) return
  page.value += 1
  await load()
}

const updateUser = async (id: string, patch: AdminUserPatch) => {
  busyUserId.value = id
  error.value = ''
  try {
    const res = await apiFetch<AdminUser>(`/admin/users/${id}`, { method: 'PATCH', body: patch })
    if (res.success) {
      users.value = users.value.map(user => user.id === id ? res.data : user)
    } else {
      error.value = res.error.message
    }
  } catch {
    error.value = '更新用户失败'
  } finally {
    busyUserId.value = null
  }
}

onMounted(load)
</script>
