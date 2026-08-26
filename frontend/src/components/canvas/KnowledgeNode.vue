<script setup lang="ts">
// 自定义知识点卡片节点：状态色边框 + 标题 + 四向锚点（hover 浮现）
// hover 底部操作条：增加下级 / 增加同级 / 删除
import { computed } from 'vue'
import { Handle, Position } from '@vue-flow/core'
import type { KNode } from '@/types'
import { STATUS_META } from '@/utils/meta'

const props = defineProps<{
  data: {
    node: KNode
    selected: boolean
    pending: boolean // 点选连线模式：该节点是待连接的源
    stageColor?: string // 学段彩条同款颜色，未匹配时为灰色
  }
}>()

const emit = defineEmits<{
  (e: 'add-child'): void
  (e: 'add-sibling'): void
  (e: 'remove'): void
}>()

const status = computed(() => STATUS_META[props.data.node.status])

function stop(ev: Event) {
  ev.stopPropagation()
}
</script>

<template>
  <div
    class="kt-node"
    :class="{ selected: data.selected, pending: data.pending }"
    :style="{ '--status-color': status.color }"
    :title="data.node.title"
  >
    <Handle id="t" type="source" :position="Position.Top" class="anchor" />
    <Handle id="r" type="source" :position="Position.Right" class="anchor" />
    <Handle id="b" type="source" :position="Position.Bottom" class="anchor" />
    <Handle id="l" type="source" :position="Position.Left" class="anchor" />

    <div class="kt-node__title">{{ data.node.title }}</div>
    <div class="kt-node__meta">
      <span class="kt-node__status" :style="{ background: status.color }">{{ status.label }}</span>
      <span v-if="data.node.stage" class="kt-node__stage">
        <i class="stage-dot" :style="{ background: data.stageColor ?? '#cbd5e1' }" />{{ data.node.stage }}
      </span>
    </div>
    <span v-if="data.node.annotation_count > 0" class="kt-node__badge" title="批注数">
      ✎ {{ data.node.annotation_count }}
    </span>

    <!-- hover 操作条 -->
    <div class="kt-node__actions" @mousedown.stop @mouseup.stop @click.stop @dblclick.stop>
      <button title="增加下一级" @click.stop="emit('add-child')"><span class="a-ico">＋</span>下级</button>
      <button title="增加同级" @click.stop="emit('add-sibling')"><span class="a-ico">＋</span>同级</button>
      <button class="danger" title="删除节点（含子树）" @click.stop="emit('remove')">🗑</button>
    </div>
  </div>
</template>

<style scoped>
.kt-node {
  --status-color: #b8c0cc;
  position: relative;
  min-width: 120px;
  max-width: 220px;
  padding: 10px 14px;
  background: #fff;
  border: 2px solid var(--status-color);
  border-radius: 10px;
  box-shadow: 0 2px 6px rgba(30, 50, 90, 0.12);
  cursor: pointer;
  transition:
    border-color 0.12s,
    box-shadow 0.12s;
}

.kt-node__badge {
  position: absolute;
  top: -9px;
  right: -9px;
  background: #fff7ed;
  color: #e8590c;
  border: 1px solid #fdba74;
  font-size: 10px;
  line-height: 1;
  padding: 3px 6px;
  border-radius: 8px;
}

.kt-node:hover {
  box-shadow: 0 4px 12px rgba(30, 50, 90, 0.22);
}

.kt-node.selected,
.kt-node.pending {
  border-color: #ff7043;
  box-shadow: 0 0 0 3px rgba(255, 112, 67, 0.28);
}

.kt-node__title {
  font-size: 13px;
  font-weight: 600;
  color: #26334d;
  text-align: center;
  word-break: break-all;
  line-height: 1.4;
}

.kt-node__meta {
  margin-top: 6px;
  display: flex;
  justify-content: center;
  align-items: center;
  gap: 6px;
}

.kt-node__status {
  color: #fff;
  font-size: 11px;
  line-height: 1;
  padding: 3px 7px;
  border-radius: 9px;
}

.kt-node__stage {
  font-size: 11px;
  color: #8a94a6;
  border: 1px solid #d5dbe5;
  border-radius: 8px;
  padding: 2px 6px;
  line-height: 1;
  display: inline-flex;
  align-items: center;
  gap: 4px;
}

.stage-dot {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  display: inline-block;
}

/* ---------- hover 操作条 ---------- */
.kt-node__actions {
  position: absolute;
  left: 50%;
  bottom: -34px;
  transform: translateX(-50%) translateY(4px);
  display: flex;
  gap: 1px;
  background: rgba(23, 30, 44, 0.92);
  border-radius: 9px;
  padding: 2px;
  box-shadow: 0 4px 12px rgba(15, 25, 45, 0.35);
  opacity: 0;
  visibility: hidden;
  transition:
    opacity 0.15s,
    transform 0.15s,
    visibility 0.15s;
  z-index: 10;
}

.kt-node:hover .kt-node__actions,
.kt-node.selected .kt-node__actions {
  opacity: 1;
  visibility: visible;
  transform: translateX(-50%) translateY(0);
}

.kt-node__actions button {
  appearance: none;
  border: none;
  background: transparent;
  color: #cdd7e8;
  font-size: 11px;
  line-height: 1;
  padding: 6px 8px;
  border-radius: 7px;
  cursor: pointer;
  white-space: nowrap;
  display: inline-flex;
  align-items: center;
  gap: 3px;
  transition:
    background 0.12s,
    color 0.12s;
}

.kt-node__actions button:hover {
  background: rgba(255, 255, 255, 0.14);
  color: #fff;
}

.kt-node__actions button.danger:hover {
  background: rgba(245, 108, 108, 0.25);
  color: #ff9a9a;
}

.a-ico {
  font-weight: 700;
}

/* 四向锚点：hover 节点时浮现 */
:deep(.vue-flow__handle.anchor) {
  width: 11px;
  height: 11px;
  background: #7c95c4;
  border: 2px solid #fff;
  opacity: 0;
  transition: opacity 0.12s;
}

.kt-node:hover :deep(.vue-flow__handle.anchor),
.kt-node.pending :deep(.vue-flow__handle.anchor) {
  opacity: 1;
}
</style>
