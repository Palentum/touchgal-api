<template>
  <div>
    <LandingHero />
    <LandingEntryCards />
    <section class="mx-auto grid max-w-6xl gap-4 px-6 pb-24 md:grid-cols-5">
      <div v-for="feature in features" :key="feature" class="rounded-3xl border border-white/10 bg-white/[0.05] p-5 text-sm text-slate-200">{{ feature }}</div>
    </section>
  </div>
</template>
<script setup lang="ts">
const features = ['游戏关键词搜索', 'uniqueId 详情查询', '脱敏独立数据库', 'API token 管理', '请求统计图表']
</script>
