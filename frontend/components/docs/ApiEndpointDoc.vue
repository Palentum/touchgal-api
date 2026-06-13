<template>
  <article class="tg-doc-article tg-api-doc">
    <header class="tg-card-coral">
      <p class="tg-eyebrow" style="color: var(--tg-on-primary);">{{ doc.method }} {{ doc.path }}</p>
      <h1 class="tg-display-lg">{{ doc.name }}</h1>
      <p style="margin-top: 20px; max-width: 760px; line-height: 1.7;">{{ doc.introduction }}</p>
    </header>

    <section class="tg-card-outline">
      <h2 class="tg-title-lg">接口基础信息</h2>
      <div class="tg-doc-meta-grid">
        <div>
          <p class="tg-stat-label">接口名称</p>
          <p class="tg-doc-meta-value">{{ doc.name }}</p>
        </div>
        <div>
          <p class="tg-stat-label">请求方法</p>
          <p class="tg-doc-meta-value"><code>{{ doc.method }}</code></p>
        </div>
        <div>
          <p class="tg-stat-label">请求路径</p>
          <p class="tg-doc-meta-value"><code>{{ doc.path }}</code></p>
        </div>
        <div>
          <p class="tg-stat-label">鉴权</p>
          <p class="tg-doc-meta-value">{{ doc.auth }}</p>
        </div>
      </div>
    </section>

    <section class="tg-card-outline">
      <div class="tg-doc-section-head">
        <h2 class="tg-title-lg">请求参数</h2>
        <span v-if="!doc.parameters.length" class="tg-badge tg-badge-success">No parameters</span>
      </div>

      <p v-if="!doc.parameters.length" class="tg-muted tg-doc-empty-line">该接口无 header、path、query 或 body 参数。</p>
      <div v-else class="tg-table-wrap" data-mobile-cards="true">
        <table class="tg-table">
          <thead>
            <tr>
              <th>参数</th>
              <th>位置</th>
              <th>必填</th>
              <th>类型</th>
              <th>说明</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="param in doc.parameters" :key="`${param.location}:${param.name}`">
              <td data-label="参数"><code>{{ param.name }}</code></td>
              <td data-label="位置">{{ param.location }}</td>
              <td data-label="必填">{{ requiredLabel(param.required) }}</td>
              <td data-label="类型"><code>{{ param.type }}</code></td>
              <td data-label="说明">{{ param.description }}</td>
            </tr>
          </tbody>
        </table>
      </div>
    </section>

    <section class="tg-card-dark">
      <h2 class="tg-title-lg">请求示例</h2>
      <p class="tg-muted" style="margin-top: 12px;">示例域名请替换为实际部署域名；业务接口不要在不可信浏览器环境暴露 token。</p>
      <div class="tg-code-window tg-doc-code-window">
        <div class="tg-code-window-bar">
          <span class="tg-window-dots" aria-hidden="true"><span /><span /><span /></span>
          <span>curl</span>
        </div>
        <pre><code>{{ doc.requestExample }}</code></pre>
      </div>
    </section>

    <section class="tg-card-outline">
      <h2 class="tg-title-lg">返回状态码与响应示例</h2>
      <div class="tg-doc-status-list">
        <article v-for="status in doc.statuses" :key="status.code" class="tg-doc-status-card">
          <div class="tg-doc-status-head">
            <span class="tg-badge" :class="status.code < 400 ? 'tg-badge-success' : status.code === 429 ? 'tg-badge-warning' : 'tg-badge-error'">
              {{ status.code }}
            </span>
            <div>
              <h3 class="tg-title-md">{{ status.title }}</h3>
              <p class="tg-muted">{{ status.description }}</p>
            </div>
          </div>

          <div class="tg-code-window tg-doc-code-window">
            <div class="tg-code-window-bar">
              <span class="tg-window-dots" aria-hidden="true"><span /><span /><span /></span>
              <span>application/json</span>
            </div>
            <pre><code>{{ status.example }}</code></pre>
          </div>

          <div class="tg-doc-field-list">
            <h4 class="tg-title-sm">返回内容解释</h4>
            <dl>
              <template v-for="field in status.fields" :key="field.name">
                <dt><code>{{ field.name }}</code></dt>
                <dd>{{ field.description }}</dd>
              </template>
            </dl>
          </div>
        </article>
      </div>
    </section>
  </article>
</template>

<script setup lang="ts">
import type { ApiEndpointDoc, ApiDocParameter } from '~/composables/apiDocs'

defineProps<{ doc: ApiEndpointDoc }>()

const requiredLabel = (required: ApiDocParameter['required']) => {
  if (required === 'conditional') return '条件必填'
  return required ? '是' : '否'
}
</script>

<style scoped>
.tg-api-doc {
  max-width: 100%;
}

.tg-doc-meta-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 18px;
  margin-top: 20px;
}

.tg-doc-meta-value {
  margin: 6px 0 0;
  color: var(--tg-body-strong);
  line-height: 1.55;
  overflow-wrap: anywhere;
}

.tg-doc-section-head,
.tg-doc-status-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
}

.tg-doc-empty-line {
  margin-top: 14px;
}

.tg-doc-code-window {
  margin-top: 18px;
  max-width: 100%;
}

.tg-doc-code-window pre {
  max-width: 100%;
  overflow-x: auto;
}

.tg-doc-status-list {
  display: grid;
  gap: 18px;
  margin-top: 20px;
}

.tg-doc-status-card {
  border: 1px solid var(--tg-hairline-soft);
  border-radius: 14px;
  background: var(--tg-canvas);
  padding: 20px;
}

.tg-doc-status-head {
  justify-content: flex-start;
}

.tg-doc-status-head .tg-badge {
  flex: 0 0 auto;
}

.tg-doc-field-list {
  margin-top: 18px;
}

.tg-doc-field-list dl {
  display: grid;
  grid-template-columns: minmax(180px, 0.35fr) minmax(0, 1fr);
  gap: 10px 16px;
  margin: 14px 0 0;
}

.tg-doc-field-list dt,
.tg-doc-field-list dd {
  margin: 0;
  line-height: 1.55;
}

.tg-doc-field-list dt {
  color: var(--tg-body-strong);
  font-weight: 600;
}

.tg-doc-field-list dd {
  color: var(--tg-muted);
}

@media (max-width: 767px) {
  .tg-doc-meta-grid,
  .tg-doc-field-list dl {
    grid-template-columns: 1fr;
  }

  .tg-doc-section-head,
  .tg-doc-status-head {
    display: grid;
  }
}
</style>
