<template>
  <section class="mx-auto max-w-md px-6 py-16">
    <h1 class="text-4xl font-black">邮箱验证码登录</h1>
    <p class="mt-3 text-slate-300">登录后可提交 API 申请、生成 token 并查看请求统计。</p>
    <AuthEmailCodeForm class="mt-8" mode="login" @verified="done" />
    <NuxtLink to="/auth/register" class="mt-5 block text-sm text-emerald-300">还没有账号？立即注册</NuxtLink>
  </section>
</template>
<script setup lang="ts">
const route = useRoute()
const done = async () => {
  await navigateTo(String(route.query.redirect || '/dashboard'))
}
</script>
