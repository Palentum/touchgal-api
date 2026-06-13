<template>
  <NuxtPage v-if="!isApiRoot" />
  <section v-else class="tg-dashboard-stack">
    <header class="tg-card-coral">
      <p class="tg-eyebrow" style="color: var(--tg-on-primary);">TouchGal API Docs</p>
      <h1 class="tg-display-lg">TouchGal API 文档</h1>
      <p style="margin-top: 20px; max-width: 760px; line-height: 1.7;">
        稳定、脱敏、可限流的 Galgame 元数据 API。公开业务接口需要 API token；健康检查与就绪检查无需 token。
      </p>
    </header>

    <section class="tg-card-outline">
      <h2 class="tg-title-lg">快速开始</h2>
      <div class="tg-doc-steps">
        <div v-for="step in quickStartSteps" :key="step.title" class="tg-doc-step">
          <span>{{ step.index }}</span>
          <div>
            <h3 class="tg-title-sm">{{ step.title }}</h3>
            <p class="tg-muted">{{ step.body }}</p>
          </div>
        </div>
      </div>
    </section>

    <section class="tg-card-dark">
      <div class="tg-doc-overview-head">
        <div>
          <p class="tg-eyebrow">Endpoints</p>
          <h2 class="tg-title-lg">接口目录</h2>
        </div>
        <span class="tg-badge tg-badge-coral">{{ apiEndpointDocs.length }} endpoints</span>
      </div>
      <div class="tg-doc-endpoint-grid">
        <NuxtLink v-for="doc in apiEndpointDocs" :key="doc.slug" :to="`/docs/api/${doc.slug}`" class="tg-doc-endpoint-card">
          <span class="tg-badge" :class="doc.auth.includes('无需') ? 'tg-badge-success' : 'tg-badge-warning'">{{ doc.method }}</span>
          <h3 class="tg-title-md">{{ doc.navLabel }}</h3>
          <p>{{ doc.path }}</p>
          <small>{{ doc.introduction }}</small>
        </NuxtLink>
      </div>
    </section>

    <section class="tg-grid-2">
      <div class="tg-card-outline">
        <h2 class="tg-title-lg">鉴权</h2>
        <p class="tg-muted tg-doc-text">
          `/v1/games/*` 与 `/v1/me` 支持 `Authorization: Bearer &lt;api_token&gt;` 或 `X-API-Token`。API token 由开发者门户生成，明文只展示一次；前端页面不要把 token 存入 localStorage 或暴露给不可信浏览器环境。
        </p>
      </div>
      <div class="tg-card-outline">
        <h2 class="tg-title-lg">限流</h2>
        <p class="tg-muted tg-doc-text">
          token 认证接口会按 token、账号、应用三维独立计数。响应头包含 `X-RateLimit-Limit-Minute`、`X-RateLimit-Remaining-Minute`、`X-RateLimit-Limit-Day`、`X-RateLimit-Remaining-Day`。
        </p>
      </div>
    </section>

    <section class="tg-card-outline">
      <h2 class="tg-title-lg">响应约定</h2>
      <p class="tg-muted tg-doc-text">所有 JSON 响应都使用统一 envelope。成功响应为 `success: true`；失败响应为 `success: false` 并返回稳定错误码。</p>
      <div class="tg-grid-2 tg-doc-response-grid">
        <div class="tg-code-window">
          <div class="tg-code-window-bar">
            <span class="tg-window-dots" aria-hidden="true"><span /><span /><span /></span>
            <span>success</span>
          </div>
          <pre><code>{
  "success": true,
  "data": {}
}</code></pre>
        </div>
        <div class="tg-code-window">
          <div class="tg-code-window-bar">
            <span class="tg-window-dots" aria-hidden="true"><span /><span /><span /></span>
            <span>error</span>
          </div>
          <pre><code>{
  "success": false,
  "error": {
    "code": "BAD_REQUEST",
    "message": "Invalid request parameters"
  }
}</code></pre>
        </div>
      </div>
    </section>
  </section>
</template>

<script setup lang="ts">
import { apiEndpointDocs } from '~/composables/apiDocs'

definePageMeta({ layout: 'docs' })

const route = useRoute()
const isApiRoot = computed(() => route.path.replace(/\/$/, '') === '/docs/api')

useHead(() => (isApiRoot.value ? { title: 'TouchGal API 文档' } : {}))

const quickStartSteps = [
  { index: '01', title: '注册开发者账号', body: '通过邮箱验证码登录开发者门户；登录态只使用 HttpOnly Cookie。' },
  { index: '02', title: '提交 API 申请', body: '在开发者门户提交账号级应用申请，等待管理员 approved。' },
  { index: '03', title: '创建 API token', body: '审批通过后创建 tgal_live token。明文只在创建响应中返回一次。' },
  { index: '04', title: '调用 /v1 接口', body: '使用 Authorization Bearer 或 X-API-Token 调用业务接口，并处理 400 / 401 / 404 / 429 等状态。' }
]
</script>

<style scoped>
.tg-doc-steps {
  display: grid;
  gap: 16px;
  margin-top: 20px;
}

.tg-doc-step {
  display: grid;
  grid-template-columns: auto minmax(0, 1fr);
  gap: 16px;
  align-items: start;
}

.tg-doc-step > span {
  display: inline-flex;
  min-width: 44px;
  height: 44px;
  align-items: center;
  justify-content: center;
  border-radius: 999px;
  background: var(--tg-primary);
  color: var(--tg-on-primary);
  font-family: var(--tg-font-display);
  font-size: 15px;
}

.tg-doc-step p {
  margin: 6px 0 0;
  line-height: 1.65;
}

.tg-doc-overview-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
}

.tg-doc-endpoint-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 16px;
  margin-top: 22px;
}

.tg-doc-endpoint-card {
  display: grid;
  gap: 10px;
  border: 1px solid rgba(250, 249, 245, 0.14);
  border-radius: 14px;
  background: rgba(250, 249, 245, 0.06);
  color: var(--tg-on-dark);
  padding: 18px;
  transition: transform 160ms ease, border-color 160ms ease, background 160ms ease;
}

.tg-doc-endpoint-card:hover {
  transform: translateY(-2px);
  border-color: rgba(204, 120, 92, 0.58);
  background: rgba(250, 249, 245, 0.1);
}

.tg-doc-endpoint-card .tg-badge {
  justify-self: start;
}

.tg-doc-endpoint-card p,
.tg-doc-endpoint-card small {
  margin: 0;
  overflow-wrap: anywhere;
}

.tg-doc-endpoint-card p {
  color: var(--tg-primary);
  font-family: var(--tg-font-code);
  font-size: 13px;
}

.tg-doc-endpoint-card small {
  color: var(--tg-on-dark-soft);
  line-height: 1.55;
}

.tg-doc-text {
  margin-top: 14px;
  line-height: 1.75;
}

.tg-doc-response-grid {
  margin-top: 20px;
}

@media (max-width: 767px) {
  .tg-doc-endpoint-grid {
    grid-template-columns: 1fr;
  }
}
</style>
