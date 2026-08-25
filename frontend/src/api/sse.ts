// SSE 流式请求工具（POST + ReadableStream 解析）

import { ApiError } from '@/api/client'

export async function* streamSSE<T = { delta?: string; error?: string }>(
  path: string,
  body: unknown,
): AsyncGenerator<T> {
  const res = await fetch(path, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  })
  if (!res.ok || !res.body) {
    let msg = `HTTP ${res.status}`
    try {
      const j = await res.json()
      if (j?.error) msg = j.error
    } catch {
      /* ignore */
    }
    throw new ApiError(res.status, msg)
  }
  const reader = res.body.getReader()
  const decoder = new TextDecoder()
  let buf = ''
  while (true) {
    const { done, value } = await reader.read()
    if (done) break
    buf += decoder.decode(value, { stream: true })
    const parts = buf.split('\n\n')
    buf = parts.pop() ?? ''
    for (const part of parts) {
      const line = part.split('\n').find((l) => l.startsWith('data:'))
      if (!line) continue
      const data = line.slice(5).trim()
      if (data === '[DONE]') return
      try {
        yield JSON.parse(data) as T
      } catch {
        /* 忽略无法解析的事件 */
      }
    }
  }
}
