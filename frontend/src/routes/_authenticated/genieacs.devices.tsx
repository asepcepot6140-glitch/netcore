import { z } from 'zod'
import { createFileRoute, Outlet } from '@tanstack/react-router'

export const Route = createFileRoute('/_authenticated/genieacs/devices')({
  validateSearch: z.object({ configId: z.coerce.number().optional() }),
  component: Outlet,
})
