<template>
  <div class="tg-card-dark">
    <div class="tg-card-outline tg-form-grid">
      <label class="tg-label">
        Token 记录
        <select v-model="selectedToken" class="tg-select">
          <option value="">选择 token 记录</option>
          <option v-for="token in tokens" :key="token.id" :value="token.tokenPrefix">{{ token.name }} · {{ token.tokenPrefix }}</option>
        </select>
      </label>

      <label class="tg-label">
        接口
        <select v-model="endpoint" class="tg-select">
          <option value="search">搜索</option>
          <option value="detail">详情</option>
          <option value="me">Token 自检</option>
        </select>
      </label>

      <label class="tg-label md:col-span-2">
        API token 明文（仅本次调试，不保存）
        <input v-model="apiToken" class="tg-input" placeholder="tgal_live_xxx">
      </label>

      <label v-if="endpoint === 'search'" class="tg-label">
        keyword
        <input v-model="keyword" class="tg-input">
      </label>

      <label v-if="endpoint === 'detail'" class="tg-label">
        uniqueId
        <input v-model="uniqueId" maxlength="8" class="tg-input">
      </label>
    </div>

    <div class="tg-code-window mt-6">
      <div class="tg-code-window-bar">
        <span>curl preview</span>
        <span class="tg-window-dots"><span /><span /><span /></span>
      </div>
      <pre><code>{{ curl }}</code></pre>
    </div>

    <button class="tg-btn tg-btn-primary mt-5" @click="send">发送请求</button>

    <div v-if="status" class="tg-badge tg-badge-success mt-5">HTTP {{ status }} · {{ elapsed }}ms</div>

    <div class="tg-code-window mt-4">
      <div class="tg-code-window-bar">
        <span>response</span>
        <span class="tg-window-dots"><span /><span /><span /></span>
      </div>
      <pre class="min-h-40"><code>{{ response }}</code></pre>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { TokenItem } from '~/composables/useDashboard'

defineProps<{ tokens: TokenItem[] }>()
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
