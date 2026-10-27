import axios, {
  AxiosError,
  type AxiosInstance,
  type InternalAxiosRequestConfig,
} from 'axios'
import { useAuthStore } from '@/stores/auth-store'
import { refreshSession } from '@/lib/auth-api'

type RetryableConfig = InternalAxiosRequestConfig & { _retried?: boolean }

// Status yang berarti refresh token memang tidak berlaku lagi (bukan gangguan jaringan).
const REJECTED_STATUSES = [400, 401, 403]

let inflight: Promise<string | null> | null = null

// Satu refresh untuk semua request 401 yang bersamaan: backend merotasi refresh token,
// jadi pemakaian kedua dengan token lama pasti ditolak.
function refreshAccessToken(): Promise<string | null> {
  inflight ??= (async () => {
    const { auth } = useAuthStore.getState()
    if (!auth.refreshToken) return null

    try {
      const data = await refreshSession(auth.refreshToken)
      auth.setAccessToken(data.access_token)
      auth.setRefreshToken(data.refresh_token)
      return data.access_token
    } catch (error) {
      if (
        error instanceof AxiosError &&
        error.response &&
        REJECTED_STATUSES.includes(error.response.status)
      ) {
        return null
      }
      // Gangguan jaringan/server: jangan paksa logout, biarkan request asli gagal.
      throw error
    }
  })().finally(() => {
    inflight = null
  })

  return inflight
}

/**
 * Saat request terautentikasi mendapat 401, tukar refresh token lalu ulangi request sekali.
 * `onSessionExpired` dipanggil hanya jika sesi benar-benar tidak bisa dipulihkan.
 */
export function installAuthRefresh(
  onSessionExpired: () => void,
  client: AxiosInstance = axios
) {
  client.interceptors.response.use(
    (response) => response,
    async (error: unknown) => {
      if (
        !(error instanceof AxiosError) ||
        error.response?.status !== 401 ||
        !error.config
      ) {
        return Promise.reject(error)
      }

      const config = error.config as RetryableConfig
      const usedHeader = String(config.headers?.get('Authorization') ?? '')

      // Request tanpa token, atau sudah diulang sekali: tidak ada yang bisa dipulihkan.
      if (!usedHeader || config._retried) {
        onSessionExpired()
        return Promise.reject(error)
      }

      const usedToken = usedHeader.replace(/^Bearer\s+/i, '')
      const currentToken = useAuthStore.getState().auth.accessToken

      let token: string | null
      try {
        // Token sudah diganti oleh refresh lain yang baru selesai: cukup pakai yang terbaru.
        token =
          currentToken && currentToken !== usedToken
            ? currentToken
            : await refreshAccessToken()
      } catch (refreshError) {
        // Error jaringan/server, bukan 401: QueryCache tidak menganggapnya sesi habis.
        return Promise.reject(refreshError)
      }

      if (!token) {
        onSessionExpired()
        return Promise.reject(error)
      }

      config._retried = true
      config.headers.set('Authorization', `Bearer ${token}`)
      return client(config)
    }
  )
}
