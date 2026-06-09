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
    />

    <div
      v-if="selectedApplication"
      class="fixed inset-0 z-50 grid place-items-center bg-black/55 px-4"
      role="dialog"
      aria-modal="true"
      aria-labelledby="review-application-title"
      @click.self="closeReviewDialog"
    >
      <form class="tg-card w-full max-w-2xl" @submit.prevent="submitReview('approve')">
        <p class="tg-eyebrow">处理申请</p>
        <h2 id="review-application-title" class="tg-title-lg">审核开发者项目</h2>

        <dl class="mt-6 grid gap-4 sm:grid-cols-2">
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

        <div class="tg-form-grid mt-6">
          <label class="tg-label">
            分钟限额
            <input v-model.number="reviewLimits.minuteLimit" class="tg-input" type="number" min="1" required>
          </label>
          <label class="tg-label">
            日限额
            <input v-model.number="reviewLimits.dailyLimit" class="tg-input" type="number" min="1" required>
          </label>
        </div>

        <p v-if="reviewError" class="tg-message-error mt-4">{{ reviewError }}</p>

        <div class="tg-actions mt-6 justify-end">
          <button type="button" class="tg-btn tg-btn-amber" :disabled="submittingReview || !canSubmitReview" @click="submitReview('reject')">
            {{ submittingReview ? '处理中...' : '拒绝' }}
          </button>
          <button type="submit" class="tg-btn tg-btn-primary" :disabled="submittingReview || !canSubmitReview">
            {{ submittingReview ? '处理中...' : '批准' }}
          </button>
        </div>
      </form>
    </div>
  </section>
</template>

<script setup lang="ts">
import type { ApplicationItem } from '~/composables/useDashboard'

definePageMeta({ layout: 'admin', middleware: 'admin' })

const { apiFetch } = useApi()
const applications = ref<ApplicationItem[]>([])
const selectedApplication = ref<ApplicationItem | null>(null)
const reviewingApplicationId = ref<string | null>(null)
const processedApplicationIds = ref<string[]>([])
const reviewError = ref('')
const reviewLimits = reactive({
  minuteLimit: 60,
  dailyLimit: 5000
})

const submittingReview = computed(() => reviewingApplicationId.value !== null)
const canSubmitReview = computed(() => reviewLimits.minuteLimit > 0 && reviewLimits.dailyLimit > 0 && reviewLimits.dailyLimit >= reviewLimits.minuteLimit)

const load = async () => {
  const res = await apiFetch<ApplicationItem[]>('/admin/applications', { query: { page: 1, limit: 50 } })
  if (res.success) applications.value = res.data
}

const openReviewDialog = (application: ApplicationItem) => {
  if (application.status !== 'pending' || processedApplicationIds.value.includes(application.id)) return
  selectedApplication.value = application
  reviewError.value = ''
  reviewLimits.minuteLimit = application.defaultMinuteLimit || 60
  reviewLimits.dailyLimit = application.defaultDailyLimit || 5000
}

const closeReviewDialog = () => {
  if (submittingReview.value) return
  selectedApplication.value = null
  reviewError.value = ''
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
  if (!application || !canSubmitReview.value) return

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

onMounted(load)
</script>
