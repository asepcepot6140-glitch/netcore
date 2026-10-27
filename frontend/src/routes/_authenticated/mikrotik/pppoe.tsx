import { createFileRoute } from '@tanstack/react-router'
import { PppoeSecrets } from '@/features/mikrotik/pppoe'

export const Route = createFileRoute('/_authenticated/mikrotik/pppoe')({
  component: PppoeSecrets,
})
