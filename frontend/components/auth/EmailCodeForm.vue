<template>
  <form class="grid gap-4 rounded-3xl border border-white/10 bg-white/[0.06] p-6" @submit.prevent="step === 'email' ? sendCode() : verifyCode()">
    <label class="grid gap-2 text-sm">
      邮箱
      <input v-model="email" type="email" required class="rounded-xl border border-white/10 bg-slate-950 px-4 py-3 text-white" placeholder="user@example.com">
    </label>
    <label v-if="mode === 'register'" class="grid gap-2 text-sm">
      昵称
      <input v-model="displayName" required maxlength="80" class="rounded-xl border border-white/10 bg-slate-950 px-4 py-3 text-white" placeholder="Kun">
    </label>
    <label v-if="step === 'code'" class="grid gap-2 text-sm">
      6 位验证码
      <input v-model="code" inputmode="numeric" maxlength="6" required class="rounded-xl border border-white/10 bg-slate-950 px-4 py-3 text-white" placeholder="123456">
    </label>
    <p v-if="message" class="text-sm" :class="error ? 'text-rose-300' : 'text-emerald-300'">{{ message }}</p>
    <button class="rounded-xl bg-emerald-400 px-4 py-3 font-bold text-slate-950" :disabled="loading">{{ loading ? '处理中...' : step === 'email' ? '发送验证码' : '完成验证' }}</button>
  </form>
</template>

<script setup lang="ts">
const props = defineProps<{ mode: 'login' | 'register' }>()
const emit = defineEmits<{ verified: [] }>()
const { apiFetch } = useApi()
const auth = useAuthStore()
const email = ref('')
const displayName = ref('')
const code = ref('')
const step = ref<'email' | 'code'>('email')
const loading = ref(false)
const message = ref('')
const error = ref(false)

const sendCode = async () => {
  loading.value = true
  message.value = ''
  error.value = false
  try {
    const path = props.mode === 'register' ? '/auth/register/start' : '/auth/login/start'
    const body = props.mode === 'register' ? { email: email.value, displayName: displayName.value } : { email: email.value }
    await apiFetch(path, { method: 'POST', body })
    step.value = 'code'
    message.value = '验证码已发送，请检查邮箱。'
  } catch {
    error.value = true
    message.value = '发送失败，请检查邮箱或稍后重试。'
  } finally {
    loading.value = false
  }
}

const verifyCode = async () => {
  loading.value = true
  message.value = ''
  error.value = false
  try {
    const path = props.mode === 'register' ? '/auth/register/verify' : '/auth/login/verify'
    await apiFetch(path, { method: 'POST', body: { email: email.value, code: code.value } })
    await auth.fetchMe()
    emit('verified')
  } catch {
    error.value = true
    message.value = '验证码错误或已过期。'
  } finally {
    loading.value = false
  }
}
</script>
