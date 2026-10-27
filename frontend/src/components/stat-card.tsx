import { type LucideIcon } from 'lucide-react'
import { cn } from '@/lib/utils'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'

const TONES = {
  blue: 'bg-blue-500/10 text-blue-600 dark:text-blue-400',
  green: 'bg-emerald-500/10 text-emerald-600 dark:text-emerald-400',
  red: 'bg-red-500/10 text-red-600 dark:text-red-400',
  violet: 'bg-violet-500/10 text-violet-600 dark:text-violet-400',
} as const

type StatCardProps = {
  title: string
  value: number | string
  description: string
  icon: LucideIcon
  tone: keyof typeof TONES
  loading: boolean
  active?: boolean
  onClick?: () => void
}

export function StatCard({
  title,
  value,
  description,
  icon: Icon,
  tone,
  loading,
  active = false,
  onClick,
}: StatCardProps) {
  const card = (
    <Card
      className={cn(
        'gap-1 py-2 sm:py-1.5',
        onClick && 'transition-colors hover:bg-muted/50',
        active && 'ring-2 ring-primary'
      )}
    >
      <CardHeader className='flex flex-row items-center justify-between gap-2 px-3 py-0 sm:px-4'>
        <CardTitle className='truncate text-xs font-medium sm:text-sm'>
          {title}
        </CardTitle>
        <span className={cn('shrink-0 rounded-md p-1 sm:p-1.5', TONES[tone])}>
          <Icon className='size-3.5 sm:size-4' />
        </span>
      </CardHeader>
      <CardContent className='px-3 py-0 sm:flex sm:items-baseline sm:gap-2 sm:px-4'>
        {loading ? (
          <Skeleton className='h-6 w-12' />
        ) : (
          <div className='text-lg leading-tight font-bold sm:text-xl'>
            {value}
          </div>
        )}
        <p className='hidden truncate text-xs text-muted-foreground sm:block'>
          {description}
        </p>
      </CardContent>
    </Card>
  )

  // Static card when there is no click handler (e.g. dashboard).
  if (!onClick) return <div className='min-w-0'>{card}</div>

  return (
    <button
      type='button'
      onClick={onClick}
      aria-pressed={active}
      className='min-w-0 rounded-xl text-start focus-visible:ring-2 focus-visible:ring-ring focus-visible:outline-none'
    >
      {card}
    </button>
  )
}
