<template>
  <section class="tg-dashboard-stack">
    <div class="tg-card-dark flex flex-wrap items-end justify-between gap-6">
      <div>
        <p class="tg-eyebrow">User Management</p>
        <h1 class="tg-display-md">用户管理</h1>
        <p class="tg-muted mt-3">检索开发者账号，编辑资料、限流、状态或删除用户。</p>
      </div>

      <form class="flex w-full flex-wrap items-end gap-3 sm:w-auto" @submit.prevent="applyFilters">
        <label class="tg-label tg-label-on-dark w-full min-w-0 sm:w-auto sm:min-w-[240px]">
          搜索用户
          <input v-model="search" class="tg-input" type="search" name="q" placeholder="邮箱或昵称">
        </label>
        <label class="tg-label tg-label-on-dark w-full min-w-0 sm:w-auto sm:min-w-[160px]">
          状态
          <select v-model="status" class="tg-select" name="status">
            <option value="all">全部</option>
            <option value="active">启用中</option>
            <option value="disabled">已停用</option>
          </select>
        </label>
        <button class="tg-btn tg-btn-cream w-full sm:w-auto" type="submit" :disabled="loading">筛选</button>
      </form>
    </div>

    <p v-if="error" class="tg-message-error">{{ error }}</p>

    <AdminUserTable
      :users="users"
      :current-user-id="auth.user?.id"
      :busy-user-id="busyUserId"
      @update="updateUser"
      @edit="openEditUser"
      @delete="openDeleteUser"
    />

    <div
      v-if="editingUser"
      class="tg-dialog-backdrop fixed inset-0 z-50 grid place-items-center bg-black/55 px-4"
      role="dialog"
      aria-modal="true"
      aria-labelledby="edit-user-title"
      @click.self="closeEditUser"
    >
      <form class="tg-dialog-panel tg-card w-full max-w-xl" @submit.prevent="saveUser">
        <p class="tg-eyebrow">编辑用户</p>
        <h2 id="edit-user-title" class="tg-title-lg">修改 {{ editingUser.displayName || editingUser.email }}</h2>
        <div class="mt-6 grid gap-4 md:grid-cols-2">
          <label class="tg-label" for="admin-user-display-name">
            昵称
            <input id="admin-user-display-name" v-model="editForm.displayName" class="tg-input mt-2" maxlength="80" autocomplete="name" autofocus>
          </label>
          <label class="tg-label" for="admin-user-email">
            邮箱
            <input id="admin-user-email" v-model="editForm.email" class="tg-input mt-2" type="email" maxlength="254" autocomplete="email" required>
          </label>
          <label class="tg-label" for="admin-user-minute-limit">
            每分钟限流
            <input id="admin-user-minute-limit" v-model="editForm.minuteLimit" class="tg-input mt-2" type="text" required>
          </label>
          <label class="tg-label" for="admin-user-daily-limit">
            每日限流
            <input id="admin-user-daily-limit" v-model="editForm.dailyLimit" class="tg-input mt-2" type="text" required>
          </label>
        </div>
        <p class="tg-muted mt-4">用户限流会作为该用户所有 API Token 的全局上限。</p>
        <p v-if="editError" class="tg-message-error mt-4">{{ editError }}</p>
        <div class="tg-actions mt-6 justify-end">
          <button type="button" class="tg-btn tg-btn-secondary" :disabled="savingUser" @click="closeEditUser">取消</button>
          <button type="submit" class="tg-btn tg-btn-primary" :disabled="savingUser">{{ savingUser ? '保存中...' : '保存' }}</button>
        </div>
      </form>
    </div>

    <div
      v-if="deletingUser"
      class="tg-dialog-backdrop fixed inset-0 z-50 grid place-items-center bg-black/55 px-4"
      role="dialog"
      aria-modal="true"
      aria-labelledby="delete-user-title"
      @click.self="closeDeleteUser"
    >
      <div class="tg-dialog-panel tg-card w-full max-w-md">
        <p class="tg-eyebrow">删除用户</p>
        <h2 id="delete-user-title" class="tg-title-lg">确认删除这个用户？</h2>
        <p class="tg-muted mt-4">
          删除后 <span class="font-semibold text-[var(--tg-body-strong)]">{{ deletingUser.displayName || deletingUser.email }}</span>
          的登录会话、申请和 API Token 会一并失效。
        </p>
        <p v-if="deleteError" class="tg-message-error mt-4">{{ deleteError }}</p>
        <div class="tg-actions mt-6 justify-end">
          <button type="button" class="tg-btn tg-btn-secondary" :disabled="deleting" @click="closeDeleteUser">取消</button>
          <button type="button" class="tg-btn tg-btn-danger" :disabled="deleting" @click="confirmDeleteUser">{{ deleting ? '删除中...' : '确认删除' }}</button>
        </div>
      </div>
    </div>

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

const { apiFetch, apiData } = useApi()
const auth = useAuthStore()
const users = ref<AdminUser[]>([])
const status = ref<UserStatus | 'all'>('all')
const search = ref('')
const page = ref(1)
const limit = 50
const loading = ref(false)
const busyUserId = ref<string | null>(null)
const error = ref('')
const editingUser = ref<AdminUser | null>(null)
const editForm = reactive({
  displayName: '',
  email: '',
  minuteLimit: '',
  dailyLimit: ''
})
const editError = ref('')
const savingUser = ref(false)
const deletingUser = ref<AdminUser | null>(null)
const deleteError = ref('')
const deleting = ref(false)

const query = computed(() => ({
  page: page.value,
  limit,
  status: status.value === 'all' ? undefined : status.value,
  q: search.value.trim() || undefined
}))
const { data: usersResponse, refresh: refreshUsers } = await apiData<AdminUser[]>('admin:users', '/admin/users', { query, dedupe: 'cancel' })

const syncUsers = () => {
  const res = usersResponse.value
  if (!res) {
    return
  }
  if (res.success) {
    users.value = res.data
    error.value = ''
  } else {
    error.value = res.error.message
  }
}

watch(usersResponse, syncUsers, { immediate: true })

const load = async () => {
  loading.value = true
  error.value = ''
  try {
    await refreshUsers()
    syncUsers()
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

const syncAuthUser = (user: AdminUser) => {
  if (auth.user?.id !== user.id) return
  auth.user = {
    ...auth.user,
    email: user.email,
    displayName: user.displayName,
    status: user.status,
    isAdmin: user.isAdmin
  }
}

const updateUser = async (id: string, patch: AdminUserPatch) => {
  busyUserId.value = id
  error.value = ''
  try {
    const res = await apiFetch<AdminUser>(`/admin/users/${id}`, { method: 'PATCH', body: patch })
    if (res.success) {
      users.value = users.value.map(user => user.id === id ? res.data : user)
      syncAuthUser(res.data)
      return { success: true as const }
    }
    error.value = res.error.message
    return { success: false as const, message: res.error.message }
  } catch {
    const message = '更新用户失败'
    error.value = message
    return { success: false as const, message }
  } finally {
    busyUserId.value = null
  }
}

const openEditUser = (user: AdminUser) => {
  editingUser.value = user
  editForm.displayName = user.displayName
  editForm.email = user.email
  editForm.minuteLimit = String(user.minuteLimit)
  editForm.dailyLimit = String(user.dailyLimit)
  editError.value = ''
}

const closeEditUser = () => {
  if (savingUser.value) return
  editingUser.value = null
  editError.value = ''
}

const saveUser = async () => {
  if (!editingUser.value) return
  const email = editForm.email.trim()
  const displayName = editForm.displayName.trim()
  const minuteLimit = Number(editForm.minuteLimit)
  const dailyLimit = Number(editForm.dailyLimit)
  if (!email) {
    editError.value = '请输入邮箱'
    return
  }
  if (!Number.isInteger(minuteLimit) || !Number.isInteger(dailyLimit) || minuteLimit <= 0 || dailyLimit <= 0) {
    editError.value = '限流必须是正整数'
    return
  }
  if (dailyLimit < minuteLimit) {
    editError.value = '每日限流不能小于每分钟限流'
    return
  }
  savingUser.value = true
  editError.value = ''
  const result = await updateUser(editingUser.value.id, { email, displayName, minuteLimit, dailyLimit })
  savingUser.value = false
  if (!result.success) {
    editError.value = result.message
    return
  }
  closeEditUser()
}

const openDeleteUser = (user: AdminUser) => {
  if (auth.user?.id === user.id) return
  deletingUser.value = user
  deleteError.value = ''
}

const closeDeleteUser = () => {
  if (deleting.value) return
  deletingUser.value = null
  deleteError.value = ''
}

const confirmDeleteUser = async () => {
  if (!deletingUser.value) return
  const id = deletingUser.value.id
  deleting.value = true
  busyUserId.value = id
  deleteError.value = ''
  error.value = ''
  try {
    const res = await apiFetch<{ deleted: boolean }>(`/admin/users/${id}`, { method: 'DELETE' })
    if (!res.success) {
      deleteError.value = res.error.message
      error.value = res.error.message
      return
    }
    users.value = users.value.filter(user => user.id !== id)
    deletingUser.value = null
    deleteError.value = ''
  } catch {
    const message = '删除用户失败'
    deleteError.value = message
    error.value = message
  } finally {
    deleting.value = false
    busyUserId.value = null
  }
}

</script>
