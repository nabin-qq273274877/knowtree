<script setup lang="ts">
import { computed } from 'vue'
import { Handle, Position } from '@vue-flow/core'
import type { KNode } from '@/types'
import { STATUS_META } from '@/utils/meta'

// 自定义知识点卡片节点：状态色边框 + 标题 + 四向锚点（hover 浮现）
const props = defineProps<{
  data: {
    node: KNode
    selected: boolean
    pending: boolean // 点选连线模式：该节点是待连接的源
  }
}>()

const status = computed(() => STATUS_META[props.data.node.status])
</script>

<template>
  <div
    class="kt-node"
    :class="{ selected: data.selected, pending: data.pending }"
    :style="{ '--status-color': status.color }"
    :title="data.node.title"
  >
    <Handle type="source" :position="Position.Top" class="anchor" />
    <Handle type="source" :position="Position.Right" class="anchor" />
    <Handle type="source" :position="Position.Bottom" class="anchor" />
    <Handle type="source" :position="Position.Left" class="anchor" />

    <div class="kt-node__title">{{ data.node.title }}</div>
    <div class="kt-node__meta">
      <span class="kt-node__status" :style="{ background: status.color }">{{ status.label }}</span>
      <span v-if="data.node.stage" class="kt-node__stage">{{ data.node.stage }}</span>
    </div>
    <span v-if="data.node.annotation_count > 0" class="kt-node__badge" title="批注数">
      ✎ {{ data.node.annotation_count }}
    </span>
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
