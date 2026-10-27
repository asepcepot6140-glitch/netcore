import { type SVGProps } from 'react'
import { cn } from '@/lib/utils'

export function Logo({ className, ...props }: SVGProps<SVGSVGElement>) {
  return (
    <svg
      id='apk-net-logo'
      viewBox='0 0 24 24'
      xmlns='http://www.w3.org/2000/svg'
      height='24'
      width='24'
      fill='none'
      stroke='currentColor'
      strokeWidth='2'
      strokeLinecap='round'
      strokeLinejoin='round'
      className={cn('size-6', className)}
      {...props}
    >
      <title>APK Net</title>
      {/* Hub-and-spoke network: central node linked to three nodes */}
      <circle cx='12' cy='12' r='2.5' />
      <circle cx='12' cy='4' r='2' />
      <circle cx='5' cy='18' r='2' />
      <circle cx='19' cy='18' r='2' />
      <path d='M12 9.5V6M10.1 13.6 6.5 16.7M13.9 13.6l3.6 3.1' />
    </svg>
  )
}
