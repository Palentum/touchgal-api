<template>
  <div class="tg-page">
    <header class="tg-nav-shell">
      <nav class="tg-container tg-nav" aria-label="文档导航">
        <NuxtLink to="/" class="tg-brand">
          <img class="tg-logo" src="/logo.webp" width="32" height="32" alt="" aria-hidden="true">
          <span>TouchGal API</span>
        </NuxtLink>
        <div class="tg-nav-actions">
          <NuxtLink to="/docs/api" class="tg-link">API 文档</NuxtLink>
          <NuxtLink to="/dashboard" class="tg-btn tg-btn-secondary">控制台</NuxtLink>
        </div>
      </nav>
    </header>
    <main class="tg-container-narrow tg-section-tight">
      <slot />
    </main>
  </div>
</template>
