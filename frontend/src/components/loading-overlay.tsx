import { Loader2 } from 'lucide-react'

// Spinner di tengah layar; overlay menahan klik selama proses berjalan.
export function LoadingOverlay({ open }: { open: boolean }) {
  if (!open) return null

  return (
    <div
      role='status'
      aria-label='Loading'
      className='fixed inset-0 z-[60] flex items-center justify-center bg-background/40 backdrop-blur-[1px]'
    >
      <Loader2 className='size-10 animate-spin text-primary' />
    </div>
  )
}
