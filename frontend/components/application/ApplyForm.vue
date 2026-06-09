<template>
  <div class="tg-dashboard-stack">
    <div v-if="existingApplication" class="tg-card">
      <p class="tg-eyebrow">申请状态</p>
      <h2 class="tg-title-lg">账号申请已提交</h2>
      <p class="tg-muted mt-3">每个账户仅需提交一次申请。审核通过后，该账户可无限创建 API token。</p>

      <div class="tg-card-outline mt-6">
        <div class="flex flex-wrap items-start justify-between gap-4">
          <div>
            <p class="tg-title-md">{{ existingApplication.projectName || existingApplication.projectUrl }}</p>
            <p class="tg-muted mt-1">{{ existingApplication.projectUrl }}</p>
          </div>
          <span class="tg-badge" :class="badge(existingApplication.status)">{{ statusText(existingApplication.status) }}</span>
        </div>
        <p v-if="existingApplication.reviewNote" class="tg-muted mt-4">{{ existingApplication.reviewNote }}</p>
      </div>

      <NuxtLink v-if="existingApplication.status === 'approved'" to="/dashboard/tokens" class="tg-btn tg-btn-primary mt-6">创建 API Token</NuxtLink>
    </div>

    <form v-else class="tg-card tg-form" @submit.prevent="submit">
      <div>
        <p class="tg-eyebrow">提交申请</p>
        <h2 class="tg-title-lg">账号级 API 申请</h2>
        <p class="tg-muted mt-3">此申请为账户级申请，每个账户只能提交一次。</p>
      </div>

      <div class="tg-form-grid">
        <label class="tg-label">
          申请用户/负责人
          <input v-model="form.applicantName" required class="tg-input">
        </label>
        <label class="tg-label">
          项目名称
          <input v-model="form.projectName" class="tg-input">
        </label>
      </div>

      <label class="tg-label">
        项目地址
        <input v-model="form.projectUrl" type="url" required class="tg-input" placeholder="https://example.com">
      </label>

      <label class="tg-label">
        预估每日请求量
        <input v-model.number="form.expectedDailyRequests" type="number" min="1" required class="tg-input">
      </label>

      <label class="tg-label">
        使用场景
        <textarea v-model="form.usageScenario" required rows="5" class="tg-textarea" />
      </label>

      <label class="flex items-start gap-3 text-sm">
        <input v-model="form.agreeToTerms" type="checkbox" class="mt-1">
        <span class="tg-muted">我确认不会使用 API 还原用户、资源下载、评论或任何主站隐私数据。</span>
      </label>

      <p v-if="message" :class="ok ? 'tg-message-ok' : 'tg-message-error'">{{ message }}</p>
      <button class="tg-btn tg-btn-primary justify-self-start">提交账号申请</button>
    </form>
  </div>
</template>

<script setup lang="ts">
import type { ApplicationItem } from '~/composables/useDashboard'

const { apiFetch } = useApi()
const auth = useAuthStore()
const access = useApplicationAccess()
const applications = access.applications
const form = reactive({ applicantName: '', projectName: '', projectUrl: '', expectedDailyRequests: 1000, usageScenario: '', agreeToTerms: false })
const message = ref('')
const ok = ref(false)
const existingApplication = computed(() => applications.value[0])
const badge = (status: string) => status === 'approved' ? 'tg-badge-success' : status === 'pending' ? 'tg-badge-warning' : 'tg-badge-error'
const statusText = (status: string) => status === 'approved' ? '已通过' : status === 'pending' ? '审核中' : status === 'rejected' ? '未通过' : '已撤销'

const loadApplications = async () => {
  await access.fetchApplications(auth.user?.id)
}

const submit = async () => {
  try {
    const res = await apiFetch<ApplicationItem>('/applications', { method: 'POST', body: form })
    if (res.success) {
      applications.value = [res.data]
      ok.value = true
      message.value = '申请已提交，等待管理员审核。'
    }
  } catch {
    ok.value = false
    message.value = '提交失败，请检查 URL、请求量和必填项。'
  }
}

onMounted(loadApplications)
</script>
