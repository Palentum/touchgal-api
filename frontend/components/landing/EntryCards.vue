<template>
  <section class="mx-auto grid max-w-6xl gap-5 px-6 pb-16 md:grid-cols-2">
    <a :href="techUrl" target="_blank" rel="noreferrer" class="group rounded-[2rem] border border-white/10 bg-white/[0.06] p-8 transition hover:-translate-y-1 hover:bg-white/[0.1]">
      <p class="text-sm uppercase tracking-[0.35em] text-emerald-300">KunMoe Docs</p>
      <h2 class="mt-4 text-3xl font-black">KunMoe/kun-touchgal-next 技术文档</h2>
      <p class="mt-4 text-slate-300">了解 TouchGal 主项目架构、部署和贡献方式。</p>
      <span class="mt-8 inline-block text-emerald-200 group-hover:translate-x-1">进入主项目文档 →</span>
    </a>
    <NuxtLink :to="apiDocsUrl" class="group rounded-[2rem] border border-emerald-300/30 bg-emerald-300/10 p-8 transition hover:-translate-y-1 hover:bg-emerald-300/15">
      <p class="text-sm uppercase tracking-[0.35em] text-sky-200">API Docs</p>
      <h2 class="mt-4 text-3xl font-black">本 API 文档</h2>
      <p class="mt-4 text-slate-300">查看鉴权、搜索、详情、限流与标准响应格式。</p>
      <span class="mt-8 inline-block text-sky-100 group-hover:translate-x-1">查看 API 文档 →</span>
    </NuxtLink>
  </section>
</template>

<script setup lang="ts">
const config = useRuntimeConfig()
const techUrl = computed(() => String(config.public.touchgalTechDocsUrl))
const apiDocsUrl = computed(() => String(config.public.apiDocsUrl))
</script>
