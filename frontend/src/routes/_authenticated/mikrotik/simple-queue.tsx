import { createFileRoute } from '@tanstack/react-router'
import { SimpleQueueManagement } from '@/features/mikrotik/simple-queue'

export const Route = createFileRoute('/_authenticated/mikrotik/simple-queue')({
  component: SimpleQueueManagement,
})
