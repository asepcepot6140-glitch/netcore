import { type ReactNode } from 'react'
import { Button } from '@/components/ui/button'

export function IconButton({
  title,
  onClick,
  destructive,
  children,
}: {
  title: string
  onClick: () => void
  destructive?: boolean
  children: ReactNode
}) {
  return (
    <Button
      variant='ghost'
      size='icon'
      className={`h-6 w-6 ${destructive ? 'text-destructive hover:text-destructive' : ''}`}
      onClick={onClick}
      title={title}
      aria-label={title}
    >
      {children}
    </Button>
  )
}
