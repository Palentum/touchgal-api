<template>
  <ClientOnly>
    <div class="h-80 min-h-0 w-full overflow-hidden">
      <VChart class="h-full min-h-0 w-full overflow-hidden" :option="option" autoresize />
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
const option = computed<EChartsOption>(() => ({
  tooltip: { trigger: 'item' },
  legend: { bottom: 0, textStyle: { color: '#cbd5e1' } },
  series: [{ type: 'pie', radius: ['45%', '70%'], data: props.data.map((i) => ({ name: i.origin || i.refererHost || 'unknown', value: i.requests })) }]
}))
</script>
