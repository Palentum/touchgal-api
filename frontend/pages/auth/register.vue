<template>
  <section class="mx-auto max-w-md px-6 py-16">
    <h1 class="text-4xl font-black">注册 TouchGal API</h1>
    <p class="mt-3 text-slate-300">本项目账号独立于 TouchGal 主站。</p>
    <AuthEmailCodeForm class="mt-8" mode="register" @verified="done" />
    <NuxtLink to="/auth/login" class="mt-5 block text-sm text-emerald-300">已有账号？登录</NuxtLink>
  </section>
</template>
<script setup lang="ts">
const done = async () => { await navigateTo('/dashboard') }
</script>
