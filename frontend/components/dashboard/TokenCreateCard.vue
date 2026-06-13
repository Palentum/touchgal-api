<template>
  <div class="tg-card-dark">
    <p class="tg-eyebrow">Create Token</p>
    <h2 class="tg-title-lg">创建 API Token</h2>
    <p class="tg-muted mt-3">账号申请通过后可创建有限数量的 API token；管理员默认具备创建权限。明文 token 只显示一次。</p>

    <div v-if="!canCreateToken" class="tg-card-outline mt-6">
      <p class="tg-title-sm">{{ hasApplication ? '账号申请尚未通过，暂不能创建 token。' : '请先提交账号级 API 申请。' }}</p>
      <NuxtLink to="/dashboard/apply" class="tg-btn tg-btn-amber mt-4">{{ hasApplication ? '查看申请状态' : '提交申请' }}</NuxtLink>
    </div>

    <div v-else class="mt-6">
      <button type="button" class="tg-btn tg-btn-primary" @click="openCreateDialog">生成 token</button>
    </div>

    <div v-if="showCreateDialog" class="tg-dialog-backdrop fixed inset-0 z-50 grid place-items-center bg-black/55 px-4" role="dialog" aria-modal="true" aria-labelledby="create-token-title">
      <form class="tg-dialog-panel tg-card w-full max-w-md" @submit.prevent="create">
        <p class="tg-eyebrow">Create Token</p>
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
        <span>API Token</span>
        <span class="tg-window-dots"><span /><span /><span /></span>
      </div>
      <div class="p-4">
        <p class="tg-code-amber font-semibold">请将此 API Token 保存在安全且易于访问的地方。出于安全原因，你将无法通过 API Tokens 管理界面再次查看它。如果你丟失了这个 Token，将需要重新创建。</p>
        <pre class="tg-token-code mt-3"><code>{{ plainToken }}</code><button
          type="button"
          class="tg-icon-btn tg-token-copy-btn"
          :aria-label="copied ? '已复制 API Token' : '复制 API Token'"
          :title="copied ? '已复制' : '复制'"
          @click="copyPlainToken"
        >
          <svg v-if="copied" viewBox="0 0 24 24" aria-hidden="true" class="h-4 w-4" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round">
            <path d="m20 6-11 11-5-5" />
          </svg>
          <svg v-else viewBox="0 0 24 24" aria-hidden="true" class="h-4 w-4" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round">
            <rect x="9" y="9" width="11" height="11" rx="2" />
            <path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1" />
          </svg>
        </button></pre>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { ApplicationItem } from '~/composables/useDashboard'
import { hasApprovedApplicationAccess } from '~/composables/useApplicationAccess'

const props = defineProps<{ applications: ApplicationItem[] }>()
const emit = defineEmits<{ created: [] }>()
const { apiFetch } = useApi()
const auth = useAuthStore()
const name = ref('')
const plainToken = ref('')
const copied = ref(false)
const showCreateDialog = ref(false)
const createError = ref('')
const creating = ref(false)
const hasApplication = computed(() => props.applications.length > 0)
const canCreateToken = computed(() => hasApprovedApplicationAccess(props.applications, auth.user?.isAdmin === true))

const openCreateDialog = () => {
  if (!canCreateToken.value) return
  name.value = ''
  createError.value = ''
  showCreateDialog.value = true
}

const closeCreateDialog = () => {
  showCreateDialog.value = false
  name.value = ''
  createError.value = ''
}

let copyResetTimer: ReturnType<typeof setTimeout> | undefined

const copyPlainToken = async () => {
  if (!plainToken.value) return

  try {
    await navigator.clipboard.writeText(plainToken.value)
  } catch {
    return
  }

  copied.value = true
  if (copyResetTimer) clearTimeout(copyResetTimer)
  copyResetTimer = setTimeout(() => {
    copied.value = false
  }, 1600)
}

onBeforeUnmount(() => {
  if (copyResetTimer) clearTimeout(copyResetTimer)
})

const create = async () => {
  if (!canCreateToken.value || creating.value) return
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
    copied.value = false
    if (copyResetTimer) {
      clearTimeout(copyResetTimer)
      copyResetTimer = undefined
    }
    closeCreateDialog()
    emit('created')
    return
  }
  createError.value = res.error.message
}
</script>

<style scoped>
.tg-token-code {
  position: relative;
  padding-right: 64px;
  white-space: pre-wrap;
  overflow-wrap: anywhere;
}

.tg-token-copy-btn {
  position: absolute;
  top: 12px;
  right: 12px;
  border-color: rgba(250, 249, 245, 0.14);
  background: rgba(250, 249, 245, 0.08);
  color: var(--tg-on-dark);
}

.tg-token-copy-btn:hover {
  background: rgba(250, 249, 245, 0.16);
}
@media (max-width: 420px) {
  .tg-token-code {
    padding-right: 48px;
  }

  .tg-token-copy-btn {
    top: 10px;
    right: 10px;
  }
}

</style>
