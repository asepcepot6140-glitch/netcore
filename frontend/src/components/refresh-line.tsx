import { useEffect, useRef, useState } from 'react'
import { cn } from '@/lib/utils'

// Keeps a short-lived flag visible for at least `minMs` so fast requests don't flicker.
function useMinimumVisible(active: boolean, minMs: number) {
  const [visible, setVisible] = useState(false)
  const shownAt = useRef(0)

  useEffect(() => {
    if (active) {
      const timer = window.setTimeout(() => {
        shownAt.current = Date.now()
        setVisible(true)
      }, 0)
      return () => window.clearTimeout(timer)
    }

    const remaining = Math.max(0, minMs - (Date.now() - shownAt.current))
    const timer = window.setTimeout(() => setVisible(false), remaining)
    return () => window.clearTimeout(timer)
  }, [active, minMs])

  return visible
}

type RefreshLineProps = {
  active: boolean
  className?: string
}

// Full-width line that shows a purple sweep only while data is being fetched.
// It always reserves its 2px height (invisible when idle), so the layout never shifts.
export function RefreshLine({ active, className }: RefreshLineProps) {
  const visible = useMinimumVisible(active, 600)

  return (
    <div
      className={cn(
        'relative h-0.5 w-full overflow-hidden rounded-full',
        className
      )}
    >
      {visible && (
        <div
          role='progressbar'
          aria-label='Refreshing data'
          className='absolute inset-0 animate-[refresh-line-sweep_1.2s_linear_infinite] bg-[linear-gradient(90deg,#8b5cf6_0%,#c4b5fd_50%,#8b5cf6_100%)] bg-[length:200%_100%]'
        />
      )}
    </div>
  )
}
