<template>
  <section class="tg-section">
    <div class="tg-container-narrow">
      <div class="tg-grid-2">
        <div>
          <p class="tg-eyebrow">Developer Access</p>
          <h1 class="tg-display-lg">注册 TouchGal API</h1>
          <p class="tg-lead">本项目账号独立于 TouchGal 主站。完成邮箱验证后，即可提交 API 访问申请。</p>
          <NuxtLink to="/auth/login" class="tg-link" style="margin-top: 28px; display: inline-flex;">已有账号？登录</NuxtLink>
        </div>

        <AuthEmailCodeForm mode="register" @verified="done" />
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
const done = async () => { await navigateTo('/dashboard/apply') }
</script>
