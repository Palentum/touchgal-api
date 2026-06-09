<template>
  <section class="tg-doc-article">
    <div class="tg-card-outline">
      <span class="tg-badge tg-badge-coral">TouchGal Docs</span>
      <h1 class="tg-display-md" style="margin-top: 20px;">TouchGal 主项目技术文档</h1>
      <p class="tg-lead">
        主项目仓库包含架构、部署和贡献信息。本 TouchGal API 项目只读取主库公开条目元数据，并写入独立脱敏库。
      </p>
      <a :href="url" target="_blank" rel="noreferrer" class="tg-btn tg-btn-primary" style="margin-top: 28px;">进入主项目技术文档</a>
    </div>

    <div class="tg-card-dark">
      <h2 class="tg-title-lg" style="color: var(--tg-on-dark);">边界说明</h2>
      <p class="tg-muted" style="margin-top: 14px; line-height: 1.7;">
        API 服务面向第三方开发者，只暴露 clean DB 中允许公开的元数据；内部 source 字段、主站用户信息、评论和资源下载链接不进入公开响应。
      </p>
    </div>
  </section>
</template>

<script setup lang="ts">
definePageMeta({ layout: 'docs' })
const config = useRuntimeConfig()
const url = computed(() => String(config.public.touchgalTechDocsUrl))
</script>
