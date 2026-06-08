<template>
  <section class="grid gap-6">
    <h1 class="text-4xl font-black">管理员后台</h1>
    <div class="grid gap-4 md:grid-cols-3"><NuxtLink class="rounded-3xl border border-white/10 bg-white/[0.06] p-8 text-xl font-black hover:bg-white/[0.1]" to="/admin/applications">审核申请</NuxtLink><NuxtLink class="rounded-3xl border border-white/10 bg-white/[0.06] p-8 text-xl font-black hover:bg-white/[0.1]" to="/admin/tokens">Token 管理</NuxtLink><NuxtLink class="rounded-3xl border border-white/10 bg-white/[0.06] p-8 text-xl font-black hover:bg-white/[0.1]" to="/admin/sync">同步状态</NuxtLink></div>
  </section>
</template>
<script setup lang="ts">
definePageMeta({ layout: 'dashboard', middleware: 'admin' })
</script>
