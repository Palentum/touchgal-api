<template>
  <div class="rounded-3xl border border-emerald-300/20 bg-emerald-300/10 p-6">
    <h3 class="text-xl font-black">创建 API Token</h3>
    <p class="mt-2 text-sm text-slate-300">账号申请通过后，可无限创建 API token。明文 token 只显示一次。</p>
    <div v-if="!hasApprovedApplication" class="mt-5 rounded-2xl border border-amber-300/30 bg-amber-300/10 p-4 text-sm text-amber-100">
      <p>{{ hasApplication ? '账号申请尚未通过，暂不能创建 token。' : '请先提交账号级 API 申请。' }}</p>
      <NuxtLink to="/apply" class="mt-3 inline-flex rounded-xl bg-amber-300 px-4 py-2 font-bold text-slate-950">{{ hasApplication ? '查看申请状态' : '提交申请' }}</NuxtLink>
    </div>
    <form v-else class="mt-5 grid gap-3" @submit.prevent="create">
      <input v-model="name" class="rounded-xl border border-white/10 bg-slate-950 px-4 py-3 text-white" placeholder="Production Token" required>
      <button class="rounded-xl bg-emerald-400 px-4 py-3 font-bold text-slate-950">生成 token</button>
    </form>
    <div v-if="plainToken" class="mt-5 rounded-2xl border border-amber-300/30 bg-amber-300/10 p-4">
      <p class="font-bold text-amber-100">请立即复制，之后无法再次查看。</p>
      <code class="mt-3 block break-all rounded-xl bg-slate-950 p-3 text-sm text-emerald-200">{{ plainToken }}</code>
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
