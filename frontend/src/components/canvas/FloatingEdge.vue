<script setup lang="ts">
// 浮动智能边：
// 1. 渲染时实时计算两节点边框锚点，线贴着节点边框（无空隙）
// 2. 若连线创建时记录了用户拖拽的把手方向（data.sh / data.th = t/r/b/l），
//    优先从那个方向出线，符合「从上面连就从上面出」的直觉
// 3. 贝塞尔曲线连接，带自然弧度；选中时高亮
import { computed } from 'vue'
import { BaseEdge, EdgeLabelRenderer, getBezierPath, Position, useVueFlow, type EdgeProps } from '@vue-flow/core'

const props = defineProps<EdgeProps>()

const { findNode } = useVueFlow()

type Side = 't' | 'r' | 'b' | 'l'

interface Anchor {
  x: number
  y: number
}

function nodeBox(nodeId: string): { x: number; y: number; w: number; h: number } | null {
  const n = findNode(nodeId)
  if (!n) return null
  const p = n.computedPosition // 已展平的绝对坐标
  const w = n.dimensions?.width ?? 190
  const h = n.dimensions?.height ?? 64
  return { x: p.x, y: p.y, w, h }
}

function anchorsOf(box: { x: number; y: number; w: number; h: number }): Record<Side, Anchor> {
  const cx = box.x + box.w / 2
  const cy = box.y + box.h / 2
  return {
    t: { x: cx, y: box.y },
    r: { x: box.x + box.w, y: cy },
    b: { x: cx, y: box.y + box.h },
    l: { x: box.x, y: cy },
  }
}

function nearest(list: Anchor[], to: Anchor): Anchor {
  let best = list[0]
  let bd = Infinity
  for (const a of list) {
    const d = (a.x - to.x) ** 2 + (a.y - to.y) ** 2
    if (d < bd) {
      bd = d
      best = a
    }
  }
  return best
}

const geometry = computed(() => {
  void props.sourceX // 依赖几何 props，位置/尺寸变化时重算
  void props.targetY
  const sBox = nodeBox(props.source)
  const tBox = nodeBox(props.target)
  if (!sBox || !tBox) {
    return { sx: props.sourceX, sy: props.sourceY, tx: props.targetX, ty: props.targetY }
  }
  const sA = anchorsOf(sBox)
  const tA = anchorsOf(tBox)
  const sList = [sA.t, sA.r, sA.b, sA.l]
  const tList = [tA.t, tA.r, tA.b, tA.l]

  // 用户拖线时选定的把手方向优先
  const sh = (props.data?.sh as Side | undefined) ?? null
  const th = (props.data?.th as Side | undefined) ?? null

  let sa: Anchor | null = sh && sA[sh] ? sA[sh] : null
  let ta: Anchor | null = th && tA[th] ? tA[th] : null
  let saSide: Side | null = sh ?? null
  let taSide: Side | null = th ?? null

  if (!sa && ta) {
    sa = nearest(sList, ta)
    saSide = sideOf(sa, sA)
  }
  if (!ta && sa) {
    ta = nearest(tList, sa)
    taSide = sideOf(ta, tA)
  }
  if (!sa || !ta) {
    // 双方都未指定：取整体最近的一对
    let best: { a: Anchor; b: Anchor; d: number } | null = null
    for (const a of sList) {
      for (const b of tList) {
        const d = (a.x - b.x) ** 2 + (a.y - b.y) ** 2
        if (!best || d < best.d) best = { a, b, d }
      }
    }
    if (best) {
      sa = best.a
      ta = best.b
      saSide = sideOf(sa, sA)
      taSide = sideOf(ta, tA)
    }
  }
  if (!sa || !ta) {
    return { sx: props.sourceX, sy: props.sourceY, tx: props.targetX, ty: props.targetY }
  }
  return { sx: sa.x, sy: sa.y, tx: ta.x, ty: ta.y, ss: saSide, ts: taSide }
})

function sideOf(a: Anchor, set: Record<Side, Anchor>): Side {
  if (set.t === a) return 't'
  if (set.r === a) return 'r'
  if (set.b === a) return 'b'
  return 'l'
}

const SIDE_TO_POS = {
  t: Position.Top,
  r: Position.Right,
  b: Position.Bottom,
  l: Position.Left,
} as const

const path = computed(() => {
  const g = geometry.value
  const [p] = getBezierPath({
    sourceX: g.sx,
    sourceY: g.sy,
    targetX: g.tx,
    targetY: g.ty,
    sourcePosition: SIDE_TO_POS[(g.ss ?? 'r') as Side],
    targetPosition: SIDE_TO_POS[(g.ts ?? 'l') as Side],
    curvature: 0.4,
  })
  return p
})

const edgeStyle = computed(() => {
  const base = (props.style ?? {}) as Record<string, string | number>
  if (!props.selected) return base
  return { ...base, stroke: '#ff7043', strokeWidth: 3 }
})

const labelPos = computed(() => {
  const g = geometry.value
  return { x: (g.sx + g.tx) / 2, y: (g.sy + g.ty) / 2 }
})
</script>

<template>
  <BaseEdge :id="props.id" :path="path" :marker-end="props.markerEnd" :style="edgeStyle" />
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
