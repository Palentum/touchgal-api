<template>
  <div class="rounded-3xl border border-white/10 bg-white/[0.06] p-6">
    <h2 class="text-2xl font-black">API 调试台</h2>
    <div class="mt-5 grid gap-4 md:grid-cols-2">
      <label class="grid gap-2 text-sm">Token 记录<select v-model="selectedToken" class="rounded-xl border border-white/10 bg-slate-950 px-4 py-3 text-white"><option value="">选择 token 记录</option><option v-for="token in tokens" :key="token.id" :value="token.tokenPrefix">{{ token.name }} · {{ token.tokenPrefix }}</option></select></label>
      <label class="grid gap-2 text-sm">接口<select v-model="endpoint" class="rounded-xl border border-white/10 bg-slate-950 px-4 py-3 text-white"><option value="search">搜索</option><option value="detail">详情</option><option value="me">Token 自检</option></select></label>
      <label class="grid gap-2 text-sm md:col-span-2">API token 明文（仅本次调试，不保存）<input v-model="apiToken" class="rounded-xl border border-white/10 bg-slate-950 px-4 py-3 text-white" placeholder="tgal_live_xxx"></label>
      <label v-if="endpoint === 'search'" class="grid gap-2 text-sm">keyword<input v-model="keyword" class="rounded-xl border border-white/10 bg-slate-950 px-4 py-3 text-white"></label>
      <label v-if="endpoint === 'detail'" class="grid gap-2 text-sm">uniqueId<input v-model="uniqueId" maxlength="8" class="rounded-xl border border-white/10 bg-slate-950 px-4 py-3 text-white"></label>
    </div>
    <pre class="mt-5 overflow-x-auto rounded-2xl bg-slate-950 p-4 text-sm text-emerald-200">{{ curl }}</pre>
    <button class="mt-4 rounded-xl bg-emerald-400 px-5 py-3 font-bold text-slate-950" @click="send">发送请求</button>
    <div v-if="status" class="mt-5 text-sm text-slate-300">HTTP {{ status }} · {{ elapsed }}ms</div>
    <pre class="mt-3 min-h-40 overflow-x-auto rounded-2xl bg-slate-950 p-4 text-sm text-slate-200">{{ response }}</pre>
  </div>
</template>
<script setup lang="ts">
import type { TokenItem } from '~/composables/useDashboard'
const props = defineProps<{ tokens: TokenItem[] }>()
const { baseURL } = useApi()
const selectedToken = ref('')
const apiToken = ref('')
const endpoint = ref<'search' | 'detail' | 'me'>('search')
const keyword = ref('summer')
const uniqueId = ref('abcd1234')
const response = ref('')
const status = ref(0)
const elapsed = ref(0)
const tokenValue = computed(() => apiToken.value || selectedToken.value || 'tgal_live_xxx')
const path = computed(() => endpoint.value === 'search' ? `/v1/games/search?keyword=${encodeURIComponent(keyword.value)}&page=1&limit=10` : endpoint.value === 'detail' ? `/v1/games/${uniqueId.value}` : '/v1/me')
const curl = computed(() => `curl "${baseURL}${path.value}" \\\n  -H "Authorization: Bearer ${tokenValue.value}"`)
const send = async () => {
  const started = performance.now()
  try {
    const res = await fetch(`${baseURL}${path.value}`, { headers: { Authorization: `Bearer ${tokenValue.value}` } })
    status.value = res.status
    response.value = JSON.stringify(await res.json(), null, 2)
  } catch (err) {
    status.value = 0
    response.value = String(err)
  } finally {
    elapsed.value = Math.round(performance.now() - started)
  }
}
</script>
