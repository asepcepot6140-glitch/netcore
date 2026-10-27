import axios from 'axios'

const configuredAPIURL = (import.meta.env.VITE_API_URL || '/api').replace(
  /\/+$/,
  ''
)

export const apiBaseURL = configuredAPIURL.endsWith('/api')
  ? configuredAPIURL
  : `${configuredAPIURL}/api`

const authAPI = axios.create({
  baseURL: apiBaseURL,
  headers: {
    'Content-Type': 'application/json',
  },
})

export interface AuthUser {
  id: number
  name: string
  email: string
  phone: string
}

export interface RegisterRequest {
  name: string
  email: string
  phone: string
  password: string
}

export interface RegisterResponse {
  message: string
  user: AuthUser
}

export interface LoginResponse {
  message: string
  data: {
    user: AuthUser
    access_token: string
    refresh_token: string
    token_type: string
    expires_in: number
  }
}

export async function registerUser(input: RegisterRequest) {
  const response = await authAPI.post<RegisterResponse>('/register', input)
  return response.data
}

export async function loginUser(identifier: string, password: string) {
  const response = await authAPI.post<LoginResponse>('/login', {
    identifier,
    password,
  })
  return response.data
}

export interface RefreshResponse {
  access_token: string
  refresh_token: string
  token_type: string
  expires_in: number
}

// Memakai instance authAPI (bukan axios global) agar tidak memicu interceptor 401 dan berulang.
export async function refreshSession(refreshToken: string) {
  const response = await authAPI.post<RefreshResponse>('/refresh', {
    refresh_token: refreshToken,
  })
  return response.data
}

export async function getCurrentUser(accessToken: string) {
  const response = await authAPI.get<AuthUser>('/me', {
    headers: { Authorization: `Bearer ${accessToken}` },
  })
  return response.data
}

export async function forgotPassword(phone: string) {
  const response = await authAPI.post<{ message: string }>('/forgot-password', {
    phone,
  })
  return response.data
}

export async function verifyOtp(phone: string, otp: string) {
  const response = await authAPI.post<{ message: string }>('/verify-otp', {
    phone,
    otp,
  })
  return response.data
}

export async function resetPassword(phone: string, otp: string, newPassword: string) {
  const response = await authAPI.post<{ message: string }>('/reset-password', {
    phone,
    otp,
    new_password: newPassword,
  })
  return response.data
}

export function getAuthErrorMessage(error: unknown) {
  if (axios.isAxiosError<{ message?: string }>(error)) {
    const message = error.response?.data?.message
    if (typeof message === 'string' && message.length > 0) {
      return message
    }
  }

  if (error instanceof Error && error.message) {
    return error.message
  }

  return 'Something went wrong. Please try again.'
}
