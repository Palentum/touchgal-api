<template>
  <div class="tg-table-wrap" data-mobile-cards="true">
    <table class="tg-table tg-application-review-table">
      <thead>
        <tr>
          <th>项目</th>
          <th class="text-center">状态</th>
          <th class="text-right" aria-label="操作"></th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="app in applications" :key="app.id">
          <td data-label="项目">
            <p class="tg-title-sm">{{ app.projectName || app.applicantName }}</p>
            <p class="tg-muted mt-1">{{ app.projectUrl || '未填写项目地址' }}</p>
          </td>
          <td class="text-center" data-label="状态">
            <span class="tg-badge" :class="statusBadgeClass(app.status)">{{ statusText(app.status) }}</span>
          </td>
          <td data-label="操作">
            <div class="flex justify-end">
              <button
                type="button"
                class="tg-icon-btn"
                :aria-label="`处理 ${app.projectName || app.applicantName}`"
                title="处理"
                :disabled="isProcessed(app)"
                @click="$emit('process', app)"
              >
                <svg viewBox="0 0 24 24" aria-hidden="true" class="h-4 w-4" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round">
                  <path d="M8 5h8" />
                  <path d="M8 12h8" />
                  <path d="M8 19h5" />
                  <path d="M19 16.5 21 18.5 19 20.5" />
                </svg>
              </button>
            </div>
          </td>
        </tr>
        <tr v-if="applications.length === 0">
          <td class="tg-empty" colspan="3" data-label="">暂无待展示的申请。</td>
        </tr>
      </tbody>
    </table>
  </div>
</template>
<script setup lang="ts">
import type { ApplicationItem } from '~/composables/useDashboard'

const props = withDefaults(defineProps<{
  applications: ApplicationItem[]
  busyApplicationId?: string | null
  processedApplicationIds?: string[]
}>(), {
  busyApplicationId: null,
  processedApplicationIds: () => []
})

defineEmits<{ process: [application: ApplicationItem] }>()

const statusText = (status: string) => {
  if (status === 'approved') return '已批准'
  if (status === 'rejected') return '已拒绝'
  if (status === 'revoked') return '已撤销'
  return '待处理'
}

const statusBadgeClass = (status: string) => {
  if (status === 'approved') return 'tg-badge-success'
  if (status === 'rejected' || status === 'revoked') return 'tg-badge-error'
  return 'tg-badge-warning'
}

const isProcessed = (application: ApplicationItem) => application.status !== 'pending' || props.busyApplicationId === application.id || props.processedApplicationIds.includes(application.id)
</script>

<style scoped>
.tg-application-review-table :deep(th),
.tg-application-review-table :deep(td) {
  vertical-align: middle;
}

.tg-application-review-table .tg-icon-btn:disabled {
  background: var(--tg-primary-disabled);
  color: var(--tg-muted);
  opacity: 1;
}
</style>
