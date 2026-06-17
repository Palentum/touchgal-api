<template>
  <section class="tg-dashboard-stack">
    <div>
      <p class="tg-eyebrow">Submit Application</p>
      <h1 class="tg-display-md">提交 API 申请</h1>
    </div>

    <ApplicationApplyForm class="max-w-3xl" />
  </section>
</template>

<script setup lang="ts">
definePageMeta({ layout: 'dashboard', middleware: 'auth' })
</script>
