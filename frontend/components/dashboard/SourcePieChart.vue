<template><ClientOnly><VChart class="h-80 w-full" :option="option" autoresize /></ClientOnly></template>
<script setup lang="ts">
import { PieChart } from 'echarts/charts'
import { LegendComponent, TooltipComponent } from 'echarts/components'
import { use } from 'echarts/core'
import { CanvasRenderer } from 'echarts/renderers'
import type { EChartsOption } from 'echarts'
import VChart from 'vue-echarts'
import type { SourceItem } from '~/composables/useDashboard'
use([CanvasRenderer, PieChart, TooltipComponent, LegendComponent])
const props = defineProps<{ data: SourceItem[] }>()
const option = computed<EChartsOption>(() => ({
  tooltip: { trigger: 'item' },
  legend: { bottom: 0, textStyle: { color: '#cbd5e1' } },
  series: [{ type: 'pie', radius: ['45%', '70%'], data: props.data.map((i) => ({ name: i.origin || i.refererHost || 'unknown', value: i.requests })) }]
}))
</script>
