<script setup lang="ts">
// 左下角地图式导航面板：方位罗盘（平移）+ 缩放 + 适应屏幕，类似在线地图控件
const emit = defineEmits<{
  (e: 'pan', dx: number, dy: number): void
  (e: 'zoom-in'): void
  (e: 'zoom-out'): void
  (e: 'fit'): void
}>()

defineProps<{ zoomPercent: number }>()
</script>

<template>
  <div class="nav-panel">
    <!-- 罗盘：四向平移 -->
    <div class="compass">
      <button class="nav-btn up" title="上移画布" @click="emit('pan', 0, -140)">▲</button>
      <button class="nav-btn left" title="左移画布" @click="emit('pan', -140, 0)">◀</button>
      <button class="nav-btn center" title="适应屏幕（显示全部节点）" @click="emit('fit')">⌖</button>
      <button class="nav-btn right" title="右移画布" @click="emit('pan', 140, 0)">▶</button>
      <button class="nav-btn down" title="下移画布" @click="emit('pan', 0, 140)">▼</button>
    </div>
    <!-- 缩放 -->
    <div class="zoomer">
      <button class="nav-btn" title="放大" @click="emit('zoom-in')">＋</button>
      <div class="zoom-val" title="当前缩放比例">{{ zoomPercent }}%</div>
      <button class="nav-btn" title="缩小" @click="emit('zoom-out')">－</button>
    </div>
  </div>
</template>

<style scoped>
.nav-panel {
  position: absolute;
  left: 14px;
  bottom: 16px;
  z-index: 20;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
}

.compass {
  display: grid;
  grid-template-columns: repeat(3, 34px);
  grid-template-rows: repeat(3, 30px);
  background: rgba(255, 255, 255, 0.95);
  border: 1px solid #e0e6ef;
  border-radius: 12px;
  box-shadow: 0 4px 14px rgba(30, 50, 90, 0.16);
  overflow: hidden;
}

.zoomer {
  width: 110px;
  display: flex;
  align-items: stretch;
  background: rgba(255, 255, 255, 0.95);
  border: 1px solid #e0e6ef;
  border-radius: 12px;
  box-shadow: 0 4px 14px rgba(30, 50, 90, 0.16);
  overflow: hidden;
}

.nav-btn {
  appearance: none;
  border: none;
  background: transparent;
  color: #3c4a63;
  font-size: 13px;
  cursor: pointer;
  padding: 0;
  transition:
    background 0.12s,
    color 0.12s;
}

.compass .up { grid-area: 1 / 2; }
.compass .left { grid-area: 2 / 1; }
.compass .center {
  grid-area: 2 / 2;
  color: #5b8def;
  font-size: 17px;
}
.compass .right { grid-area: 2 / 3; }
.compass .down { grid-area: 3 / 2; }

.compass .nav-btn:hover,
.zoomer .nav-btn:hover {
  background: #eef3fc;
  color: #2563eb;
}

.nav-btn:active {
  background: #dfe9fb;
}

.zoomer .nav-btn {
  width: 36px;
  font-size: 16px;
}

.zoom-val {
  flex: 1;
  min-width: 38px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 11px;
  font-weight: 600;
  color: #7a8699;
  border-left: 1px solid #edf0f5;
  border-right: 1px solid #edf0f5;
  user-select: none;
}
</style>
