<template>
  <div class="grid gap-4">
    <div v-if="existingApplication" class="rounded-3xl border border-white/10 bg-white/[0.06] p-6">
      <h3 class="text-xl font-black">账号申请已提交</h3>
      <p class="mt-2 text-sm text-slate-300">每个账户仅需提交一次申请。审核通过后，该账户可无限创建 API token。</p>
      <div class="mt-5 rounded-2xl bg-slate-950/70 p-4">
        <div class="flex items-center justify-between gap-4">
          <div><p class="font-bold">{{ existingApplication.projectName || existingApplication.projectUrl }}</p><p class="text-sm text-slate-400">{{ existingApplication.projectUrl }}</p></div>
          <span class="rounded-full px-3 py-1 text-xs" :class="badge(existingApplication.status)">{{ statusText(existingApplication.status) }}</span>
        </div>
        <p v-if="existingApplication.reviewNote" class="mt-3 text-sm text-slate-300">{{ existingApplication.reviewNote }}</p>
      </div>
      <NuxtLink v-if="existingApplication.status === 'approved'" to="/dashboard/tokens" class="mt-5 inline-flex rounded-xl bg-emerald-400 px-5 py-3 font-bold text-slate-950">创建 API Token</NuxtLink>
    </div>
    <form v-else class="grid gap-4 rounded-3xl border border-white/10 bg-white/[0.06] p-6" @submit.prevent="submit">
      <p class="text-sm text-slate-300">此申请为账户级申请，每个账户只能提交一次。</p>
      <div class="grid gap-4 md:grid-cols-2">
        <label class="grid gap-2 text-sm">申请用户/负责人<input v-model="form.applicantName" required class="rounded-xl border border-white/10 bg-slate-950 px-4 py-3 text-white outline-none focus:border-emerald-300"></label>
        <label class="grid gap-2 text-sm">项目名称<input v-model="form.projectName" class="rounded-xl border border-white/10 bg-slate-950 px-4 py-3 text-white outline-none focus:border-emerald-300"></label>
      </div>
      <label class="grid gap-2 text-sm">项目地址<input v-model="form.projectUrl" type="url" required class="rounded-xl border border-white/10 bg-slate-950 px-4 py-3 text-white outline-none focus:border-emerald-300" placeholder="https://example.com"></label>
      <label class="grid gap-2 text-sm">预估每日请求量<input v-model.number="form.expectedDailyRequests" type="number" min="1" required class="rounded-xl border border-white/10 bg-slate-950 px-4 py-3 text-white outline-none focus:border-emerald-300"></label>
      <label class="grid gap-2 text-sm">使用场景<textarea v-model="form.usageScenario" required rows="5" class="rounded-xl border border-white/10 bg-slate-950 px-4 py-3 text-white outline-none focus:border-emerald-300" /></label>
      <label class="flex items-start gap-3 text-sm text-slate-300"><input v-model="form.agreeToTerms" type="checkbox" class="mt-1">我确认不会使用 API 还原用户、资源下载、评论或任何主站隐私数据。</label>
      <p v-if="message" class="text-sm" :class="ok ? 'text-emerald-300' : 'text-rose-300'">{{ message }}</p>
      <button class="rounded-xl bg-emerald-400 px-5 py-3 font-bold text-slate-950">提交账号申请</button>
    </form>
  </div>
</template>

<script setup lang="ts">
import type { ApplicationItem } from '~/composables/useDashboard'

const { apiFetch } = useApi()
const applications = ref<ApplicationItem[]>([])
const form = reactive({ applicantName: '', projectName: '', projectUrl: '', expectedDailyRequests: 1000, usageScenario: '', agreeToTerms: false })
const message = ref('')
const ok = ref(false)
const existingApplication = computed(() => applications.value[0])
const badge = (status: string) => status === 'approved' ? 'bg-emerald-400 text-slate-950' : status === 'pending' ? 'bg-amber-300 text-slate-950' : 'bg-rose-400 text-white'
const statusText = (status: string) => status === 'approved' ? '已通过' : status === 'pending' ? '审核中' : status === 'rejected' ? '未通过' : '已撤销'
const loadApplications = async () => {
  const res = await apiFetch<ApplicationItem[]>('/applications')
  if (res.success) applications.value = res.data
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

