import { z } from 'zod'
import { createFileRoute } from '@tanstack/react-router'
import { GenieACSDeviceDetail } from '@/features/genieacs/device-detail'

export const Route = createFileRoute(
  '/_authenticated/genieacs/devices/$deviceId'
)({
  validateSearch: z.object({ configId: z.coerce.number() }),
  component: GenieACSDeviceDetail,
})
