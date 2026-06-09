<template>
  <section class="tg-section">
    <div class="tg-container-narrow">
      <div class="tg-grid-2">
        <div>
          <p class="tg-eyebrow">Console Login</p>
          <h1 class="tg-display-lg">邮箱验证码登录</h1>
          <p class="tg-lead">登录后可提交 API 申请、生成 token，并查看请求统计。session 继续由 HttpOnly Cookie 维护。</p>
          <NuxtLink to="/auth/register" class="tg-link" style="margin-top: 28px; display: inline-flex;">还没有账号？立即注册</NuxtLink>
        </div>

        <AuthEmailCodeForm mode="login" @verified="done" />
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
definePageMeta({
  middleware: async () => {
    if (import.meta.server) {
      return
    }

    const auth = useAuthStore()
    if (!auth.loaded) {
      await auth.fetchMe()
    }
    if (auth.isAccountDisabled) {
      return navigateTo('/account-disabled')
    }
    if (auth.user) {
      return navigateTo('/dashboard')
    }
  }
})

const route = useRoute()
const done = async () => {
  await navigateTo(String(route.query.redirect || '/dashboard'))
}
</script>
