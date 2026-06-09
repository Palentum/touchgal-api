<template>
  <div class="tg-table-wrap">
    <table class="tg-table">
      <thead>
        <tr>
          <th>项目</th>
          <th>状态</th>
          <th>请求量</th>
          <th>审核</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="app in applications" :key="app.id">
          <td>
            <p class="tg-title-sm">{{ app.projectName || app.applicantName }}</p>
            <p class="tg-muted mt-1">{{ app.projectUrl }}</p>
          </td>
          <td>
            <span class="tg-badge" :class="statusBadgeClass(app.status)">{{ app.status }}</span>
          </td>
          <td>{{ app.expectedDailyRequests }}</td>
          <td>
            <div class="tg-actions">
              <button class="tg-btn tg-btn-primary" @click="$emit('review', app.id, 'approve')">批准</button>
              <button class="tg-btn tg-btn-amber" @click="$emit('review', app.id, 'reject')">拒绝</button>
              <button class="tg-btn tg-btn-danger" @click="$emit('review', app.id, 'revoke')">撤销</button>
            </div>
          </td>
        </tr>
        <tr v-if="applications.length === 0">
          <td class="tg-empty" colspan="4">暂无待展示的申请。</td>
        </tr>
      </tbody>
    </table>
  </div>
</template>
<script setup lang="ts">
import type { ApplicationItem } from '~/composables/useDashboard'
defineProps<{ applications: ApplicationItem[] }>()
defineEmits<{ review: [id: string, action: 'approve' | 'reject' | 'revoke'] }>()

const statusBadgeClass = (status: string) => {
  if (status === 'approved') return 'tg-badge-success'
  if (status === 'rejected' || status === 'revoked') return 'tg-badge-error'
  return 'tg-badge-warning'
}
</script>
