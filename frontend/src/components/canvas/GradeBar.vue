<script lang="ts">
// 顶部分级彩条：视口内出现某学段的节点时该段彩条展开；
// 视野外的学段收缩成小圆点标记。点击任意段可聚焦到该学段的节点。
export interface GradeSegment {
  key: string
  label: string
  color: string
  /** 该学段节点总数 */
  count: number
  /** 当前视口内的节点数 */
  inView: number
}

function dominantKeyOf(segs: GradeSegment[]): string | null {
  let best: GradeSegment | null = null
  for (const s of segs) {
    if (s.inView > 0 && (!best || s.inView > best.inView)) best = s
  }
  return best?.key ?? null
}
</script>

<script setup lang="ts">
const props = defineProps<{ segments: GradeSegment[] }>()
defineEmits<{ (e: 'select', key: string): void }>()

const dominantKey = () => dominantKeyOf(props.segments)
</script>

<template>
  <div class="grade-bar">
    <button
      v-for="s in props.segments"
      :key="s.key"
      class="seg"
      :class="{ expanded: s.inView > 0, dominant: s.key === dominantKey(), empty: s.count === 0 && s.inView === 0 }"
      :style="{ '--seg-color': s.color }"
      :title="`${s.label} · 共 ${s.count} 个节点${s.inView ? `，视口内 ${s.inView}` : ''}`"
      @click="$emit('select', s.key)"
    >
      <span v-if="s.inView > 0" class="seg__label">{{ s.label }}</span>
      <span v-if="s.inView > 0 && s.inView !== s.count" class="seg__count">{{ s.inView }}/{{ s.count }}</span>
    </button>
  </div>
</template>

<style scoped>
.grade-bar {
  position: absolute;
  top: 12px;
  left: 14px;
  right: 66px;
  height: 26px;
  display: flex;
  gap: 6px;
  z-index: 20;
  pointer-events: none;
}

.seg {
  pointer-events: auto;
  appearance: none;
  border: none;
  cursor: pointer;
  padding: 0;
  height: 100%;
  min-width: 0;
  flex: 0 1 22px;
  border-radius: 13px;
  background: var(--seg-color);
  box-shadow:
    inset 0 0 0 1px rgba(255, 255, 255, 0.35),
    0 2px 6px rgba(30, 50, 90, 0.25);
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  overflow: hidden;
  white-space: nowrap;
  opacity: 0.85;
  transition:
    flex-grow 0.35s cubic-bezier(0.4, 0, 0.2, 1),
    flex-basis 0.35s cubic-bezier(0.4, 0, 0.2, 1),
    opacity 0.2s,
    box-shadow 0.2s,
    transform 0.15s;
}

.seg.empty {
  opacity: 0.28;
}

.seg:hover {
  transform: translateY(-2px);
  opacity: 1;
}

.seg.expanded {
  flex-grow: 1;
  min-width: 88px;
  max-width: 220px;
  opacity: 1;
  background:
    linear-gradient(180deg, rgba(255, 255, 255, 0.22), rgba(0, 0, 0, 0.06)),
    var(--seg-color);
}

.seg.dominant {
  box-shadow:
    inset 0 0 0 1px rgba(255, 255, 255, 0.5),
    0 0 0 2px rgba(255, 255, 255, 0.9),
    0 3px 10px rgba(30, 50, 90, 0.35);
}

.seg__label {
  color: #fff;
  font-size: 12px;
  font-weight: 700;
  letter-spacing: 0.5px;
  text-shadow: 0 1px 2px rgba(0, 0, 0, 0.25);
}

.seg__count {
  color: rgba(255, 255, 255, 0.92);
  font-size: 10.5px;
  background: rgba(0, 0, 0, 0.18);
  border-radius: 9px;
  padding: 2px 6px;
  line-height: 1;
}
</style>
