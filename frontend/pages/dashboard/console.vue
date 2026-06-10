<template>
  <section class="tg-dashboard-stack">
    <div>
      <p class="tg-eyebrow">API Console</p>
      <h1 class="tg-display-md">API 调试台</h1>
      <p class="tg-lead">使用一次性输入的明文 token 验证 Public API 响应；页面不会保存 token。</p>
    </div>

    <DashboardApiConsole />
  </section>
</template>

<script setup lang="ts">

definePageMeta({ layout: 'dashboard', middleware: 'auth' })
</script>
