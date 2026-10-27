import { useEffect, useState } from 'react'
import type { GenieACSDevice } from './genieacs-api'

const STALE_INFORM_MS = 24 * 60 * 60 * 1000
const CLOCK_TICK_MS = 60_000

// Refreshed periodically so "stale" stays accurate without calling Date.now during render.
export function useNow() {
  const [now, setNow] = useState(() => Date.now())
  useEffect(() => {
    const timer = setInterval(() => setNow(Date.now()), CLOCK_TICK_MS)
    return () => clearInterval(timer)
  }, [])
  return now
}

export function isStaleDevice(device: GenieACSDevice, now: number) {
  const lastInform = device.last_inform ? Date.parse(device.last_inform) : NaN
  return Number.isFinite(lastInform) && now - lastInform > STALE_INFORM_MS
}
