import axios, { AxiosError, type InternalAxiosRequestConfig } from 'axios'
import { clearCookies } from '@/test-utils/cookies'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { useAuthStore } from '@/stores/auth-store'
import { installAuthRefresh } from './auth-refresh'

const { refreshSessionMock } = vi.hoisted(() => ({
  refreshSessionMock: vi.fn(),
}))

vi.mock('@/lib/auth-api', () => ({ refreshSession: refreshSessionMock }))

function unauthorized(config: InternalAxiosRequestConfig) {
  return new AxiosError('Unauthorized', 'ERR_BAD_REQUEST', config, null, {
    status: 401,
    statusText: 'Unauthorized',
    data: {},
    headers: {},
    config,
  })
}

// Klien palsu: hanya token "new-access" yang diterima server.
function createClient() {
  const calls: string[] = []
  const client = axios.create({
    adapter: async (config) => {
      const header = String(config.headers.get('Authorization') ?? '')
      calls.push(header)
      if (header === 'Bearer new-access') {
        return {
          data: { ok: true },
          status: 200,
          statusText: 'OK',
          headers: {},
          config,
        }
      }
      throw unauthorized(config)
    },
  })
  return { client, calls }
}

function setup() {
  const onSessionExpired = vi.fn()
  const { client, calls } = createClient()
  installAuthRefresh(onSessionExpired, client)
  return { onSessionExpired, client, calls }
}

function request(client: ReturnType<typeof createClient>['client']) {
  return client.get('/data', {
    headers: { Authorization: 'Bearer old-access' },
  })
}

describe('installAuthRefresh', () => {
  beforeEach(() => {
    clearCookies()
    useAuthStore.getState().auth.reset()
    refreshSessionMock.mockReset()
  })

  it('refreshes the tokens and retries the request once on 401', async () => {
    const { onSessionExpired, client, calls } = setup()
    const { auth } = useAuthStore.getState()
    auth.setAccessToken('old-access')
    auth.setRefreshToken('old-refresh')
    refreshSessionMock.mockResolvedValue({
      access_token: 'new-access',
      refresh_token: 'new-refresh',
    })

    const response = await request(client)

    expect(response.data).toEqual({ ok: true })
    expect(refreshSessionMock).toHaveBeenCalledWith('old-refresh')
    expect(calls).toEqual(['Bearer old-access', 'Bearer new-access'])
    expect(useAuthStore.getState().auth.accessToken).toBe('new-access')
    expect(useAuthStore.getState().auth.refreshToken).toBe('new-refresh')
    expect(onSessionExpired).not.toHaveBeenCalled()
  })

  it('shares one refresh between simultaneous 401 responses', async () => {
    const { client } = setup()
    const { auth } = useAuthStore.getState()
    auth.setAccessToken('old-access')
    auth.setRefreshToken('old-refresh')
    refreshSessionMock.mockResolvedValue({
      access_token: 'new-access',
      refresh_token: 'new-refresh',
    })

    const results = await Promise.all([
      request(client),
      request(client),
      request(client),
    ])

    expect(results.every((result) => result.status === 200)).toBe(true)
    expect(refreshSessionMock).toHaveBeenCalledTimes(1)
  })

  it('expires the session when the refresh token is rejected', async () => {
    const { onSessionExpired, client } = setup()
    const { auth } = useAuthStore.getState()
    auth.setAccessToken('old-access')
    auth.setRefreshToken('revoked-refresh')
    refreshSessionMock.mockRejectedValue(unauthorized({} as never))

    await expect(request(client)).rejects.toMatchObject({
      response: { status: 401 },
    })
    expect(onSessionExpired).toHaveBeenCalledOnce()
  })

  it('expires the session when there is no refresh token', async () => {
    const { onSessionExpired, client } = setup()
    useAuthStore.getState().auth.setAccessToken('old-access')

    await expect(request(client)).rejects.toMatchObject({
      response: { status: 401 },
    })
    expect(refreshSessionMock).not.toHaveBeenCalled()
    expect(onSessionExpired).toHaveBeenCalledOnce()
  })

  it('keeps the session when the refresh fails because of the network', async () => {
    const { onSessionExpired, client } = setup()
    const { auth } = useAuthStore.getState()
    auth.setAccessToken('old-access')
    auth.setRefreshToken('old-refresh')
    refreshSessionMock.mockRejectedValue(new Error('Network Error'))

    await expect(request(client)).rejects.toThrow('Network Error')
    expect(onSessionExpired).not.toHaveBeenCalled()
    expect(useAuthStore.getState().auth.refreshToken).toBe('old-refresh')
  })
})
