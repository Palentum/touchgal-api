<template>
  <article class="tg-doc-article">
    <header class="tg-card-coral">
      <p class="tg-eyebrow" style="color: var(--tg-on-primary);">TouchGal API Docs</p>
      <h1 class="tg-display-lg">TouchGal API 文档</h1>
      <p style="margin-top: 20px; max-width: 680px; line-height: 1.7;">
        稳定、脱敏、可限流的 Galgame 条目 API。业务接口需要 API token，health 除外。
      </p>
    </header>

    <section v-for="section in sections" :key="section.title" class="tg-card-outline">
      <div style="display: flex; flex-wrap: wrap; align-items: center; gap: 12px; justify-content: space-between;">
        <h2 class="tg-title-lg">{{ section.title }}</h2>
        <span v-if="section.code" class="tg-badge tg-badge-coral">Example</span>
      </div>
      <p class="tg-muted" style="margin-top: 14px; white-space: pre-line; line-height: 1.75;">{{ section.body }}</p>

      <div v-if="section.code" class="tg-code-window" style="margin-top: 18px;">
        <div class="tg-code-window-bar">
          <span class="tg-window-dots" aria-hidden="true"><span /><span /><span /></span>
          <span>{{ section.title }}</span>
        </div>
        <pre><code>{{ section.code }}</code></pre>
      </div>
    </section>
  </article>
</template>

<script setup lang="ts">
definePageMeta({ layout: 'docs' })
const sections = [
  { title: '快速开始', body: '1. 注册/登录开发者账号。\n2. 提交一次账号级 API 申请。\n3. 管理员 approved 后，该账户可无限创建 token。\n4. 使用 Authorization: Bearer <token> 调用 /v1。' },
  { title: '鉴权', body: 'Header 支持 Authorization: Bearer <api_token> 或 X-API-Token。登录 session 使用 HttpOnly Cookie，不存入 localStorage。' },
  { title: '限流', body: '每个 token 独立 minute_limit 与 daily_limit。响应头包含 X-RateLimit-Limit-Minute、X-RateLimit-Remaining-Minute、X-RateLimit-Limit-Day、X-RateLimit-Remaining-Day。' },
  { title: '错误码', body: 'BAD_REQUEST / UNAUTHORIZED / FORBIDDEN / CONFLICT / NOT_FOUND / RATE_LIMITED / INTERNAL_ERROR', code: '{\n  "success": false,\n  "error": {\n    "code": "UNAUTHORIZED",\n    "message": "Missing or invalid API token"\n  }\n}' },
  { title: 'GET /v1/health', body: '无需 token。返回服务状态和版本。' },
  { title: 'GET /v1/games/search', body: '参数 keyword 或 q 必填，page 默认 1，limit 默认 20 最大 50。默认仅返回 content_limit=sfw 且 deleted_at is null 的条目。', code: 'curl "https://api.example.com/v1/games/search?keyword=summer&page=1&limit=10" \\\n  -H "Authorization: Bearer tgal_live_xxx"' },
  { title: 'GET /v1/games/{uniqueId}', body: '返回条目详情、别名、标签、会社和评分聚合。不返回 source_patch_id、主站 user_id、评论或资源下载链接。', code: 'curl "https://api.example.com/v1/games/abcd1234" \\\n  -H "Authorization: Bearer tgal_live_xxx"' },
  { title: 'GET /v1/me', body: 'Token 自检，返回 tokenPrefix、applicationId、applicationStatus、minuteLimit、dailyLimit。', code: 'curl "https://api.example.com/v1/me" \\\n  -H "Authorization: Bearer tgal_live_xxx"' },
  { title: 'TypeScript fetch 示例', body: '前端或服务端均可使用 fetch。不要把 token 暴露给不可信浏览器环境。', code: "const res = await fetch('https://api.example.com/v1/games/search?keyword=summer', {\n  headers: { Authorization: 'Bearer tgal_live_xxx' }\n})\nconst json = await res.json()" },
  { title: 'Go net/http 示例', body: '服务端调用示例。', code: 'req, _ := http.NewRequest("GET", "https://api.example.com/v1/games/abcd1234", nil)\nreq.Header.Set("Authorization", "Bearer tgal_live_xxx")\nresp, err := http.DefaultClient.Do(req)' },
  { title: '字段说明与版本策略', body: 'uniqueId 为公开 8 字符 ID；publishTime=source_created_at；releaseDate=released；updatedAt=source_updated_at；resourceUpdateTime=resource_updated_at。当前版本前缀 /v1，破坏性变化会升级到 /v2。' }
]
</script>
