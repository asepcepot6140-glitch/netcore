import { createFileRoute } from '@tanstack/react-router'
import { GenieACSIndex } from '@/features/genieacs'

export const Route = createFileRoute('/_authenticated/genieacs/')({
  component: GenieACSIndex,
})
