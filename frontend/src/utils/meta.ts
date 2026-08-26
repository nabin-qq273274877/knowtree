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
// 固定学段序列：学前 → 幼儿园小/中/大班 → 一至六年级 → 初一至初三 → 高一至高三
// → 大一至大四 → 硕士 → 博士；无法匹配的归入最后的「未知领域」。
export interface GradeDef {
  key: string
  label: string
  color: string
  aliases: string[]
}

interface StageSeed {
  key: string
  label: string
  aliases: string[]
}

const STAGE_SEEDS: StageSeed[] = [
  { key: 'xueqian', label: '学前', aliases: ['学前', '学前班', '托班', '早教'] },
  { key: 'xb', label: '小班', aliases: ['小班'] },
  { key: 'zb', label: '中班', aliases: ['中班'] },
  { key: 'db', label: '大班', aliases: ['大班'] },
  { key: 'g1', label: '一年级', aliases: ['一年级', '1年级'] },
  { key: 'g2', label: '二年级', aliases: ['二年级', '2年级'] },
  { key: 'g3', label: '三年级', aliases: ['三年级', '3年级'] },
  { key: 'g4', label: '四年级', aliases: ['四年级', '4年级'] },
  { key: 'g5', label: '五年级', aliases: ['五年级', '5年级'] },
  { key: 'g6', label: '六年级', aliases: ['六年级', '6年级'] },
  { key: 'j1', label: '初一', aliases: ['初一', '七年级', '7年级'] },
  { key: 'j2', label: '初二', aliases: ['初二', '八年级', '8年级'] },
  { key: 'j3', label: '初三', aliases: ['初三', '九年级', '9年级'] },
  { key: 's1', label: '高一', aliases: ['高一', '十年级', '10年级'] },
  { key: 's2', label: '高二', aliases: ['高二', '十一年级', '11年级'] },
  { key: 's3', label: '高三', aliases: ['高三', '十二年级', '12年级'] },
  { key: 'd1', label: '大一', aliases: ['大一', '大学一', '本科一', '大学', '本科', '高校', '学院'] },
  { key: 'd2', label: '大二', aliases: ['大二', '大学二', '本科二'] },
  { key: 'd3', label: '大三', aliases: ['大三', '大学三', '本科三'] },
  { key: 'd4', label: '大四', aliases: ['大四', '大学四', '本科四'] },
  { key: 'master', label: '硕士', aliases: ['硕士', '研究生', '研一', '研二', '研三'] },
  { key: 'phd', label: '博士', aliases: ['博士', '博士生', '博士后', 'phd'] },
]

// 彩虹锚点色：在 22 个学段间做线性插值，保证相邻学段颜色平滑可辨
const RAINBOW_ANCHORS = [
  '#e91e63', '#f44336', '#ff9800', '#ffc107', '#cddc39',
  '#8bc34a', '#4caf50', '#009688', '#00bcd4', '#3f51b5', '#673ab7',
]

function hexToRgb(hex: string): [number, number, number] {
  return [
    parseInt(hex.slice(1, 3), 16),
    parseInt(hex.slice(3, 5), 16),
    parseInt(hex.slice(5, 7), 16),
  ]
}

function rgbToHex(rgb: [number, number, number]): string {
  return '#' + rgb.map((v) => Math.max(0, Math.min(255, Math.round(v))).toString(16).padStart(2, '0')).join('')
}

function rampColor(t: number): string {
  const clamped = Math.max(0, Math.min(1, t))
  const seg = clamped * (RAINBOW_ANCHORS.length - 1)
  const i = Math.min(RAINBOW_ANCHORS.length - 2, Math.floor(seg))
  const f = seg - i
  const a = hexToRgb(RAINBOW_ANCHORS[i])
  const b = hexToRgb(RAINBOW_ANCHORS[i + 1])
  return rgbToHex([a[0] + (b[0] - a[0]) * f, a[1] + (b[1] - a[1]) * f, a[2] + (b[2] - a[2]) * f])
}

export const GRADES: GradeDef[] = STAGE_SEEDS.map((s, i) => ({
  key: s.key,
  label: s.label,
  color: rampColor(i / (STAGE_SEEDS.length - 1)),
  aliases: s.aliases,
}))

// 无法匹配任何学段的节点统一归入「未知领域」段（不是设置按钮！）
export const UNSET_GRADE = {
  key: '__unknown__',
  label: '未知领域',
  color: '#8a93a6',
  aliases: ['未知领域', '未知', '其他', '杂项', '未分类'],
}

const CN_NUM: Record<string, number> = {
  一: 1, 二: 2, 三: 3, 四: 4, 五: 5, 六: 6, 七: 7, 八: 8, 九: 9, 十: 10,
}

/** 年级数字(1-12) → 学段定义 */
function gradeByNumber(n: number): GradeDef | null {
  const key = n >= 1 && n <= 6 ? `g${n}` : n >= 7 && n <= 9 ? `j${n - 6}` : n >= 10 && n <= 12 ? `s${n - 9}` : null
  if (!key) return null
  return GRADES.find((g) => g.key === key) ?? null
}

// 预展开所有别名词并按长度倒序，保证「大学三年级」优先命中「大三」而非「大学」
const ALL_ALIASES: { text: string; grade: GradeDef }[] = GRADES.flatMap((g) =>
  g.aliases.map((text) => ({ text, grade: g })),
).sort((a, b) => b.text.length - a.text.length)

function matchAlias(stage: string): GradeDef | null {
  for (const { text, grade } of ALL_ALIASES) {
    if (stage.includes(text)) return grade
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
  return gradeByNumber(n)
}

function matchNumYear(stage: string): GradeDef | null {
  const m = stage.match(/(\d{1,2})\s*年级/)
  if (!m) return null
  return gradeByNumber(Number(m[1]))
}

/** 把自由文本的学段标签匹配到标准学段；匹配不到返回 null（归入未知领域） */
export function matchGrade(stage?: string | null): GradeDef | null {
  if (!stage) return null
  const s = stage.trim().toLowerCase()
  if (!s) return null
  for (const a of UNSET_GRADE.aliases) {
    if (s.includes(a)) return null // 明确写了「未知/其他」的直接进未知领域
  }
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
