import type { NodeStatus, EdgeRelation } from '@/types'

// 学习状态视觉规范（画布卡片 / 树 / 详情面板共用）
export const STATUS_META: Record<
  NodeStatus,
  { label: string; color: string; elType: 'info' | 'primary' | 'warning' | 'success' | 'danger' }
> = {
  not_started: { label: '未学', color: '#b8c0cc', elType: 'info' },
  learning: { label: '学习中', color: '#409eff', elType: 'primary' },
  partial: { label: '部分学会', color: '#e6a23c', elType: 'warning' },
  mastered: { label: '已学会', color: '#67c23a', elType: 'success' },
  forgotten: { label: '已遗忘', color: '#f56c6c', elType: 'danger' },
}

// 连线类型视觉规范：层级线 / 前置依赖（橙虚线+箭头）/ 一般关联（蓝实线）
export const LINE_STYLE = {
  hierarchy: { stroke: '#9aa7bf', strokeWidth: 1.5 },
  prerequisite: { stroke: '#e8590c', strokeWidth: 2, strokeDasharray: '6 4' },
  related: { stroke: '#5b8def', strokeWidth: 2 },
} as const

export const RELATION_LABEL: Record<EdgeRelation, string> = {
  prerequisite: '前置依赖',
  related: '一般关联',
}
