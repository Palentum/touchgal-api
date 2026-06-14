<template>
  <section class="tg-section">
    <div class="tg-container-narrow">
      <div class="tg-grid-2">
        <div>
          <p class="tg-eyebrow">Console Login</p>
          <h1 class="tg-display-lg">登录开发者控制台</h1>
          <p class="tg-lead">使用邮箱验证码登录。</p>
          <NuxtLink to="/auth/register" class="tg-link" style="margin-top: 28px; display: inline-flex;">还没有账号？立即注册</NuxtLink>
        </div>

        <AuthEmailCodeForm mode="login" @verified="done" />
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { sanitizePostLoginRedirect } from '~/utils/redirect'

definePageMeta({
  middleware: async (to) => {
    if (import.meta.server) {
      return
    }
    const auth = useAuthStore()
    if (!auth.loaded) {
      await auth.fetchMe()
    }
    if (auth.hasAuthFetchError && import.meta.client) {
      await auth.fetchMe({ refresh: true })
    }
    if (auth.hasAuthFetchError) {
      return
    }
    if (auth.isAccountDisabled) {
      return navigateTo('/account-disabled')
    }
    if (auth.user) {
      return navigateTo(sanitizePostLoginRedirect(to.query.redirect))
    }
  }
})

const route = useRoute()
const done = async () => {
  await navigateTo(sanitizePostLoginRedirect(route.query.redirect))
}
</script>
