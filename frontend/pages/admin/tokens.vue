<template>
  <section class="tg-dashboard-stack">
    <div>
      <p class="tg-eyebrow">API Token</p>
      <h1 class="tg-display-md">全部 Token</h1>
      <p class="tg-lead">查看已签发的 API Token，并在需要时直接删除。</p>
    </div>
    <DashboardTokenTable :tokens="tokens" @delete="openDeleteToken" />
    <div
      v-if="deletingToken"
      class="tg-dialog-backdrop fixed inset-0 z-50 grid place-items-center bg-black/55 px-4"
      role="dialog"
      aria-modal="true"
      aria-labelledby="delete-admin-token-title"
      @click.self="closeDeleteToken"
    >
      <div class="tg-dialog-panel tg-card w-full max-w-md">
        <p class="tg-eyebrow">删除 Token</p>
        <h2 id="delete-admin-token-title" class="tg-title-lg">确认删除这个 token？</h2>
        <p class="tg-muted mt-4">
          删除后 <span class="font-semibold text-[var(--tg-body-strong)]">{{ deletingToken.name }}</span>
          将立即失效，使用该 token 的 API 请求会被拒绝。
        </p>
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
definePageMeta({ layout: 'admin', middleware: 'admin' })
const { apiFetch, apiData } = useApi()
const { data: tokensResponse, refresh: refreshTokens } = await apiData<TokenItem[]>('admin:tokens', '/admin/tokens', { query: { page: 1, limit: 50 } })
const tokens = computed(() => tokensResponse.value?.success ? tokensResponse.value.data : [])
const deletingToken = ref<TokenItem | null>(null)
const deleteError = ref('')
const deleting = ref(false)
const load = async () => { await refreshTokens() }

const openDeleteToken = (id: string) => {
  const token = tokens.value.find(item => item.id === id)
  if (!token) return
  deletingToken.value = token
  deleteError.value = ''
}

const closeDeleteToken = () => {
  if (deleting.value) return
  deletingToken.value = null
  deleteError.value = ''
}

const confirmDeleteToken = async () => {
  if (!deletingToken.value) return
  const id = deletingToken.value.id
  deleting.value = true
  deleteError.value = ''
  try {
    const res = await apiFetch<{ deleted: boolean }>(`/admin/tokens/${id}`, { method: 'DELETE' })
    if (!res.success) {
      deleteError.value = res.error.message
      return
    }
    deletingToken.value = null
    deleteError.value = ''
    await load()
  } catch {
    deleteError.value = '删除 Token 失败'
  } finally {
    deleting.value = false
  }
}
</script>
