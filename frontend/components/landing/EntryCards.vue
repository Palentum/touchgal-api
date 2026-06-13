<template>
  <section class="tg-section-tight">
    <div class="tg-container">
      <div class="tg-grid-2">
        <a :href="techUrl" target="_blank" rel="noreferrer" class="tg-card-outline">
          <span class="tg-badge tg-badge-coral">TouchGal Docs</span>
          <h2 class="tg-display-sm" style="margin-top: 24px;">主项目技术文档</h2>
          <p class="tg-lead">
            了解 TouchGal 主项目架构、部署和贡献方式。
          </p>
          <span class="tg-link" style="margin-top: 28px; display: inline-flex;">进入主项目文档 →</span>
        </a>

        <NuxtLink :to="apiDocsUrl" class="tg-card-coral">
          <span class="tg-badge">API Docs</span>
          <h2 class="tg-display-sm" style="margin-top: 24px;">本 API 文档</h2>
          <p style="margin-top: 20px; line-height: 1.7;">
            查看鉴权、搜索、详情、限流与标准响应格式。示例覆盖 curl、TypeScript fetch 和 Go net/http。
          </p>
          <span class="tg-btn tg-btn-cream" style="margin-top: 28px;">查看 API 文档 →</span>
        </NuxtLink>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
const config = useRuntimeConfig()
const techUrl = computed(() => String(config.public.touchgalTechDocsUrl))
const apiDocsUrl = computed(() => String(config.public.apiDocsUrl))
</script>
