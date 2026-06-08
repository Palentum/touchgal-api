<template>
  <div class="min-h-screen bg-[#08111f] text-slate-100">
    <header class="sticky top-0 z-30 border-b border-white/10 bg-[#08111f]/85 backdrop-blur">
      <nav class="mx-auto flex max-w-6xl items-center justify-between px-6 py-4">
        <NuxtLink to="/" class="text-lg font-black tracking-tight">TouchGal API</NuxtLink>
        <div class="flex items-center gap-4 text-sm text-slate-300">
          <NuxtLink to="/docs/api" class="hover:text-white">API 文档</NuxtLink>
          <NuxtLink to="/auth/register" class="hover:text-white">申请 API</NuxtLink>
          <NuxtLink to="/dashboard" class="rounded-full bg-emerald-400 px-4 py-2 font-semibold text-slate-950">用户后台</NuxtLink>
        </div>
      </nav>
    </header>
    <main>
      <slot />
    </main>
  </div>
</template>
