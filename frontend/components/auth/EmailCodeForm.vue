<template>
  <form class="tg-card tg-form" @submit.prevent="step === 'email' ? sendCode() : verifyCode()">
    <label v-if="mode === 'register'" class="tg-label">
      昵称
      <input v-model="displayName" required maxlength="80" class="tg-input" :disabled="step === 'code' || loading">
    </label>

    <label class="tg-label">
      邮箱
      <input
        v-model="email"
        type="email"
        required
        class="tg-input"
        :disabled="step === 'code' || loading"
      >
    </label>

    <div v-if="cooldownOnly" class="tg-card-outline tg-code-cooldown-card" role="status" aria-live="polite">
      <div class="tg-code-cooldown-mark" aria-hidden="true">!</div>
      <div class="tg-code-cooldown-content">
        <p class="tg-eyebrow tg-code-cooldown-eyebrow">Cooldown</p>
        <p class="tg-title-sm">{{ message }}</p>
      </div>
      <button class="tg-btn tg-btn-secondary tg-code-cooldown-cancel" type="button" :disabled="loading" @click="editEmail">取消</button>
    </div>

    <div v-else-if="step === 'code'" class="tg-card-outline tg-form">
      <div>
        <p class="tg-eyebrow">Email Verification</p>
        <p class="tg-title-sm">验证码已发送至 {{ email }}</p>
        <p class="tg-field-note mt-2">{{ codeTimingText }}</p>
      </div>

      <label class="tg-label">
        验证码
        <input v-model="code" inputmode="numeric" pattern="[0-9]*" maxlength="6" required class="tg-input" autocomplete="one-time-code">
      </label>

      <div class="tg-actions">
        <button class="tg-btn tg-btn-secondary tg-code-resend-btn" type="button" :disabled="loading || resendRemaining > 0" @click="sendCode">
          {{ resendButtonText }}
        </button>
        <button class="tg-link tg-link-button" type="button" :disabled="loading" @click="editEmail">修改邮箱</button>
      </div>
    </div>

    <p v-if="message && !cooldownOnly" :class="error ? 'tg-message-error' : 'tg-message-ok'" aria-live="polite">{{ message }}</p>

    <button v-if="!cooldownOnly" class="tg-btn tg-btn-primary" type="submit" :disabled="loading">
      {{ loading ? '处理中...' : step === 'email' ? '发送验证码' : '完成验证' }}
    </button>
  </form>
</template>

<script setup lang="ts">
type CodeStartData = {
  sent: boolean
  expiresInSeconds?: number
  resendCooldownSeconds?: number
}

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
const cooldownOnly = ref(false)
const now = ref(Date.now())
const codeExpiresAt = ref<number | null>(null)
const resendAvailableAt = ref<number | null>(null)

const fallbackCodeTTLSeconds = 10 * 60
const fallbackResendCooldownSeconds = 60

let timer: ReturnType<typeof setInterval> | null = null

const secondsUntil = (timestamp: number | null) => {
  if (!timestamp) {
    return 0
  }
  return Math.max(0, Math.ceil((timestamp - now.value) / 1000))
}

const codeRemaining = computed(() => secondsUntil(codeExpiresAt.value))
const resendRemaining = computed(() => secondsUntil(resendAvailableAt.value))

const codeTimingText = computed(() => {
  if (codeExpiresAt.value && codeRemaining.value <= 0) {
    return '验证码已过期，请重新发送后输入新验证码。'
  }
  if (codeExpiresAt.value) {
    return '验证码十分钟内有效。'
  }
  return `验证码通常 ${Math.round(fallbackCodeTTLSeconds / 60)} 分钟内有效；未收到邮件可重新发送。`
})

const resendButtonText = computed(() => {
  if (resendRemaining.value > 0) {
    return `重新发送 (${resendRemaining.value}s)`
  }
  return '重新发送验证码'
})

const startTimer = () => {
  if (timer) {
    return
  }
  timer = setInterval(() => {
    now.value = Date.now()
    if (codeRemaining.value === 0 && resendRemaining.value === 0) {
      stopTimer()
    }
  }, 1000)
}

const stopTimer = () => {
  if (!timer) {
    return
  }
  clearInterval(timer)
  timer = null
}

const positiveSeconds = (value: number | undefined, fallback: number) => {
  if (!value || value <= 0) {
    return fallback
  }
  return value
}

const applyCodeTiming = (data?: CodeStartData) => {
  const current = Date.now()
  now.value = current
  codeExpiresAt.value = current + positiveSeconds(data?.expiresInSeconds, fallbackCodeTTLSeconds) * 1000
  resendAvailableAt.value = current + positiveSeconds(data?.resendCooldownSeconds, fallbackResendCooldownSeconds) * 1000
  startTimer()
}

const applyCooldownOnly = () => {
  const current = Date.now()
  now.value = current
  resendAvailableAt.value = current + fallbackResendCooldownSeconds * 1000
  if (!codeExpiresAt.value) {
    codeExpiresAt.value = current + fallbackCodeTTLSeconds * 1000
  }
  startTimer()
}

const formatDuration = (totalSeconds: number) => {
  const minutes = Math.floor(totalSeconds / 60)
  const seconds = totalSeconds % 60
  if (minutes === 0) {
    return `${seconds} 秒`
  }
  return `${minutes} 分 ${seconds.toString().padStart(2, '0')} 秒`
}

const sendCode = async () => {
  loading.value = true
  message.value = ''
  error.value = false
  try {
    const path = props.mode === 'register' ? '/auth/register/start' : '/auth/login/start'
    const body = props.mode === 'register' ? { email: email.value, displayName: displayName.value } : { email: email.value }
    const response = await apiFetch<CodeStartData>(path, { method: 'POST', body })
    code.value = ''
    step.value = 'code'
    cooldownOnly.value = false
    applyCodeTiming(response.success ? response.data : undefined)
  } catch (err) {
    if (apiErrorCode(err) === 'ACCOUNT_DISABLED') {
      await redirectAccountDisabled()
      return
    }
    if (apiErrorCode(err) === 'CODE_COOLDOWN' || apiErrorCode(err) === 'RATE_LIMITED') {
      step.value = 'code'
      applyCooldownOnly()
      cooldownOnly.value = true
      error.value = true
      message.value = '请求过于频繁，请等待 1 分钟后重新发送。'
      return
    }
    cooldownOnly.value = false
    error.value = true
    message.value = apiErrorMessage(err) || '发送失败，请检查邮箱或稍后重试。'
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
    await auth.fetchMe({ refresh: true })
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
    const codeError = apiErrorCode(err)
    if (codeError === 'EXPIRED_CODE' || codeRemaining.value === 0) {
      message.value = '验证码已过期，请点击“重新发送验证码”获取新验证码。'
      return
    }
    if (codeError === 'INVALID_CODE' || codeError === 'BAD_REQUEST') {
      message.value = '验证码不正确，请核对 6 位数字；未收到邮件可在倒计时结束后重发。'
      return
    }
    message.value = apiErrorMessage(err) || '验证失败，请稍后重试。'
  } finally {
    loading.value = false
  }
}

const editEmail = () => {
  step.value = 'email'
  code.value = ''
  message.value = ''
  cooldownOnly.value = false
  error.value = false
  codeExpiresAt.value = null
  resendAvailableAt.value = null
  stopTimer()
}

const redirectAccountDisabled = async () => {
  auth.markAccountDisabled()
  await navigateTo('/account-disabled')
}

const apiErrorCode = (err: unknown) => {
  return (err as { data?: { error?: { code?: string } } }).data?.error?.code
}

const apiErrorMessage = (err: unknown) => {
  return (err as { data?: { error?: { message?: string } } }).data?.error?.message
}

onUnmounted(stopTimer)
</script>
