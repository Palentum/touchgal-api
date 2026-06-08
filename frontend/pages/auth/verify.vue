<template>
  <section class="mx-auto max-w-xl px-6 py-16">
    <h1 class="text-4xl font-black">验证码验证</h1>
    <p class="mt-4 text-slate-300">注册和登录页面已内置验证码输入步骤。若验证码过期，请返回对应页面重新发送。</p>
    <div class="mt-8 flex gap-3"><NuxtLink class="rounded-xl bg-emerald-400 px-4 py-3 font-bold text-slate-950" to="/auth/login">登录</NuxtLink><NuxtLink class="rounded-xl border border-white/10 px-4 py-3" to="/auth/register">注册</NuxtLink></div>
  </section>
</template>
