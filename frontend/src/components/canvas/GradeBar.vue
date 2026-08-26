<script lang="ts">
// 顶部分级彩条：视口进入某学段分区时该段彩条展开；
// 视野外的学段收缩成小圆点标记。当前选中节点所属学段始终展开并加白圈。
// 点击任意段可聚焦到该学段的节点。
export interface GradeSegment {
  key: string
  label: string
  color: string
  /** 该学段节点总数 */
  count: number
  /** 学段分区是否与当前视口相交 */
  inView: boolean
}

function dominantKeyOf(segs: GradeSegment[]): string | null {
  let best: GradeSegment | null = null
  for (const s of segs) {
    if (s.count > 0 && (!best || s.count > best.count)) best = s
  }
  return best?.key ?? null
}

export default {}
</script>

<script setup lang="ts">
const props = defineProps<{ segments: GradeSegment[]; activeKey?: string | null }>()
defineEmits<{ (e: 'select', key: string): void }>()

const dominantKey = () => dominantKeyOf(props.segments)

function isActive(key: string) {
  return !!props.activeKey && props.activeKey === key
}
</script>

<template>
  <div class="grade-bar">
    <button
      v-for="s in props.segments"
      :key="s.key"
      class="seg"
      :class="{
        expanded: s.inView || isActive(s.key),
        active: isActive(s.key),
        dominant: !props.activeKey && s.key === dominantKey(),
        empty: s.count === 0 && !isActive(s.key),
      }"
      :style="{ '--seg-color': s.color }"
      :title="`${s.label} · 共 ${s.count} 个知识点${isActive(s.key) ? ' · 当前选中' : ''}`"
      @click="$emit('select', s.key)"
    >
      <span v-if="s.inView || isActive(s.key)" class="seg__label">{{ s.label }}</span>
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
  max-width: 240px;
  opacity: 1;
  background:
    linear-gradient(180deg, rgba(255, 255, 255, 0.22), rgba(0, 0, 0, 0.06)),
    var(--seg-color);
}

.seg.active {
  box-shadow:
    inset 0 0 0 1px rgba(255, 255, 255, 0.5),
    0 0 0 2.5px rgba(255, 255, 255, 0.95),
    0 3px 12px rgba(30, 50, 90, 0.4);
  transform: translateY(-1px);
}

.seg.dominant {
  box-shadow:
    inset 0 0 0 1px rgba(255, 255, 255, 0.5),
    0 0 0 2px rgba(255, 255, 255, 0.75),
    0 3px 10px rgba(30, 50, 90, 0.35);
}

.seg__label {
  color: #fff;
  font-size: 12px;
  font-weight: 700;
  letter-spacing: 0.5px;
  text-shadow: 0 1px 2px rgba(0, 0, 0, 0.25);
}
</style>
