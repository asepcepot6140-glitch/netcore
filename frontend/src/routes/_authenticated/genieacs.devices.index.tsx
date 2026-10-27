import { createFileRoute } from '@tanstack/react-router'
import { GenieACSDevices } from '@/features/genieacs/devices'

export const Route = createFileRoute('/_authenticated/genieacs/devices/')({
  component: GenieACSDevices,
})
