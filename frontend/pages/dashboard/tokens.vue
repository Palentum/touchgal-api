<template>
  <section class="tg-dashboard-stack">
    <div>
      <p class="tg-eyebrow">API Tokens Management</p>
      <h1 class="tg-display-md">Token 管理</h1>
      <p class="tg-lead">创建、复制、重命名和删除 token。</p>
    </div>

    <DashboardTokenCreateCard :applications="apps" @created="load" />
    <DashboardTokenTable :tokens="tokens" :can-edit="true" @edit="openEditToken" @delete="openDeleteToken" />
    <div v-if="editingToken" class="tg-dialog-backdrop fixed inset-0 z-50 grid place-items-center bg-black/55 px-4" role="dialog" aria-modal="true" aria-labelledby="edit-token-title">
      <form class="tg-dialog-panel tg-card w-full max-w-md" @submit.prevent="saveTokenName">
        <p class="tg-eyebrow">编辑 Token</p>
        <h2 id="edit-token-title" class="tg-title-lg">修改 token 名称</h2>
        <label class="tg-label mt-6" for="token-name">名称</label>
        <input id="token-name" v-model="editName" class="tg-input mt-2" maxlength="100" autocomplete="off" autofocus>
        <p v-if="editError" class="tg-message-error mt-4">{{ editError }}</p>
        <div class="tg-actions mt-6 justify-end">
          <button type="button" class="tg-btn tg-btn-secondary" :disabled="savingName" @click="closeEditToken">取消</button>
          <button type="submit" class="tg-btn tg-btn-primary" :disabled="savingName || editName.trim().length === 0">保存</button>
        </div>
      </form>
    </div>
    <div v-if="deletingToken" class="tg-dialog-backdrop fixed inset-0 z-50 grid place-items-center bg-black/55 px-4" role="dialog" aria-modal="true" aria-labelledby="delete-token-title">
      <div class="tg-dialog-panel tg-card w-full max-w-md">
        <p class="tg-eyebrow">删除 Token</p>
        <h2 id="delete-token-title" class="tg-title-lg">确认删除这个 token？</h2>
        <p class="tg-muted mt-4">删除后 <span class="font-semibold text-[var(--tg-body-strong)]">{{ deletingToken.name }}</span> 将立即失效，使用该 token 的 API 请求会被拒绝。</p>
        <p v-if="deleteError" class="tg-message-error mt-4">{{ deleteError }}</p>
        <div class="tg-actions mt-6 justify-end">
          <button type="button" class="tg-btn tg-btn-secondary" :disabled="deleting" @click="closeDeleteToken">取消</button>
          <button type="button" class="tg-btn tg-btn-danger" :disabled="deleting" @click="confirmDeleteToken">{{ deleting ? '删除中...' : '确认删除' }}</button>
        </div>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import type { TokenItem } from '~/composables/useDashboard'

definePageMeta({ layout: 'dashboard', middleware: 'auth' })
const { apiFetch, apiData } = useApi()
const access = useApplicationAccess()
const apps = access.applications
const { data: tokensResponse, refresh: refreshTokens } = await apiData<TokenItem[]>('dashboard:tokens', '/tokens')
const tokens = computed(() => tokensResponse.value?.success ? tokensResponse.value.data : [])
const editingToken = ref<TokenItem | null>(null)
const editName = ref('')
const editError = ref('')
const savingName = ref(false)
const deletingToken = ref<TokenItem | null>(null)
const deleteError = ref('')
const deleting = ref(false)
const load = async () => {
  await refreshTokens()
}

const openEditToken = (token: TokenItem) => {
  editingToken.value = token
  editName.value = token.name
  editError.value = ''
}

const closeEditToken = () => {
  editingToken.value = null
  editName.value = ''
  editError.value = ''
}

const saveTokenName = async () => {
  if (!editingToken.value) return
  const name = editName.value.trim()
  if (!name) {
    editError.value = '请输入 token 名称'
    return
  }
  savingName.value = true
  editError.value = ''
  const res = await apiFetch<TokenItem>(`/tokens/${editingToken.value.id}`, { method: 'PATCH', body: { name } })
  savingName.value = false
  if (!res.success) {
    editError.value = res.error.message
    return
  }
  closeEditToken()
  await load()
}

const openDeleteToken = (id: string) => {
  const token = tokens.value.find((item) => item.id === id)
  if (!token) return
  deletingToken.value = token
  deleteError.value = ''
}

const closeDeleteToken = () => {
  deletingToken.value = null
  deleteError.value = ''
}

const confirmDeleteToken = async () => {
  if (!deletingToken.value) return
  deleting.value = true
  deleteError.value = ''
  const id = deletingToken.value.id
  const res = await apiFetch(`/tokens/${id}`, { method: 'DELETE' })
  deleting.value = false
  if (!res.success) {
    deleteError.value = res.error.message
    return
  }
  closeDeleteToken()
  await load()
}

</script>
