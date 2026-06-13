<template>
  <ClientOnly>
    <div class="tg-chart-box">
      <VChart class="h-full min-h-[260px] w-full md:min-h-[320px]" :option="option" autoresize />
    </div>
  </ClientOnly>
</template>
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
const chartColors = ['#cc785c', '#5db8a6', '#e8a55a', '#141413', '#efe9de', '#c64545']

const option = computed<EChartsOption>(() => ({
  color: chartColors,
  tooltip: {
    trigger: 'item',
    backgroundColor: '#181715',
    borderColor: '#252320',
    textStyle: { color: '#faf9f5' }
  },
  legend: { bottom: 0, textStyle: { color: '#6c6a64' } },
  series: [
    {
      type: 'pie',
      radius: ['45%', '70%'],
      data: props.data.map((i) => ({ name: i.origin || i.refererHost || 'unknown', value: i.requests }))
    }
  ]
}))
</script>
