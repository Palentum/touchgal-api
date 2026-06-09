<template>
  <section class="tg-dashboard-stack">
    <div>
      <p class="tg-eyebrow">账号准入</p>
      <h1 class="tg-display-md">提交 API 申请</h1>
      <p class="tg-lead">每个账户仅可提交一次申请；审核通过后，该账户可无限生成 API token。</p>
    </div>

    <ApplicationApplyForm class="max-w-3xl" />
  </section>
</template>

<script setup lang="ts">
definePageMeta({ layout: 'dashboard', middleware: 'auth' })
</script>
