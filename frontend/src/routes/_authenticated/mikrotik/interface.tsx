import { createFileRoute } from '@tanstack/react-router'
import { InterfaceManagement } from '@/features/mikrotik/interface'

export const Route = createFileRoute('/_authenticated/mikrotik/interface')({
  component: InterfaceManagement,
})
