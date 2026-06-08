<template>
  <section class="mx-auto max-w-3xl px-6 py-10">
    <h1 class="text-4xl font-black">提交 API 申请</h1>
    <p class="mt-3 text-slate-300">每个账户仅可提交一次申请；审核通过后，该账户可无限生成 API token。</p>
    <ApplicationApplyForm class="mt-8" />
  </section>
</template>
<script setup lang="ts">
definePageMeta({ middleware: 'auth' })
</script>
