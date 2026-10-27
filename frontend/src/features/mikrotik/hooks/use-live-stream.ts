import { useEffect, useRef } from 'react'
import { useAuthStore } from '@/stores/auth-store'
import { apiBaseURL } from '@/lib/auth-api'

// Change to one table since the previous event. `upsert` holds full new or changed rows; `order`
// lists every id in display order and is only present when the list or its order changed (an id
// missing from it means the row is gone).
export interface Delta<T> {
  upsert: T[]
  order: string[] | null
}

export class StreamHttpError extends Error {
  status: number

  constructor(status: number) {
    super(`Live stream failed with status ${status}`)
    this.status = status
  }
}

// Applies a delta to the rows currently shown. Returns the same array when nothing changed.
export function applyDelta<T>(
  current: T[],
  delta: Delta<T> | undefined,
  getId: (row: T) => string
): T[] {
  if (!delta) return current
  const { upsert, order } = delta
  if (upsert.length === 0 && !order) return current

  const upserts = new Map(upsert.map((row) => [getId(row), row]))
  if (!order) {
    return current.map((row) => upserts.get(getId(row)) ?? row)
  }

  const known = new Map(current.map((row) => [getId(row), row]))
  for (const [id, row] of upserts) known.set(id, row)
  const next: T[] = []
  for (const id of order) {
    const row = known.get(id)
    if (row) next.push(row)
  }
  return next
}

// Reads the Server-Sent Events stream with fetch (EventSource cannot send the Authorization
// header). Resolves when the server ends the stream (about once a minute) so the caller can
// reconnect; rejects on HTTP errors or network failures.
export async function streamLive<T>(
  path: string,
  mikrotikId: number,
  onLive: (live: T) => void,
  signal: AbortSignal
) {
  const accessToken = useAuthStore.getState().auth.accessToken
  if (!accessToken) {
    throw new Error('Session not found. Please sign in again.')
  }

  const response = await fetch(
    `${apiBaseURL}${path}?mikrotik_id=${mikrotikId}`,
    {
      headers: {
        Authorization: `Bearer ${accessToken}`,
        Accept: 'text/event-stream',
      },
      signal,
    }
  )
  if (!response.ok || !response.body) {
    throw new StreamHttpError(response.status)
  }

  const reader = response.body.pipeThrough(new TextDecoderStream()).getReader()
  let buffer = ''
  for (;;) {
    const { value, done } = await reader.read()
    if (done) return
    buffer += value

    // Events are separated by a blank line; keep the unfinished tail for the next chunk.
    const events = buffer.split('\n\n')
    buffer = events.pop() ?? ''
    for (const event of events) {
      const data = event
        .split('\n')
        .filter((line) => line.startsWith('data:'))
        .map((line) => line.slice(5).trim())
        .join('')
      if (!data) continue
      try {
        onLive(JSON.parse(data) as T)
      } catch {
        // A malformed event is skipped; the next one carries fresh data.
      }
    }
  }
}

const RETRY_DELAY_MS = 3_000
const FULL_RETRY_DELAY_MS = 15_000

// Keeps one live stream open for the selected router. It reconnects when the server closes the
// stream, waits longer when the router or user has no free stream (429), drops the stream while
// the tab is hidden, and reconnects when the tab is visible again.
export function useLiveStream<T>(
  path: string,
  routerId: string,
  onLive: (live: T) => void
) {
  const handler = useRef(onLive)
  useEffect(() => {
    handler.current = onLive
  })

  useEffect(() => {
    if (!routerId) return

    let controller: AbortController | null = null
    let retryTimer: number | undefined
    let stopped = false

    const connect = async () => {
      if (stopped || document.hidden) return
      controller = new AbortController()
      let delay = 0
      try {
        await streamLive<T>(
          path,
          Number(routerId),
          (live) => handler.current(live),
          controller.signal
        )
      } catch (error) {
        if (stopped || controller.signal.aborted) return
        delay =
          error instanceof StreamHttpError && error.status === 429
            ? FULL_RETRY_DELAY_MS
            : RETRY_DELAY_MS
      }
      if (!stopped) retryTimer = window.setTimeout(() => void connect(), delay)
    }

    const onVisibilityChange = () => {
      window.clearTimeout(retryTimer)
      if (document.hidden) {
        controller?.abort()
      } else {
        void connect()
      }
    }

    document.addEventListener('visibilitychange', onVisibilityChange)
    void connect()

    return () => {
      stopped = true
      document.removeEventListener('visibilitychange', onVisibilityChange)
      window.clearTimeout(retryTimer)
      controller?.abort()
    }
  }, [path, routerId])
}
