<script setup lang="ts">
// 浮动智能边：渲染时实时计算两节点之间最靠近的一对边框锚点连线，
// 保证线始终贴着节点边框（无空隙），且方向符合两节点的相对位置。
import { computed } from 'vue'
import { BaseEdge, EdgeLabelRenderer, getSmoothStepPath, useVueFlow, type EdgeProps } from '@vue-flow/core'

const props = defineProps<EdgeProps>()

const { findNode } = useVueFlow()

interface Anchor {
  x: number
  y: number
}

function anchorsOf(nodeId: string): { c: Anchor; t: Anchor; r: Anchor; b: Anchor; l: Anchor } | null {
  const n = findNode(nodeId)
  if (!n) return null
  const p = n.computedPosition // 已展平的绝对坐标
  const w = n.dimensions?.width ?? 190
  const h = n.dimensions?.height ?? 64
  const cx = p.x + w / 2
  const cy = p.y + h / 2
  return {
    c: { x: cx, y: cy },
    t: { x: cx, y: p.y },
    r: { x: p.x + w, y: cy },
    b: { x: cx, y: p.y + h },
    l: { x: p.x, y: cy },
  }
}

const geometry = computed(() => {
  void props.sourceX // 依赖任一几何 prop，节点/位置变化时重算
  void props.targetY
  const s = anchorsOf(props.source)
  const t = anchorsOf(props.target)
  if (!s || !t) {
    // 节点尚未挂载时的兜底：退回 vue-flow 计算好的把手坐标
    return { sx: props.sourceX, sy: props.sourceY, tx: props.targetX, ty: props.targetY }
  }
  // 在四边锚点中选距离最近的一对（排除中心点）
  const sList: Anchor[] = [s.t, s.r, s.b, s.l]
  const tList: Anchor[] = [t.t, t.r, t.b, t.l]
  let best: { a: Anchor; b: Anchor; d: number } | null = null
  for (const a of sList) {
    for (const b of tList) {
      const d = (a.x - b.x) ** 2 + (a.y - b.y) ** 2
      if (!best || d < best.d) best = { a, b, d }
    }
  }
  if (!best) return { sx: props.sourceX, sy: props.sourceY, tx: props.targetX, ty: props.targetY }
  return { sx: best.a.x, sy: best.a.y, tx: best.b.x, ty: best.b.y }
})

const path = computed(() => {
  const g = geometry.value
  const [p] = getSmoothStepPath({
    sourceX: g.sx,
    sourceY: g.sy,
    targetX: g.tx,
    targetY: g.ty,
    borderRadius: 14,
  })
  return p
})

const labelPos = computed(() => {
  const g = geometry.value
  return { x: (g.sx + g.tx) / 2, y: (g.sy + g.ty) / 2 }
})
</script>

<template>
  <BaseEdge :id="props.id" :path="path" :marker-end="props.markerEnd" :style="props.style" />
  <EdgeLabelRenderer v-if="props.label">
    <span
      class="fe-label"
      :style="{ transform: `translate(-50%, -50%) translate(${labelPos.x}px, ${labelPos.y}px)` }"
    >{{ props.label }}</span>
  </EdgeLabelRenderer>
</template>

<style>
/* EdgeLabelRenderer 渲染在 svg 之外，需要全局样式 */
.fe-label {
  position: absolute;
  pointer-events: none;
  background: #fff;
  border: 1px solid #f3b98a;
  color: #e8590c;
  font-size: 11px;
  line-height: 1;
  padding: 3px 7px;
  border-radius: 9px;
}
</style>
