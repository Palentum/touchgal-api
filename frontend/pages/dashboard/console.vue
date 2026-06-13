<template>
  <section class="tg-dashboard-stack">
    <div>
      <p class="tg-eyebrow">API Console</p>
      <h1 class="tg-display-md">API 调试台</h1>
      <p class="tg-lead">使用 API Token 调试与验证 API 响应。</p>
    </div>

    <DashboardApiConsole />
  </section>
</template>

<script setup lang="ts">

definePageMeta({ layout: 'dashboard', middleware: 'auth' })
</script>
