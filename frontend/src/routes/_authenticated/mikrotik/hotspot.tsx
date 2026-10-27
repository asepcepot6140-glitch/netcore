import { createFileRoute } from '@tanstack/react-router'
import { HotspotManagement } from '@/features/mikrotik/hotspot'

export const Route = createFileRoute('/_authenticated/mikrotik/hotspot')({
  component: HotspotManagement,
})
