<script setup lang="ts">
// 左下角地图式导航面板：圆形罗盘（平移）+ 半透明缩放条
// 中心按钮：单击 = 定位到当前节点；双击 = 全览所有节点
const emit = defineEmits<{
  (e: 'pan', dx: number, dy: number): void
  (e: 'zoom-in'): void
  (e: 'zoom-out'): void
  (e: 'fit'): void
  (e: 'fit-all'): void
}>()

defineProps<{ zoomPercent: number }>()
</script>

<template>
  <div class="nav-panel">
    <!-- 圆形罗盘：四向平移 + 中心定位 -->
    <div class="compass">
      <button class="dir n" title="上移画布" @click="emit('pan', 0, -140)">▲</button>
      <button class="dir w" title="左移画布" @click="emit('pan', -140, 0)">◀</button>
      <button class="dir center" title="定位当前节点（双击：全览全部）" @click="emit('fit')" @dblclick="emit('fit-all')">⌖</button>
      <button class="dir e" title="右移画布" @click="emit('pan', 140, 0)">▶</button>
      <button class="dir s" title="下移画布" @click="emit('pan', 0, 140)">▼</button>
    </div>
    <!-- 缩放 -->
    <div class="zoomer">
      <button class="zbtn" title="放大" @click="emit('zoom-in')">＋</button>
      <div class="zoom-val" title="当前缩放比例">{{ zoomPercent }}%</div>
      <button class="zbtn" title="缩小" @click="emit('zoom-out')">－</button>
    </div>
  </div>
</template>

<style scoped>
.nav-panel {
  position: absolute;
  left: 16px;
  bottom: 18px;
  z-index: 20;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 10px;
}

/* ---------- 圆形罗盘 ---------- */
.compass {
  position: relative;
  width: 96px;
  height: 96px;
  border-radius: 50%;
  background:
    radial-gradient(circle at 50% 42%, rgba(255, 255, 255, 0.5), rgba(255, 255, 255, 0.28));
  backdrop-filter: blur(6px);
  border: 1px solid rgba(120, 140, 170, 0.35);
  box-shadow: 0 4px 16px rgba(30, 50, 90, 0.18);
}

.dir {
  position: absolute;
  appearance: none;
  border: none;
  background: transparent;
  color: #46566f;
  cursor: pointer;
  padding: 0;
  transition:
    color 0.12s,
    transform 0.12s,
    background 0.12s;
}

.dir:hover {
  color: #2563eb;
  transform: scale(1.15);
}

.dir.n,
.dir.s {
  left: 50%;
  width: 30px;
  height: 24px;
  margin-left: -15px;
  border-radius: 8px;
}

.dir.n { top: 3px; }
.dir.s { bottom: 3px; }

.dir.w,
.dir.e {
  top: 50%;
  height: 30px;
  width: 24px;
  margin-top: -15px;
  border-radius: 8px;
}

.dir.w { left: 4px; }
.dir.e { right: 4px; }

.dir.center {
  left: 50%;
  top: 50%;
  transform: translate(-50%, -50%);
  width: 38px;
  height: 38px;
  border-radius: 50%;
  background: rgba(255, 255, 255, 0.75);
  border: 1px solid rgba(120, 140, 170, 0.45);
  color: #2f6bff;
  font-size: 18px;
  box-shadow: 0 1px 6px rgba(30, 50, 90, 0.22);
}

.dir.center:hover {
  transform: translate(-50%, -50%) scale(1.06);
  color: #1d4ed8;
}

/* 刻度装饰 */
.compass::after {
  content: '';
  position: absolute;
  inset: 10px;
  border-radius: 50%;
  border: 1px dashed rgba(120, 140, 170, 0.28);
  pointer-events: none;
}

/* ---------- 缩放条（半透明） ---------- */
.zoomer {
  display: flex;
  align-items: stretch;
  min-width: 104px;
  border-radius: 999px;
  overflow: hidden;
  background: rgba(255, 255, 255, 0.55);
  backdrop-filter: blur(6px);
  border: 1px solid rgba(120, 140, 170, 0.35);
  box-shadow: 0 3px 12px rgba(30, 50, 90, 0.14);
}

.zbtn {
  appearance: none;
  border: none;
  background: transparent;
  color: #46566f;
  font-size: 15px;
  width: 34px;
  cursor: pointer;
  padding: 7px 0;
  transition:
    background 0.12s,
    color 0.12s;
}

.zbtn:hover {
  background: rgba(37, 99, 235, 0.12);
  color: #2563eb;
}

.zoom-val {
  flex: 1;
  min-width: 40px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 11px;
  font-weight: 600;
  color: #5c6b82;
  user-select: none;
}
</style>
