export default defineNuxtConfig({
  compatibilityDate: '2026-06-07',
  modules: ['@pinia/nuxt', '@nuxt/ui'],
  css: ['~/assets/css/main.css'],
  app: {
    head: {
      link: [{ rel: 'icon', type: 'image/webp', href: '/logo.webp' }]
    }
  },
  runtimeConfig: {
    public: {
      apiBaseUrl: process.env.NUXT_PUBLIC_API_BASE_URL || 'http://localhost:8080',
      touchgalTechDocsUrl:
        process.env.NUXT_PUBLIC_TOUCHGAL_TECH_DOCS_URL || 'https://github.com/KUN1007/kun-touchgal-next',
      apiDocsUrl: process.env.NUXT_PUBLIC_API_DOCS_URL || '/docs/api',
      turnstileSiteKey: process.env.NUXT_PUBLIC_TURNSTILE_SITE_KEY || ''
    }
  },
  typescript: {
    typeCheck: true,
    strict: true
  }
})

