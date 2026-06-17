<template>
  <div class="tg-table-wrap" data-mobile-cards="true">
    <table class="tg-table tg-application-review-table">
      <thead>
        <tr>
          <th>项目</th>
          <th>申请人</th>
          <th class="text-center">状态</th>
          <th class="text-right" aria-label="操作"></th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="app in applications" :key="app.id">
          <td data-label="项目">
            <p class="tg-title-sm">{{ app.projectName || '未填写项目名称' }}</p>
            <p class="tg-muted mt-1">{{ app.projectUrl || '未填写项目地址' }}</p>
          </td>
          <td data-label="申请人">
            <p class="tg-title-sm tg-application-owner-name">{{ ownerDisplayName(app) }}</p>
            <p class="tg-muted mt-1">{{ ownerAccount(app) }}</p>
          </td>
          <td class="text-center" data-label="状态">
            <span class="tg-badge" :class="statusBadgeClass(app.status)">{{ statusText(app.status) }}</span>
          </td>
          <td data-label="操作">
            <div class="flex flex-wrap justify-end gap-1 sm:gap-2">
              <button
                v-if="app.status === 'pending'"
                type="button"
                class="tg-icon-btn"
                :aria-label="`处理 ${app.projectName || app.applicantName}`"
                title="处理"
                :disabled="isProcessDisabled(app)"
                @click="$emit('process', app)"
              >
                <svg viewBox="0 0 24 24" aria-hidden="true" class="h-4 w-4" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round">
                  <path d="M8 5h8" />
                  <path d="M8 12h8" />
                  <path d="M8 19h5" />
                  <path d="M19 16.5 21 18.5 19 20.5" />
                </svg>
              </button>
              <button
                v-if="canView(app)"
                type="button"
                class="tg-icon-btn"
                :aria-label="`查看 ${app.projectName || app.applicantName}`"
                title="查看"
                :disabled="props.busyApplicationId === app.id"
                @click="$emit('view', app)"
              >
                <svg viewBox="0 0 24 24" aria-hidden="true" class="h-4 w-4" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round">
                  <path d="M2.5 12s3.5-6.5 9.5-6.5S21.5 12 21.5 12 18 18.5 12 18.5 2.5 12 2.5 12Z" />
                  <circle cx="12" cy="12" r="2.5" />
                </svg>
              </button>
            </div>
          </td>
        </tr>
        <tr v-if="applications.length === 0">
          <td class="tg-empty" colspan="4" data-label="">暂无待展示的申请。</td>
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

defineEmits<{ process: [application: ApplicationItem]; view: [application: ApplicationItem] }>()

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

const ownerDisplayName = (application: ApplicationItem) => application.owner?.displayName || application.applicantName || '未设置昵称'
const ownerAccount = (application: ApplicationItem) => application.owner?.email || '未获取邮箱'

const canView = (application: ApplicationItem) => application.status === 'approved' || application.status === 'rejected'
const isProcessDisabled = (application: ApplicationItem) => props.busyApplicationId === application.id || props.processedApplicationIds.includes(application.id)
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

.tg-application-owner-name.tg-title-sm {
  color: var(--tg-body-strong);
}
</style>
