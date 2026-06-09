<template>
  <section class="tg-dashboard-stack">
    <div>
      <p class="tg-eyebrow">API Tokens</p>
      <h1 class="tg-display-md">Token 管理</h1>
      <p class="tg-lead">创建、复制、重命名和删除 token。</p>
    </div>

    <DashboardTokenCreateCard :applications="apps" @created="load" />
    <DashboardTokenTable :tokens="tokens" :can-edit="true" @edit="openEditToken" @delete="deleteToken" />
    <div v-if="editingToken" class="fixed inset-0 z-50 grid place-items-center bg-black/55 px-4" role="dialog" aria-modal="true" aria-labelledby="edit-token-title">
      <form class="tg-card w-full max-w-md" @submit.prevent="saveTokenName">
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
  </section>
</template>

<script setup lang="ts">
import type { ApplicationItem, TokenItem } from '~/composables/useDashboard'

definePageMeta({ layout: 'dashboard', middleware: 'auth' })
const dash = useDashboard()
const { apiFetch } = useApi()
const apps = ref<ApplicationItem[]>([])
const tokens = ref<TokenItem[]>([])
const editingToken = ref<TokenItem | null>(null)
const editName = ref('')
const editError = ref('')
const savingName = ref(false)
const load = async () => {
  const [appRes, tokenRes] = await Promise.all([dash.applications(), dash.tokens()])
  if (appRes.success) apps.value = appRes.data
  if (tokenRes.success) tokens.value = tokenRes.data
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

const deleteToken = async (id: string) => {
  await apiFetch(`/tokens/${id}`, { method: 'DELETE' })
  await load()
}

onMounted(load)
</script>
