<template>
  <section class="tg-section">
    <div class="tg-container tg-hero-grid">
      <div>
        <p class="tg-eyebrow">TouchGal Metadata API</p>
        <h1 class="tg-display-xl">TouchGal 官方 GalGame 条目信息 API</h1>
        <p class="tg-lead">
          TouchGal API 提供搜索、信息获取等服务。
        </p>
        <div class="tg-actions" style="margin-top: 32px;">
          <NuxtLink to="/auth/register" class="tg-btn tg-btn-primary">立即申请 API</NuxtLink>
          <NuxtLink to="/dashboard" class="tg-btn tg-btn-secondary">登录控制台</NuxtLink>
        </div>
      </div>

      <aside class="tg-hero-card" aria-label="API 响应示例">
        <div class="tg-code-window tg-hero-code-window">
          <div class="tg-code-window-bar">
            <span class="tg-window-dots" aria-hidden="true"><span /><span /><span /></span>
            <span>GET /v1/games/search</span>
          </div>
          <pre><code><span class="tg-code-muted">curl</span> <span class="tg-code-teal">"https://developer.touchgal.com/api/v1/games/search?keyword=summer"</span> \
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
      </aside>
    </div>
  </section>
</template>
