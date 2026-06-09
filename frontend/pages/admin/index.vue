<template>
  <section class="tg-dashboard-stack">
    <div>
      <p class="tg-eyebrow">管理后台</p>
      <h1 class="tg-display-md">管理员后台</h1>
      <p class="tg-lead">集中处理开发者申请、API Token 与同步任务状态。</p>
    </div>
    <div class="tg-grid-3">
      <NuxtLink class="tg-card-outline" to="/admin/applications">
        <p class="tg-eyebrow">申请审核</p>
        <h2 class="tg-title-lg">审核申请</h2>
        <p class="tg-muted mt-3">批准、拒绝或撤销开发者项目访问。</p>
      </NuxtLink>
      <NuxtLink class="tg-card-outline" to="/admin/tokens">
        <p class="tg-eyebrow">API Token</p>
        <h2 class="tg-title-lg">Token 管理</h2>
        <p class="tg-muted mt-3">查看全部 API Token 状态并处理吊销。</p>
      </NuxtLink>
      <NuxtLink class="tg-card-outline" to="/admin/sync">
        <p class="tg-eyebrow">同步任务</p>
        <h2 class="tg-title-lg">同步状态</h2>
        <p class="tg-muted mt-3">查看 clean DB 同步记录并触发同步。</p>
      </NuxtLink>
    </div>
  </section>
</template>
<script setup lang="ts">
definePageMeta({ layout: 'admin', middleware: 'admin' })
</script>
