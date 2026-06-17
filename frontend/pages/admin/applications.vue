<template>
  <section class="tg-dashboard-stack">
    <div>
      <p class="tg-eyebrow">Application Review</p>
      <h1 class="tg-display-md">申请审核</h1>
      <p class="tg-lead">按项目查看开发者 API 访问申请，并执行管理员审核动作。</p>
    </div>

    <AdminApplicationReviewTable
      :applications="applications"
      :busy-application-id="reviewingApplicationId"
      :processed-application-ids="processedApplicationIds"
      @process="openReviewDialog"
      @view="openDetailsDialog"
    />

    <div
      v-if="selectedApplication"
      class="tg-dialog-backdrop fixed inset-0 z-50 grid place-items-center bg-black/55 px-4"
      role="dialog"
      aria-modal="true"
      aria-labelledby="review-application-title"
      @click.self="closeReviewDialog"
    >
      <form class="tg-dialog-panel tg-card w-full max-w-2xl" @submit.prevent="submitReview('approve')">
        <p class="tg-eyebrow">Application Review</p>
        <h2 id="review-application-title" class="tg-title-lg">{{ isReviewDialog ? '审核开发者项目' : '查看申请详情' }}</h2>

        <dl class="mt-6 grid gap-4 sm:grid-cols-2">
          <div>
            <dt class="tg-muted text-sm">申请人</dt>
            <dd class="mt-1 font-semibold text-[var(--tg-body-strong)]">{{ selectedApplication.applicantName || '未填写' }}</dd>
          </div>
          <div>
            <dt class="tg-muted text-sm">项目名称</dt>
            <dd class="mt-1 font-semibold text-[var(--tg-body-strong)]">{{ selectedApplication.projectName || '未填写' }}</dd>
          </div>
          <div>
            <dt class="tg-muted text-sm">项目地址</dt>
            <dd class="mt-1 break-all">
              <a v-if="selectedApplication.projectUrl" class="tg-link" :href="selectedApplication.projectUrl" target="_blank" rel="noopener noreferrer">
                {{ selectedApplication.projectUrl }}
              </a>
              <span v-else>未填写</span>
            </dd>
          </div>
          <div>
            <dt class="tg-muted text-sm">项目预计请求量</dt>
            <dd class="mt-1 font-semibold text-[var(--tg-body-strong)]">{{ selectedApplication.expectedDailyRequests }} 次/日</dd>
          </div>
          <div class="sm:col-span-2">
            <dt class="tg-muted text-sm">项目使用场景</dt>
            <dd class="mt-1 whitespace-pre-wrap text-[var(--tg-body-strong)]">{{ selectedApplication.usageScenario || '未填写' }}</dd>
          </div>
        </dl>

        <dl v-if="!isReviewDialog" class="mt-6 grid gap-4 rounded-3xl border border-[var(--tg-hairline-soft)] bg-[var(--tg-surface-soft)] p-4 sm:grid-cols-2">
          <div>
            <dt class="tg-muted text-sm">审核状态</dt>
            <dd class="mt-1">
              <span class="tg-badge" :class="statusBadgeClass(selectedApplication.status)">{{ statusText(selectedApplication.status) }}</span>
            </dd>
          </div>
          <div>
            <dt class="tg-muted text-sm">审核时间</dt>
            <dd class="mt-1 font-semibold text-[var(--tg-body-strong)]">{{ formatDateTime(selectedApplication.reviewedAt) }}</dd>
          </div>
          <div>
            <dt class="tg-muted text-sm">分钟限额</dt>
            <dd class="mt-1 font-semibold text-[var(--tg-body-strong)]">{{ limitText(selectedApplication.defaultMinuteLimit, '次/分钟') }}</dd>
          </div>
          <div>
            <dt class="tg-muted text-sm">日限额</dt>
            <dd class="mt-1 font-semibold text-[var(--tg-body-strong)]">{{ limitText(selectedApplication.defaultDailyLimit, '次/日') }}</dd>
          </div>
        </dl>

        <div v-if="isReviewDialog" class="tg-form-grid mt-6">
          <label class="tg-label">
            分钟限额
            <input v-model.number="reviewLimits.minuteLimit" class="tg-input" type="number" min="1" required>
          </label>
          <label class="tg-label">
            日限额
            <input v-model.number="reviewLimits.dailyLimit" class="tg-input" type="number" min="1" required>
          </label>
        </div>

        <p v-if="isReviewDialog && reviewError" class="tg-message-error mt-4">{{ reviewError }}</p>

        <div v-if="isReviewDialog" class="tg-actions mt-6 justify-end">
          <button type="button" class="tg-btn tg-btn-amber" :disabled="submittingReview || !canSubmitReview" @click="submitReview('reject')">
            {{ submittingReview ? '处理中...' : '拒绝' }}
          </button>
          <button type="submit" class="tg-btn tg-btn-primary" :disabled="submittingReview || !canSubmitReview">
            {{ submittingReview ? '处理中...' : '批准' }}
          </button>
        </div>
        <div v-else class="tg-actions mt-6 justify-end">
          <button type="button" class="tg-btn tg-btn-secondary" @click="closeReviewDialog">关闭</button>
        </div>
      </form>
    </div>
  </section>
</template>

<script setup lang="ts">
import type { ApplicationItem } from '~/composables/useDashboard'

definePageMeta({ layout: 'admin', middleware: 'admin' })

type ApplicationDialogMode = 'review' | 'view'

const { apiFetch, apiData } = useApi()
const { data: applicationsResponse, refresh: refreshApplications } = await apiData<ApplicationItem[]>('admin:applications', '/admin/applications', { query: { page: 1, limit: 50 } })
const applications = ref<ApplicationItem[]>([])
const selectedApplication = ref<ApplicationItem | null>(null)
const applicationDialogMode = ref<ApplicationDialogMode>('review')
const reviewingApplicationId = ref<string | null>(null)
const processedApplicationIds = ref<string[]>([])
const reviewError = ref('')
const reviewLimits = reactive({
  minuteLimit: 60,
  dailyLimit: 5000
})

const submittingReview = computed(() => reviewingApplicationId.value !== null)
const isReviewDialog = computed(() => applicationDialogMode.value === 'review')
const canSubmitReview = computed(() => reviewLimits.minuteLimit > 0 && reviewLimits.dailyLimit > 0 && reviewLimits.dailyLimit >= reviewLimits.minuteLimit)

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

const formatDateTime = (value?: string) => value ? value.slice(0, 19).replace('T', ' ') : '未记录'
const limitText = (value: number | undefined, unit: string) => value && value > 0 ? `${value} ${unit}` : '未设置'

const syncApplications = () => {
  if (applicationsResponse.value?.success) {
    applications.value = applicationsResponse.value.data
  }
}

watch(applicationsResponse, syncApplications, { immediate: true })

const load = async () => {
  await refreshApplications()
  syncApplications()
}

const openReviewDialog = (application: ApplicationItem) => {
  if (application.status !== 'pending' || processedApplicationIds.value.includes(application.id)) return
  applicationDialogMode.value = 'review'
  selectedApplication.value = application
  reviewError.value = ''
  reviewLimits.minuteLimit = application.defaultMinuteLimit || 60
  reviewLimits.dailyLimit = application.defaultDailyLimit || 5000
}

const openDetailsDialog = (application: ApplicationItem) => {
  if (application.status !== 'approved' && application.status !== 'rejected') return
  selectedApplication.value = application
  applicationDialogMode.value = 'view'
  reviewError.value = ''
  reviewLimits.minuteLimit = application.defaultMinuteLimit || 60
  reviewLimits.dailyLimit = application.defaultDailyLimit || 5000
}

const closeReviewDialog = () => {
  if (submittingReview.value) return
  selectedApplication.value = null
  applicationDialogMode.value = 'review'
}

const markApplicationProcessed = (application: ApplicationItem) => {
  if (!processedApplicationIds.value.includes(application.id)) {
    processedApplicationIds.value.push(application.id)
  }
  const index = applications.value.findIndex((item) => item.id === application.id)
  if (index !== -1) {
    applications.value[index] = application
  }
}

const submitReview = async (action: 'approve' | 'reject') => {
  const application = selectedApplication.value
  if (!isReviewDialog.value || !application || !canSubmitReview.value) return

  reviewingApplicationId.value = application.id
  reviewError.value = ''
  try {
    const res = await apiFetch<ApplicationItem>(`/admin/applications/${application.id}/${action}`, {
      method: 'POST',
      body: {
        minuteLimit: reviewLimits.minuteLimit,
        dailyLimit: reviewLimits.dailyLimit
      }
    })
    if (!res.success) {
      reviewError.value = res.error.message
      return
    }
    markApplicationProcessed(res.data)
    selectedApplication.value = null
    await load()
  } catch {
    reviewError.value = '审核失败，请稍后重试。'
  } finally {
    reviewingApplicationId.value = null
  }
}

</script>
