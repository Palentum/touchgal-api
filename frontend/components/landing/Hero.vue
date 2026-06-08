<template>
  <section class="relative overflow-hidden px-6 py-24">
    <div class="absolute inset-0 bg-[radial-gradient(circle_at_20%_20%,rgba(16,185,129,.25),transparent_35%),radial-gradient(circle_at_80%_10%,rgba(56,189,248,.2),transparent_30%)]" />
    <div class="relative mx-auto max-w-6xl">
      <p class="mb-5 inline-flex rounded-full border border-emerald-300/30 bg-emerald-300/10 px-4 py-2 text-sm text-emerald-100">独立脱敏库 · Token 限流 · 开发者控制台</p>
      <h1 class="max-w-4xl text-5xl font-black leading-tight tracking-tight md:text-7xl">TouchGal Developer API</h1>
      <p class="mt-6 max-w-2xl text-xl text-slate-300">为开发者提供稳定、脱敏、可限流的 Galgame 条目 API，账号、数据库与 KunMoe 主项目完全隔离。</p>
      <div class="mt-10 flex flex-wrap gap-4">
        <NuxtLink to="/apply" class="rounded-full bg-emerald-400 px-6 py-3 font-bold text-slate-950 shadow-lg shadow-emerald-500/20">立即申请 API</NuxtLink>
        <NuxtLink to="/dashboard" class="rounded-full border border-white/15 px-6 py-3 font-bold text-white">登录控制台</NuxtLink>
      </div>
    </div>
  </section>
</template>
