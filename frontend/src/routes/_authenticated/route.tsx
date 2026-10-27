import { createFileRoute, redirect } from '@tanstack/react-router'
import { useAuthStore } from '@/stores/auth-store'
import { AuthenticatedLayout } from '@/components/layout/authenticated-layout'

export const Route = createFileRoute('/_authenticated')({
  beforeLoad: () => {
    if (!useAuthStore.getState().auth.accessToken) {
      throw redirect({ to: '/sign-in', replace: true })
    }
  },
  component: AuthenticatedLayout,
})
