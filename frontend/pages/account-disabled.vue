<template>
  <section class="tg-section tg-account-disabled">
    <div class="tg-container-narrow">
      <div class="tg-card tg-disabled-card">
        <div>
          <p class="tg-eyebrow">Account Suspended</p>
          <h1 class="tg-display-lg">账户已被停用</h1>
          <p class="tg-lead">该账户已无法请求 API、生成 API Token 或访问开发者后台。</p>

          <div class="tg-disabled-panel">
            <p v-if="auth.user?.email" class="tg-muted">停用账户：<strong>{{ auth.user.email }}</strong></p>
            <p v-else class="tg-muted">如果你刚刚尝试登录，此邮箱对应的开发者账户已被停用。</p>
            <p class="tg-muted">如需恢复访问，请联系 TouchGal API 管理员并进行情况说明。</p>
          </div>

          <div class="tg-actions tg-disabled-actions">
            <button v-if="auth.user" class="tg-btn tg-btn-secondary" type="button" :disabled="signingOut" @click="signOut">
              {{ signingOut ? '退出中...' : '退出当前账号' }}
            </button>
            <button v-else class="tg-btn tg-btn-secondary" type="button" @click="continueToLogin">使用其他账号登录</button>
            <a class="tg-btn tg-btn-primary" href="https://discord.gg/e4QePvPQTB" target="_blank" rel="noreferrer">联系管理员</a>
            <NuxtLink class="tg-link" to="/">返回首页</NuxtLink>
          </div>
        </div>
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
    if (auth.user && !auth.isAccountDisabled) {
      return navigateTo('/dashboard')
    }
  }
})

const auth = useAuthStore()
const signingOut = ref(false)

const signOut = async () => {
  signingOut.value = true
  try {
    await auth.logout()
    await navigateTo('/auth/login')
  } finally {
    signingOut.value = false
  }
}

const continueToLogin = async () => {
  auth.clearAccountDisabled()
  await navigateTo('/auth/login')
}
</script>

<style scoped>
.tg-account-disabled {
  min-height: calc(100vh - 64px);
  display: flex;
  align-items: center;
  background:
    radial-gradient(circle at 82% 18%, rgba(198, 69, 69, 0.12), transparent 22rem),
    radial-gradient(circle at 18% 88%, rgba(232, 165, 90, 0.14), transparent 26rem);
}

.tg-disabled-card {
  display: block;
  border: 1px solid rgba(198, 69, 69, 0.18);
  box-shadow: 0 28px 80px rgba(37, 35, 32, 0.08);
}
.tg-disabled-panel {
  margin-top: 28px;
  border-left: 3px solid var(--tg-error);
  padding: 16px 0 16px 20px;
  background: rgba(250, 249, 245, 0.42);
}

.tg-disabled-panel p {
  margin: 0;
}

.tg-disabled-panel p + p {
  margin-top: 8px;
}

.tg-disabled-actions {
  margin-top: 28px;
}

@media (max-width: 720px) {
  .tg-account-disabled {
    align-items: flex-start;
  }
}
</style>
