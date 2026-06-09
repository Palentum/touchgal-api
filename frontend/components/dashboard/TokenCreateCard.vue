<template>
  <div class="tg-card-dark">
    <p class="tg-eyebrow">创建 Token</p>
    <h2 class="tg-title-lg">创建 API Token</h2>
    <p class="tg-muted mt-3">账号申请通过后可创建 API token。明文 token 只显示一次。</p>

    <div v-if="!hasApprovedApplication" class="tg-card-outline mt-6">
      <p class="tg-title-sm">{{ hasApplication ? '账号申请尚未通过，暂不能创建 token。' : '请先提交账号级 API 申请。' }}</p>
      <NuxtLink to="/dashboard/apply" class="tg-btn tg-btn-amber mt-4">{{ hasApplication ? '查看申请状态' : '提交申请' }}</NuxtLink>
    </div>

    <form v-else class="tg-card-outline tg-form mt-6" @submit.prevent="create">
      <label class="tg-label">
        Token 名称
        <input v-model="name" class="tg-input" placeholder="Production Token" required>
      </label>
      <button class="tg-btn tg-btn-primary justify-self-start">生成 token</button>
    </form>

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
const hasApplication = computed(() => props.applications.length > 0)
const hasApprovedApplication = computed(() => props.applications.some((app) => app.status === 'approved'))

const create = async () => {
  if (!hasApprovedApplication.value) return
  const res = await apiFetch<{ token: string }>('/tokens', { method: 'POST', body: { name: name.value } })
  if (res.success) {
    plainToken.value = res.data.token
    name.value = ''
    emit('created')
  }
}
</script>
