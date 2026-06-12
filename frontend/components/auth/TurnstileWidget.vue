<template>
  <div class="tg-turnstile">
    <div ref="container" class="tg-turnstile-widget" />
  </div>
</template>

<script setup lang="ts">
type TurnstileRenderOptions = {
  sitekey: string
  callback: (token: string) => void
  'expired-callback': () => void
  'error-callback': () => void
  'timeout-callback': () => void
}

type TurnstileAPI = {
  render: (container: HTMLElement, options: TurnstileRenderOptions) => string
  reset: (widgetId?: string) => void
  remove?: (widgetId: string) => void
}

declare global {
  interface Window {
    turnstile?: TurnstileAPI
    __touchgalTurnstileLoad?: Promise<void>
  }
}

const props = defineProps<{
  siteKey: string
  modelValue: string
  resetKey: number
}>()

const emit = defineEmits<{
  'update:modelValue': [token: string]
  error: [message: string]
}>()

const container = ref<HTMLElement | null>(null)
const widgetId = ref<string | null>(null)

const turnstileScriptSrc = 'https://challenges.cloudflare.com/turnstile/v0/api.js?render=explicit'
const expiredMessage = '人机验证已过期，请重新完成验证。'
const failedMessage = '人机验证失败，请重试。'

const loadTurnstile = () => {
  if (window.turnstile) {
    return Promise.resolve()
  }
  if (window.__touchgalTurnstileLoad) {
    return window.__touchgalTurnstileLoad
  }

  window.__touchgalTurnstileLoad = new Promise<void>((resolve, reject) => {
    const existingScript = document.querySelector<HTMLScriptElement>(`script[src="${turnstileScriptSrc}"]`)
    if (existingScript) {
      existingScript.addEventListener('load', () => resolve(), { once: true })
      existingScript.addEventListener('error', () => reject(new Error('Turnstile script failed to load')), { once: true })
      return
    }

    const script = document.createElement('script')
    script.src = turnstileScriptSrc
    script.async = true
    script.defer = true
    script.onload = () => resolve()
    script.onerror = () => reject(new Error('Turnstile script failed to load'))
    document.head.appendChild(script)
  }).catch((err) => {
    window.__touchgalTurnstileLoad = undefined
    throw err
  })

  return window.__touchgalTurnstileLoad
}

const fail = (message: string) => {
  emit('update:modelValue', '')
  emit('error', message)
}

const resetWidget = () => {
  emit('update:modelValue', '')
  if (!widgetId.value || !window.turnstile) {
    return
  }
  window.turnstile.reset(widgetId.value)
}

const removeWidget = () => {
  if (!widgetId.value || !window.turnstile?.remove) {
    widgetId.value = null
    return
  }
  window.turnstile.remove(widgetId.value)
  widgetId.value = null
}

const renderWidget = async () => {
  if (!container.value || !props.siteKey || widgetId.value) {
    return
  }

  try {
    await loadTurnstile()
    if (!window.turnstile || !container.value) {
      fail(failedMessage)
      return
    }
    widgetId.value = window.turnstile.render(container.value, {
      sitekey: props.siteKey,
      callback: (token: string) => {
        emit('update:modelValue', token)
        emit('error', '')
      },
      'expired-callback': () => fail(expiredMessage),
      'error-callback': () => fail(failedMessage),
      'timeout-callback': () => fail(failedMessage)
    })
  } catch {
    fail(failedMessage)
  }
}

watch(() => props.resetKey, () => {
  resetWidget()
})

watch(() => props.siteKey, () => {
  removeWidget()
  emit('update:modelValue', '')
  void renderWidget()
})

onMounted(() => {
  void renderWidget()
})

onBeforeUnmount(() => {
  removeWidget()
})
</script>
