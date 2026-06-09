<template>
  <div class="tg-dashboard-stack">
    <div v-if="shownApplication" class="tg-card">
      <p class="tg-eyebrow">申请状态</p>
      <h2 class="tg-title-lg">API 申请已提交</h2>

      <div class="tg-card-outline mt-6">
        <div class="flex flex-wrap items-center justify-between gap-4">
          <div>
            <p class="tg-title-md">{{ shownApplication.projectName || shownApplication.projectUrl }}</p>
            <p class="tg-muted mt-1">{{ shownApplication.projectUrl }}</p>
          </div>
          <span class="tg-badge" :class="badge(shownApplication.status)">{{ statusText(shownApplication.status) }}</span>
        </div>
        <p v-if="shownApplication.reviewNote && shownApplication.reviewNote !== 'Reviewed'" class="tg-muted mt-4">{{ shownApplication.reviewNote }}</p>
      </div>

      <div v-if="shownApplication.status === 'approved' || shownApplication.status === 'rejected'" class="tg-actions mt-6">
        <NuxtLink v-if="shownApplication.status === 'approved'" to="/dashboard/tokens" class="tg-btn tg-btn-primary">创建 API Token</NuxtLink>
        <button v-if="shownApplication.status === 'rejected'" type="button" class="tg-btn tg-btn-primary" @click="startReapply">重新申请</button>
      </div>
    </div>

    <form v-else class="tg-card tg-form" @submit.prevent="submit">

      <label class="tg-label">
        项目名称
        <input v-model="form.projectName" class="tg-input">
      </label>

      <label class="tg-label">
        项目地址
        <input v-model="form.projectUrl" type="url" required class="tg-input" placeholder="https://example.com">
      </label>

      <label class="tg-label">
        预估每日请求量
        <input v-model="form.expectedDailyRequests" required class="tg-input">
      </label>

      <label class="tg-label">
        使用场景
        <textarea v-model="form.usageScenario" required rows="5" class="tg-textarea" />
      </label>


      <p v-if="message" :class="ok ? 'tg-message-ok' : 'tg-message-error'">{{ message }}</p>
      <button class="tg-btn tg-btn-primary justify-self-start">提交申请</button>
    </form>
  </div>
</template>

<script setup lang="ts">
import type { ApplicationItem } from '~/composables/useDashboard'

const { apiFetch } = useApi()
const auth = useAuthStore()
const access = useApplicationAccess()
const applications = access.applications
const initialForm = () => ({ projectName: '', projectUrl: '', expectedDailyRequests: '1000', usageScenario: '' })
const form = reactive(initialForm())
const message = ref('')
const ok = ref(false)
const isReapplying = ref(false)
const existingApplication = computed(() => applications.value[0])
const shownApplication = computed(() => isReapplying.value ? null : existingApplication.value)
const badge = (status: string) => status === 'approved' ? 'tg-badge-success' : status === 'pending' ? 'tg-badge-warning' : 'tg-badge-error'
const statusText = (status: string) => status === 'approved' ? '已通过' : status === 'pending' ? '审核中' : status === 'rejected' ? '未通过' : '已撤销'

const startReapply = () => {
  Object.assign(form, initialForm())
  message.value = ''
  ok.value = false
  isReapplying.value = true
}

const loadApplications = async () => {
  await access.fetchApplications(auth.user?.id)
}

const submit = async () => {
  try {
    const applicantName = auth.user?.displayName?.trim() || auth.user?.email || ''
    const res = await apiFetch<ApplicationItem>('/applications', { method: 'POST', body: { ...form, applicantName, expectedDailyRequests: Number(form.expectedDailyRequests) } })
    if (res.success) {
      applications.value = [res.data, ...applications.value.filter((app) => app.id !== res.data.id)]
      isReapplying.value = false
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
