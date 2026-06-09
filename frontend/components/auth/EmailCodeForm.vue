<template>
  <form class="tg-card tg-form" @submit.prevent="step === 'email' ? sendCode() : verifyCode()">
    <label v-if="mode === 'register'" class="tg-label">
      昵称
      <input v-model="displayName" required maxlength="80" class="tg-input">
    </label>

    <label class="tg-label">
      邮箱
      <input
        v-model="email"
        type="email"
        required
        class="tg-input"
      >
    </label>

    <label v-if="step === 'code'" class="tg-label">
      6 位验证码
      <input v-model="code" inputmode="numeric" maxlength="6" required class="tg-input" placeholder="123456">
    </label>

    <p v-if="message" :class="error ? 'tg-message-error' : 'tg-message-ok'">{{ message }}</p>

    <button class="tg-btn tg-btn-primary" type="submit" :disabled="loading">
      {{ loading ? '处理中...' : step === 'email' ? '发送验证码' : '完成验证' }}
    </button>
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
  } catch (err) {
    if (apiErrorCode(err) === 'ACCOUNT_DISABLED') {
      await redirectAccountDisabled()
      return
    }
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
    if (auth.isAccountDisabled) {
      await redirectAccountDisabled()
      return
    }
    emit('verified')
  } catch (err) {
    if (apiErrorCode(err) === 'ACCOUNT_DISABLED') {
      await redirectAccountDisabled()
      return
    }
    error.value = true
    message.value = '验证码错误或已过期。'
  } finally {
    loading.value = false
  }
}

const redirectAccountDisabled = async () => {
  auth.markAccountDisabled()
  await navigateTo('/account-disabled')
}

const apiErrorCode = (err: unknown) => {
  return (err as { data?: { error?: { code?: string } } }).data?.error?.code
}
</script>
