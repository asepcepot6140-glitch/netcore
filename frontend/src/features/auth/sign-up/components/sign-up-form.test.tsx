import { beforeEach, describe, expect, it, vi } from 'vitest'
import { render } from 'vitest-browser-react'
import { userEvent } from 'vitest/browser'
import { SignUpForm } from './sign-up-form'

const { registerUserMock, toastErrorMock, toastSuccessMock, navigateMock } =
  vi.hoisted(() => ({
    registerUserMock: vi.fn(),
    toastErrorMock: vi.fn(),
    toastSuccessMock: vi.fn(),
    navigateMock: vi.fn(),
  }))

vi.mock('@/lib/auth-api', () => ({
  getAuthErrorMessage: (error: unknown) =>
    error instanceof Error ? error.message : 'Request failed',
  registerUser: registerUserMock,
}))

vi.mock('sonner', () => ({
  toast: { error: toastErrorMock, success: toastSuccessMock },
}))

vi.mock('@tanstack/react-router', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@tanstack/react-router')>()
  return { ...actual, useNavigate: () => navigateMock }
})

describe('SignUpForm', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    registerUserMock.mockResolvedValue({
      message: 'Registration successful',
      user: {
        id: 7,
        name: 'Test User',
        email: 'a@b.com',
        phone: '081234567890',
      },
    })
  })

  it('renders backend registration fields and submit button', async () => {
    const screen = await render(<SignUpForm />)

    await expect
      .element(screen.getByRole('textbox', { name: /^Full name$/i }))
      .toBeInTheDocument()
    await expect
      .element(screen.getByRole('textbox', { name: /^Email$/i }))
      .toBeInTheDocument()
    await expect
      .element(screen.getByRole('textbox', { name: /^WhatsApp number$/i }))
      .toBeInTheDocument()
    await expect
      .element(screen.getByLabelText(/^Password$/i))
      .toBeInTheDocument()
    await expect
      .element(screen.getByLabelText(/^Confirm Password$/i))
      .toBeInTheDocument()
  })

  it('shows required field validation on empty submission', async () => {
    const screen = await render(<SignUpForm />)
    await userEvent.click(
      screen.getByRole('button', { name: /^Create Account$/i })
    )

    await expect
      .element(screen.getByText('Name must be at least 2 characters long.'))
      .toBeInTheDocument()
    await expect
      .element(screen.getByText('Please enter your email.'))
      .toBeInTheDocument()
    await expect
      .element(
        screen.getByText('WhatsApp number must be at least 10 characters long.')
      )
      .toBeInTheDocument()
    await expect
      .element(screen.getByText('Please enter your password.'))
      .toBeInTheDocument()
  })

  it('posts the backend registration fields and navigates to sign in', async () => {
    const screen = await render(<SignUpForm />)
    await userEvent.fill(
      screen.getByRole('textbox', { name: /^Full name$/i }),
      'Test User'
    )
    await userEvent.fill(
      screen.getByRole('textbox', { name: /^Email$/i }),
      'a@b.com'
    )
    await userEvent.fill(
      screen.getByRole('textbox', { name: /^WhatsApp number$/i }),
      '081234567890'
    )
    await userEvent.fill(screen.getByLabelText(/^Password$/i), '12345678')
    await userEvent.fill(
      screen.getByLabelText(/^Confirm Password$/i),
      '12345678'
    )
    await userEvent.click(
      screen.getByRole('button', { name: /^Create Account$/i })
    )

    await vi.waitFor(() => expect(registerUserMock).toHaveBeenCalledOnce())
    expect(registerUserMock).toHaveBeenCalledWith({
      name: 'Test User',
      email: 'a@b.com',
      phone: '081234567890',
      password: '12345678',
    })
    expect(toastSuccessMock).toHaveBeenCalledWith('Registration successful')
    await vi.waitFor(() =>
      expect(navigateMock).toHaveBeenCalledWith({
        to: '/sign-in',
        replace: true,
      })
    )
  })

  it('shows backend registration failures', async () => {
    registerUserMock.mockRejectedValueOnce(
      new Error('Email is already registered')
    )
    const screen = await render(<SignUpForm />)
    await userEvent.fill(
      screen.getByRole('textbox', { name: /^Full name$/i }),
      'Test User'
    )
    await userEvent.fill(
      screen.getByRole('textbox', { name: /^Email$/i }),
      'a@b.com'
    )
    await userEvent.fill(
      screen.getByRole('textbox', { name: /^WhatsApp number$/i }),
      '081234567890'
    )
    await userEvent.fill(screen.getByLabelText(/^Password$/i), '12345678')
    await userEvent.fill(
      screen.getByLabelText(/^Confirm Password$/i),
      '12345678'
    )
    await userEvent.click(
      screen.getByRole('button', { name: /^Create Account$/i })
    )

    await vi.waitFor(() =>
      expect(toastErrorMock).toHaveBeenCalledWith('Email is already registered')
    )
    expect(navigateMock).not.toHaveBeenCalled()
  })
})
