<template>
  <div class="tg-card-dark">
    <p class="tg-eyebrow">创建 Token</p>
    <h2 class="tg-title-lg">创建 API Token</h2>
    <p class="tg-muted mt-3">账号申请通过后可创建 API token。明文 token 只显示一次。</p>

    <div v-if="!hasApprovedApplication" class="tg-card-outline mt-6">
      <p class="tg-title-sm">{{ hasApplication ? '账号申请尚未通过，暂不能创建 token。' : '请先提交账号级 API 申请。' }}</p>
      <NuxtLink to="/dashboard/apply" class="tg-btn tg-btn-amber mt-4">{{ hasApplication ? '查看申请状态' : '提交申请' }}</NuxtLink>
    </div>

    <div v-else class="mt-6">
      <button type="button" class="tg-btn tg-btn-primary" @click="openCreateDialog">生成 token</button>
    </div>

    <div v-if="showCreateDialog" class="fixed inset-0 z-50 grid place-items-center bg-black/55 px-4" role="dialog" aria-modal="true" aria-labelledby="create-token-title">
      <form class="tg-card w-full max-w-md" @submit.prevent="create">
        <p class="tg-eyebrow">创建 Token</p>
        <h2 id="create-token-title" class="tg-title-lg">输入 API Token 名称</h2>
        <label class="tg-label mt-6" for="new-token-name">API Token 名称</label>
        <input id="new-token-name" v-model="name" class="tg-input mt-2" maxlength="100" autocomplete="off" autofocus>
        <p v-if="createError" class="tg-message-error mt-4">{{ createError }}</p>
        <div class="tg-actions mt-6 justify-end">
          <button type="button" class="tg-btn tg-btn-secondary" :disabled="creating" @click="closeCreateDialog">取消</button>
          <button type="submit" class="tg-btn tg-btn-primary" :disabled="creating || name.trim().length === 0">生成</button>
        </div>
      </form>
    </div>

    <div v-if="plainToken" class="tg-code-window mt-6">
      <div class="tg-code-window-bar">
        <span>一次性明文 token</span>
        <span class="tg-window-dots"><span /><span /><span /></span>
      </div>
      <div class="p-4">
        <p class="tg-code-amber font-semibold">请立即复制，之后无法再次查看。</p>
        <pre class="mt-3"><code>{{ plainToken }}</code></pre>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { ApplicationItem } from '~/composables/useDashboard'

const props = defineProps<{ applications: ApplicationItem[] }>()
const emit = defineEmits<{ created: [] }>()
const { apiFetch } = useApi()
const name = ref('')
const plainToken = ref('')
const showCreateDialog = ref(false)
const createError = ref('')
const creating = ref(false)
const hasApplication = computed(() => props.applications.length > 0)
const hasApprovedApplication = computed(() => props.applications.some((app) => app.status === 'approved'))

const openCreateDialog = () => {
  if (!hasApprovedApplication.value) return
  name.value = ''
  createError.value = ''
  showCreateDialog.value = true
}

const closeCreateDialog = () => {
  showCreateDialog.value = false
  name.value = ''
  createError.value = ''
}

const create = async () => {
  if (!hasApprovedApplication.value || creating.value) return
  const tokenName = name.value.trim()
  if (!tokenName) {
    createError.value = '请输入 API Token 名称'
    return
  }
  creating.value = true
  createError.value = ''
  const res = await apiFetch<{ token: string }>('/tokens', { method: 'POST', body: { name: tokenName } })
  creating.value = false
  if (res.success) {
    plainToken.value = res.data.token
    closeCreateDialog()
    emit('created')
    return
  }
  createError.value = res.error.message
}
</script>
