import { createFileRoute } from '@tanstack/react-router'
import { MikrotikRouters } from '@/features/mikrotik/router'

export const Route = createFileRoute('/_authenticated/mikrotik/router')({
  component: MikrotikRouters,
})
