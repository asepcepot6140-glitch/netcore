import { beforeEach, describe, expect, it, vi } from 'vitest'
import { render, type RenderResult } from 'vitest-browser-react'
import { type Locator, userEvent } from 'vitest/browser'
import { UserAuthForm } from './user-auth-form'

const {
  loginUserMock,
  toastErrorMock,
  toastSuccessMock,
  navigateMock,
  setUserMock,
  setAccessTokenMock,
  setRefreshTokenMock,
} = vi.hoisted(() => ({
  loginUserMock: vi.fn(),
  toastErrorMock: vi.fn(),
  toastSuccessMock: vi.fn(),
  navigateMock: vi.fn(),
  setUserMock: vi.fn(),
  setAccessTokenMock: vi.fn(),
  setRefreshTokenMock: vi.fn(),
}))

vi.mock('@/lib/auth-api', () => ({
  getAuthErrorMessage: (error: unknown) =>
    error instanceof Error ? error.message : 'Request failed',
  loginUser: loginUserMock,
}))

vi.mock('sonner', () => ({
  toast: { error: toastErrorMock, success: toastSuccessMock },
}))

vi.mock('@/stores/auth-store', () => ({
  useAuthStore: () => ({
    auth: {
      setUser: setUserMock,
      setAccessToken: setAccessTokenMock,
      setRefreshToken: setRefreshTokenMock,
    },
  }),
}))

vi.mock('@tanstack/react-router', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@tanstack/react-router')>()
  return {
    ...actual,
    useNavigate: () => navigateMock,
    Link: ({
      children,
      to,
      className,
      ...rest
    }: {
      children?: React.ReactNode
      to: string
      className?: string
    }) => (
      <a href={to} className={className} {...rest}>
        {children}
      </a>
    ),
  }
})

describe('UserAuthForm', () => {
  let screen: RenderResult
  let identifierInput: Locator
  let passwordInput: Locator
  let signInButton: Locator

  beforeEach(async () => {
    vi.clearAllMocks()
    loginUserMock.mockResolvedValue({
      message: 'Login successful',
      data: {
        user: {
          id: 7,
          name: 'Test User',
          email: 'a@b.com',
          phone: '081234567890',
        },
        access_token: 'backend-access-token',
        refresh_token: 'backend-refresh-token',
        token_type: 'Bearer',
        expires_in: 900,
      },
    })
    screen = await render(<UserAuthForm />)
    identifierInput = screen.getByRole('textbox', {
      name: /^Email or WhatsApp number$/i,
    })
    passwordInput = screen.getByLabelText(/^Password$/i)
    signInButton = screen.getByRole('button', { name: /^Sign in$/i })
  })

  it('renders login fields and submit button', async () => {
    await expect.element(identifierInput).toBeInTheDocument()
    await expect.element(passwordInput).toBeInTheDocument()
    await expect.element(signInButton).toBeInTheDocument()
  })

  it('shows validation messages when submitting empty form', async () => {
    await userEvent.click(signInButton)

    await expect
      .element(screen.getByText('Enter your email or WhatsApp number.'))
      .toBeInTheDocument()
    await expect
      .element(screen.getByText('Please enter your password.'))
      .toBeInTheDocument()
  })

  it('sends the backend identifier and stores the returned user and token', async () => {
    await userEvent.fill(identifierInput, '081234567890')
    await userEvent.fill(passwordInput, '12345678')
    await userEvent.click(signInButton)

    await vi.waitFor(() => expect(loginUserMock).toHaveBeenCalledOnce())
    expect(loginUserMock).toHaveBeenCalledWith('081234567890', '12345678')
    expect(setUserMock).toHaveBeenCalledWith({
      id: 7,
      name: 'Test User',
      email: 'a@b.com',
      phone: '081234567890',
    })
    expect(setAccessTokenMock).toHaveBeenCalledWith('backend-access-token')
    expect(setRefreshTokenMock).toHaveBeenCalledWith('backend-refresh-token')
    expect(toastSuccessMock).toHaveBeenCalledWith('Login successful')
    await vi.waitFor(() =>
      expect(navigateMock).toHaveBeenCalledWith({ to: '/', replace: true })
    )
  })

  it('shows backend login failures', async () => {
    loginUserMock.mockRejectedValueOnce(new Error('Invalid email or password'))
    await userEvent.fill(identifierInput, 'a@b.com')
    await userEvent.fill(passwordInput, '12345678')
    await userEvent.click(signInButton)

    await vi.waitFor(() =>
      expect(toastErrorMock).toHaveBeenCalledWith('Invalid email or password')
    )
    expect(setAccessTokenMock).not.toHaveBeenCalled()
    expect(navigateMock).not.toHaveBeenCalled()
  })
})
