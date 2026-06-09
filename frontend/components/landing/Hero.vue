<template>
  <section class="tg-section">
    <div class="tg-container tg-hero-grid">
      <div>
        <p class="tg-eyebrow">TouchGal Metadata API</p>
        <h1 class="tg-display-xl">给 Galgame 产品和研究工作流的干净元数据入口</h1>
        <p class="tg-lead">
          TouchGal API 提供独立、脱敏、可限流的公开条目元数据。使用邮箱验证码登录，提交申请后即可在控制台管理 token、调试请求并观察调用量。
        </p>
        <div class="tg-actions" style="margin-top: 32px;">
          <NuxtLink to="/auth/register" class="tg-btn tg-btn-primary">立即申请 API</NuxtLink>
          <NuxtLink to="/dashboard" class="tg-btn tg-btn-secondary">登录控制台</NuxtLink>
        </div>
      </div>

      <aside class="tg-hero-card" aria-label="API 响应示例">
        <div class="tg-code-window">
          <div class="tg-code-window-bar">
            <span class="tg-window-dots" aria-hidden="true"><span /><span /><span /></span>
            <span>GET /v1/games/search</span>
          </div>
          <pre><code><span class="tg-code-muted">curl</span> <span class="tg-code-teal">"https://api.touchgal.dev/v1/games/search?keyword=summer"</span> \
  -H <span class="tg-code-amber">"Authorization: Bearer tgal_live_xxx"</span>

<span class="tg-code-muted">HTTP/1.1 200 OK</span>
{
  <span class="tg-code-accent">"success"</span>: true,
  <span class="tg-code-accent">"data"</span>: {
    <span class="tg-code-accent">"items"</span>: [
      {
        <span class="tg-code-accent">"uniqueId"</span>: <span class="tg-code-teal">"abcd1234"</span>,
        <span class="tg-code-accent">"name"</span>: <span class="tg-code-teal">"Summer Pockets"</span>,
        <span class="tg-code-accent">"contentLimit"</span>: <span class="tg-code-teal">"sfw"</span>
      }
    ]
  }
}</code></pre>
        </div>
        <div class="tg-grid-3" style="margin-top: 20px;">
          <div>
            <span class="tg-badge tg-badge-success">SFW 默认</span>
            <p class="tg-muted" style="margin-top: 10px;">公开 clean DB 边界</p>
          </div>
          <div>
            <span class="tg-badge tg-badge-warning">Rate Limit</span>
            <p class="tg-muted" style="margin-top: 10px;">按 token 独立限流</p>
          </div>
          <div>
            <span class="tg-badge tg-badge-coral">v1</span>
            <p class="tg-muted" style="margin-top: 10px;">稳定版本前缀</p>
          </div>
        </div>
      </aside>
    </div>
  </section>
</template>
