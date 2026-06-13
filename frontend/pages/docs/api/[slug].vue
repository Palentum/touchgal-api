<template>
  <DocsApiEndpointDoc :doc="doc" />
</template>

<script setup lang="ts">
import { getApiEndpointDoc } from '~/composables/apiDocs'

const route = useRoute()
const rawSlug = route.params.slug
const slug = Array.isArray(rawSlug) ? rawSlug[0] : String(rawSlug)
const doc = getApiEndpointDoc(slug)

if (!doc) {
  throw createError({ statusCode: 404, statusMessage: 'API document not found' })
}

definePageMeta({ layout: 'docs' })
useHead({ title: `${doc.name} - TouchGal API 文档` })
</script>
