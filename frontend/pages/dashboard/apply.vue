<template>
  <section class="grid gap-6">
    <div>
      <h1 class="text-4xl font-black">提交 API 申请</h1>
      <p class="mt-3 max-w-3xl text-slate-300">每个账户仅可提交一次申请；审核通过后，该账户可无限生成 API token。</p>
    </div>
    <ApplicationApplyForm class="max-w-3xl" />
  </section>
</template>

<script setup lang="ts">
definePageMeta({ layout: 'dashboard', middleware: 'auth' })
</script>
