<template>
  <section class="tg-section">
    <div class="tg-container-narrow">
      <div class="tg-card-outline">
        <span class="tg-badge tg-badge-warning">Email Code</span>
        <h1 class="tg-display-md" style="margin-top: 20px;">验证码验证</h1>
        <p class="tg-lead">注册和登录页面已内置验证码输入步骤。若验证码过期，请返回对应页面重新发送。</p>
        <div class="tg-actions" style="margin-top: 28px;">
          <NuxtLink class="tg-btn tg-btn-primary" to="/auth/login">登录</NuxtLink>
          <NuxtLink class="tg-btn tg-btn-secondary" to="/auth/register">注册</NuxtLink>
        </div>
      </div>
    </div>
  </section>
</template>
