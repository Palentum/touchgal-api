<template>
  <section class="rounded-3xl border border-slate-900/10 bg-white/70 p-8">
    <h1 class="text-4xl font-black">TouchGal 主项目技术文档</h1>
    <p class="mt-4 text-slate-700">主项目仓库包含架构、部署和贡献信息。本 TouchGal API 项目只读取主库公开条目元数据，并写入独立脱敏库。</p>
    <a :href="url" target="_blank" rel="noreferrer" class="mt-8 inline-flex rounded-xl bg-slate-950 px-5 py-3 font-bold text-white">进入主项目技术文档</a>
  </section>
</template>
<script setup lang="ts">
definePageMeta({ layout: 'docs' })
const config = useRuntimeConfig()
const url = computed(() => String(config.public.touchgalTechDocsUrl))
</script>
