<template>
  <form class="grid gap-4 rounded-3xl border border-white/10 bg-white/[0.06] p-6" @submit.prevent="submit">
    <div class="grid gap-4 md:grid-cols-2">
      <label class="grid gap-2 text-sm">申请用户/负责人<input v-model="form.applicantName" required class="rounded-xl border border-white/10 bg-slate-950 px-4 py-3 text-white outline-none focus:border-emerald-300"></label>
      <label class="grid gap-2 text-sm">项目名称<input v-model="form.projectName" class="rounded-xl border border-white/10 bg-slate-950 px-4 py-3 text-white outline-none focus:border-emerald-300"></label>
    </div>
    <label class="grid gap-2 text-sm">项目地址<input v-model="form.projectUrl" type="url" required class="rounded-xl border border-white/10 bg-slate-950 px-4 py-3 text-white outline-none focus:border-emerald-300" placeholder="https://example.com"></label>
    <label class="grid gap-2 text-sm">预估每日请求量<input v-model.number="form.expectedDailyRequests" type="number" min="1" required class="rounded-xl border border-white/10 bg-slate-950 px-4 py-3 text-white outline-none focus:border-emerald-300"></label>
    <label class="grid gap-2 text-sm">使用场景<textarea v-model="form.usageScenario" required rows="5" class="rounded-xl border border-white/10 bg-slate-950 px-4 py-3 text-white outline-none focus:border-emerald-300" /></label>
    <label class="flex items-start gap-3 text-sm text-slate-300"><input v-model="form.agreeToTerms" type="checkbox" class="mt-1">我确认不会使用 API 还原用户、资源下载、评论或任何主站隐私数据。</label>
    <p v-if="message" class="text-sm" :class="ok ? 'text-emerald-300' : 'text-rose-300'">{{ message }}</p>
    <button class="rounded-xl bg-emerald-400 px-5 py-3 font-bold text-slate-950">提交申请</button>
  </form>
</template>

<script setup lang="ts">
const { apiFetch } = useApi()
const form = reactive({ applicantName: '', projectName: '', projectUrl: '', expectedDailyRequests: 1000, usageScenario: '', agreeToTerms: false })
const message = ref('')
const ok = ref(false)
const submit = async () => {
  try {
    await apiFetch('/applications', { method: 'POST', body: form })
    ok.value = true
    message.value = '申请已提交，等待管理员审核。'
  } catch {
    ok.value = false
    message.value = '提交失败，请检查 URL、请求量和必填项。'
  }
}
</script>

