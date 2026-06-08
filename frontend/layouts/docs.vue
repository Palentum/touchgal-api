<template>
  <div class="min-h-screen bg-[#f6f1e8] text-slate-950">
    <header class="border-b border-slate-900/10 bg-[#f6f1e8]/90 backdrop-blur">
      <div class="mx-auto flex max-w-5xl justify-between px-6 py-4">
        <NuxtLink to="/" class="font-black">TouchGal API</NuxtLink>
        <NuxtLink to="/dashboard" class="text-sm font-semibold">控制台</NuxtLink>
      </div>
    </header>
    <main class="mx-auto max-w-5xl px-6 py-10">
      <slot />
    </main>
  </div>
</template>
