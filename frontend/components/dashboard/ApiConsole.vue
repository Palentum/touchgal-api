<template>
  <div class="tg-card-dark">
    <div class="tg-card-outline tg-form-grid">
      <label class="tg-label md:col-span-2">
        API token
        <input v-model="apiToken" class="tg-input" placeholder="tgal_live_xxx">
      </label>

      <label class="tg-label">
        接口
        <select v-model="endpoint" class="tg-select">
          <option value="search">搜索</option>
          <option value="detail">详情</option>
          <option value="me">Token 自检</option>
        </select>
      </label>

      <label v-if="endpoint === 'search'" class="tg-label">
        keyword
        <input v-model="keyword" class="tg-input">
      </label>

      <label v-else-if="endpoint === 'detail'" class="tg-label">
        uniqueId
        <input v-model="uniqueId" maxlength="8" class="tg-input">
      </label>

      <label v-else class="tg-label">
        参数
        <input class="tg-input" value="无需参数" disabled>
      </label>
    </div>

    <div class="tg-code-window tg-console-code-window mt-6">
      <div class="tg-code-window-bar">
        <span>请求预览</span>
        <span class="tg-window-dots"><span /><span /><span /></span>
      </div>
      <pre><code>{{ curl }}</code></pre>
    </div>

    <div class="tg-console-action-row mt-5">
      <button class="tg-btn tg-btn-primary" :disabled="!authToken" @click="send">发送请求</button>
      <div v-if="status" class="tg-badge tg-badge-success tg-console-status">HTTP {{ status }} · {{ elapsed }}ms</div>
    </div>

    <div class="tg-code-window tg-console-code-window mt-4">
      <div class="tg-code-window-bar">
        <span>接口响应</span>
        <span class="tg-window-dots"><span /><span /><span /></span>
      </div>
      <pre class="min-h-40"><code>{{ response }}</code></pre>
    </div>
  </div>
</template>
<script setup lang="ts">

const { baseURL } = useApi()
const apiToken = ref('')
const endpoint = ref<'search' | 'detail' | 'me'>('search')
const keyword = ref('summer')
const uniqueId = ref('abcd1234')
const response = ref('')
const status = ref(0)
const elapsed = ref(0)
const authToken = computed(() => apiToken.value.trim())
const tokenValue = computed(() => authToken.value || 'tgal_live_xxx')
const path = computed(() => endpoint.value === 'search' ? `/v1/games/search?keyword=${encodeURIComponent(keyword.value)}&page=1&limit=10` : endpoint.value === 'detail' ? `/v1/games/${uniqueId.value}` : '/v1/me')
const curl = computed(() => `curl "${baseURL}${path.value}" \\\n  -H "Authorization: Bearer ${tokenValue.value}"`)

const send = async () => {
  if (!authToken.value) {
    status.value = 0
    elapsed.value = 0
    response.value = '请先输入完整 API token 明文。'
    return
  }
  const started = performance.now()
  try {
    const res = await fetch(`${baseURL}${path.value}`, { headers: { Authorization: `Bearer ${authToken.value}` } })
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

<style scoped>
.tg-console-action-row {
  display: flex;
  align-items: center;
  gap: 16px;
}

.tg-console-status {
  display: flex;
  width: fit-content;
  margin-left: auto;
}

.tg-console-code-window {
  max-width: 100%;
  min-width: 0;
}

.tg-console-code-window pre {
  max-width: 100%;
  overflow-x: hidden;
  white-space: pre-wrap;
  overflow-wrap: anywhere;
  word-break: break-word;
}

.tg-console-code-window code {
  white-space: inherit;
}
</style>
