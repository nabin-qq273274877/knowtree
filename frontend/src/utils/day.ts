// 极简时间格式化（避免引入 dayjs 全包，后续如需再换）
export const dayjs = {
  format(unix: number): string {
    if (!unix) return '—'
    const d = new Date(unix * 1000)
    const p = (n: number) => String(n).padStart(2, '0')
    return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())} ${p(d.getHours())}:${p(d.getMinutes())}`
  },
}
