<template>
  <div class="tg-card">
    <p class="tg-eyebrow">准入状态</p>
    <h2 class="tg-title-lg">账号申请状态</h2>

    <div class="tg-dashboard-stack mt-5">
      <div v-for="app in applications" :key="app.id" class="tg-card-outline">
        <div class="flex flex-wrap items-start justify-between gap-4">
          <div>
            <p class="tg-title-md">{{ app.projectName || app.projectUrl }}</p>
            <p class="tg-muted mt-1">{{ app.projectUrl }}</p>
          </div>
          <span class="tg-badge" :class="badge(app.status)">{{ statusText(app.status) }}</span>
        </div>
        <p v-if="app.reviewNote" class="tg-muted mt-4">{{ app.reviewNote }}</p>
      </div>

      <p v-if="applications.length === 0" class="tg-empty">暂无账号申请。</p>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { ApplicationItem } from '~/composables/useDashboard'

defineProps<{ applications: ApplicationItem[] }>()
const badge = (status: string) => status === 'approved' ? 'tg-badge-success' : status === 'pending' ? 'tg-badge-warning' : 'tg-badge-error'
const statusText = (status: string) => status === 'approved' ? '已通过' : status === 'pending' ? '审核中' : status === 'rejected' ? '未通过' : '已撤销'
</script>
