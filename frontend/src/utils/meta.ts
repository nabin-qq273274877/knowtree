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

// ---------- 学段 / 年级（画布顶部分级彩条用） ----------
// 覆盖幼儿园到大学：每个年级一个专属颜色，节点按 stage 字段模糊匹配归段。
export interface GradeDef {
  key: string
  label: string
  color: string
  aliases: string[]
}

export const GRADES: GradeDef[] = [
  { key: 'kindergarten', label: '幼儿园', color: '#ec4899', aliases: ['幼儿园', '学前'] },
  { key: 'g1', label: '一年级', color: '#ef4444', aliases: ['一年级', '1年级'] },
  { key: 'g2', label: '二年级', color: '#f97316', aliases: ['二年级', '2年级'] },
  { key: 'g3', label: '三年级', color: '#f59e0b', aliases: ['三年级', '3年级'] },
  { key: 'g4', label: '四年级', color: '#84cc16', aliases: ['四年级', '4年级'] },
  { key: 'g5', label: '五年级', color: '#22c55e', aliases: ['五年级', '5年级'] },
  { key: 'g6', label: '六年级', color: '#14b8a6', aliases: ['六年级', '6年级'] },
  { key: 'j1', label: '初一', color: '#06b6d4', aliases: ['初一', '七年级', '7年级'] },
  { key: 'j2', label: '初二', color: '#3b82f6', aliases: ['初二', '八年级', '8年级'] },
  { key: 'j3', label: '初三', color: '#6366f1', aliases: ['初三', '九年级', '9年级'] },
  { key: 's1', label: '高一', color: '#8b5cf6', aliases: ['高一', '十年级', '10年级'] },
  { key: 's2', label: '高二', color: '#a855f7', aliases: ['高二', '十一年级', '11年级'] },
  { key: 's3', label: '高三', color: '#d946ef', aliases: ['高三', '十二年级', '12年级'] },
  { key: 'university', label: '大学', color: '#64748b', aliases: ['大学', '本科', '高校', '学院', '大一', '大二', '大三', '大四', '研究生'] },
]

// 未匹配到任何学段的节点统一归入灰色「未设置」段
export const UNSET_GRADE = { key: '__unset__', label: '未设置', color: '#98a2b3' }

const CN_NUM: Record<string, number> = {
  一: 1, 二: 2, 三: 3, 四: 4, 五: 5, 六: 6, 七: 7, 八: 8, 九: 9, 十: 10,
}

function gradeByIndex(i: number): GradeDef | null {
  if (i < 1 || i > 12) return null
  return GRADES[i] // GRADES[0]=幼儿园，1..12 对应年级
}

function matchAlias(stage: string): GradeDef | null {
  for (const g of GRADES) {
    for (const a of g.aliases) {
      if (stage.includes(a)) return g
    }
  }
  return null
}

function matchCnYear(stage: string): GradeDef | null {
  // 处理「十一/十二年级」等中文数字写法
  const m = stage.match(/([一二三四五六七八九十]{1,2})\s*年级/)
  if (!m) return null
  const word = m[1]
  let n = 0
  if (word === '十') n = 10
  else if (word === '十一') n = 11
  else if (word === '十二') n = 12
  else if (word.startsWith('十')) n = 10 + (CN_NUM[word[1]] ?? 0)
  else if (word.endsWith('十')) n = (CN_NUM[word[0]] ?? 0) * 10
  else n = CN_NUM[word] ?? 0
  return gradeByIndex(n)
}

function matchNumYear(stage: string): GradeDef | null {
  const m = stage.match(/(\d{1,2})\s*年级/)
  if (!m) return null
  return gradeByIndex(Number(m[1]))
}

/** 把自由文本的学段标签匹配到标准学段；匹配不到返回 null（归入未设置） */
export function matchGrade(stage?: string | null): GradeDef | null {
  if (!stage) return null
  const s = stage.trim().toLowerCase()
  if (!s) return null
  return matchAlias(s) ?? matchNumYear(s) ?? matchCnYear(s)
}

// ---------- 学段纵向分区 ----------
// 画布按学段划分为若干纵向分区（幼儿园最左，向右依次到大学），
// 每个分区宽度固定，节点拖拽不允许越出自己学段的分区。
export const GRADE_COL_WIDTH = 560

/** 学段 key → 分区序号（未设置排在最后）；-1 表示未知 */
export function gradeColumnIndex(key: string | null | undefined): number {
  if (!key) return GRADES.length // 未设置分区
  const i = GRADES.findIndex((g) => g.key === key)
  return i >= 0 ? i : GRADES.length
}

/** 分区序号 → 世界坐标 x 范围 */
export function gradeColumnRange(index: number): { x0: number; x1: number } {
  return { x0: index * GRADE_COL_WIDTH, x1: (index + 1) * GRADE_COL_WIDTH }
}
