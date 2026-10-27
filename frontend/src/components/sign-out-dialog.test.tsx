import { beforeEach, describe, expect, it, vi } from 'vitest'
import { render } from 'vitest-browser-react'
import { userEvent } from 'vitest/browser'
import { SignOutDialog } from './sign-out-dialog'

const navigate = vi.fn()
const reset = vi.fn()

vi.mock('@/stores/auth-store', () => ({
  useAuthStore: () => ({
    auth: { reset },
  }),
}))

vi.mock('@tanstack/react-router', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@tanstack/react-router')>()
  return {
    ...actual,
    useNavigate: () => navigate,
  }
})

describe('SignOutDialog', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('calls auth.reset and navigates to a clean sign-in url', async () => {
    const { getByRole } = await render(
      <SignOutDialog open onOpenChange={vi.fn()} />
    )

    await userEvent.click(getByRole('button', { name: /^Sign out$/i }))

    expect(reset).toHaveBeenCalledOnce()
    expect(navigate).toHaveBeenCalledWith({
      to: '/sign-in',
      replace: true,
    })
  })

  it('does not call reset or navigate when Cancel is clicked', async () => {
    const { getByRole } = await render(
      <SignOutDialog open onOpenChange={vi.fn()} />
    )

    await userEvent.click(getByRole('button', { name: /^Cancel$/i }))

    expect(reset).not.toHaveBeenCalled()
    expect(navigate).not.toHaveBeenCalled()
  })
})
