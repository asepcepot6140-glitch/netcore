import axios from 'axios'
import { useAuthStore } from '@/stores/auth-store'
import { apiBaseURL } from '@/lib/auth-api'
import type { Delta } from '../hooks/use-live-stream'

export interface HotspotUser {
  id: string
  name: string
  password: string
  profile: string
  server: string
  address: string
  mac_address: string
  limit_uptime: string
  limit_bytes_in: string
  limit_bytes_out: string
  limit_bytes_total: string
  uptime: string
  bytes_in: string
  bytes_out: string
  comment: string
  disabled: boolean
}

export interface HotspotProfile {
  id: string
  name: string
  address_pool: string
  shared_users: string
  rate_limit: string
  session_timeout: string
  idle_timeout: string
  keepalive_timeout: string
  status_autorefresh: string
  add_mac_cookie: string
  mac_cookie_timeout: string
  address_list: string
  insert_queue_before: string
  parent_queue: string
  queue_type: string
  transparent_proxy: string
  on_login: string
  on_logout: string
  comment: string
}

export interface HotspotActive {
  id: string
  user: string
  address: string
  mac_address: string
  server: string
  login_by: string
  uptime: string
  idle_time: string
  session_time_left: string
  bytes_in: string
  bytes_out: string
}

export interface HotspotHost {
  id: string
  mac_address: string
  address: string
  to_address: string
  server: string
  uptime: string
  idle_time: string
  bytes_in: string
  bytes_out: string
  authorized: boolean
  bypassed: boolean
  dhcp: boolean
  dynamic: boolean
  comment: string
}

interface HotspotRates {
  tx_rate: number | null
  rx_rate: number | null
}

// One stream event: only changed rows (upsert) plus the full display order when it changed.
export interface HotspotLive {
  users?: Delta<HotspotUser>
  active?: Delta<HotspotActive & HotspotRates>
  hosts?: Delta<HotspotHost & HotspotRates>
}

// Edit sends only changed fields; an empty string clears the value on the router.
export type HotspotUserInput = Partial<
  Omit<HotspotUser, 'id' | 'uptime' | 'bytes_in' | 'bytes_out'>
>

export type HotspotProfileInput = Partial<Omit<HotspotProfile, 'id'>>

interface DataResponse<T> {
  data: T
}

interface MessageResponse {
  message: string
}

function requestConfig(mikrotikId: number) {
  const accessToken = useAuthStore.getState().auth.accessToken
  if (!accessToken) {
    throw new Error('Session not found. Please sign in again.')
  }

  return {
    headers: { Authorization: `Bearer ${accessToken}` },
    params: { mikrotik_id: mikrotikId },
  }
}

async function list<T>(path: string, mikrotikId: number) {
  const response = await axios.get<DataResponse<T[]>>(
    `${apiBaseURL}/hotspot/${path}`,
    requestConfig(mikrotikId)
  )
  return response.data.data
}

async function save<TInput>(
  path: string,
  mikrotikId: number,
  input: TInput,
  id?: string
) {
  const config = requestConfig(mikrotikId)
  const response = id
    ? await axios.put<MessageResponse>(
        `${apiBaseURL}/hotspot/${path}/${encodeURIComponent(id)}`,
        input,
        config
      )
    : await axios.post<MessageResponse>(
        `${apiBaseURL}/hotspot/${path}`,
        input,
        config
      )
  return response.data
}

async function remove(path: string, mikrotikId: number, id: string) {
  const response = await axios.delete<MessageResponse>(
    `${apiBaseURL}/hotspot/${path}/${encodeURIComponent(id)}`,
    requestConfig(mikrotikId)
  )
  return response.data
}

export const getHotspotUsers = (mikrotikId: number) =>
  list<HotspotUser>('users', mikrotikId)

export const saveHotspotUser = (
  mikrotikId: number,
  input: HotspotUserInput,
  id?: string
) => save('users', mikrotikId, input, id)

export const deleteHotspotUser = (mikrotikId: number, id: string) =>
  remove('users', mikrotikId, id)

export const setHotspotUserDisabled = (
  mikrotikId: number,
  id: string,
  disabled: boolean
) => save('users', mikrotikId, { disabled }, id)

export const getHotspotProfiles = (mikrotikId: number) =>
  list<HotspotProfile>('profiles', mikrotikId)

export const saveHotspotProfile = (
  mikrotikId: number,
  input: HotspotProfileInput,
  id?: string
) => save('profiles', mikrotikId, input, id)

export const deleteHotspotProfile = (mikrotikId: number, id: string) =>
  remove('profiles', mikrotikId, id)

export const getHotspotServers = (mikrotikId: number) =>
  list<string>('servers', mikrotikId)

export const getHotspotPools = (mikrotikId: number) =>
  list<string>('pools', mikrotikId)

export const getHotspotQueues = (mikrotikId: number) =>
  list<string>('queues', mikrotikId)

export const getHotspotQueueTypes = (mikrotikId: number) =>
  list<string>('queue-types', mikrotikId)

export const getHotspotActive = (mikrotikId: number) =>
  list<HotspotActive>('active', mikrotikId)

export const kickHotspotActive = (mikrotikId: number, id: string) =>
  remove('active', mikrotikId, id)

export const getHotspotHosts = (mikrotikId: number) =>
  list<HotspotHost>('hosts', mikrotikId)

export const deleteHotspotHost = (mikrotikId: number, id: string) =>
  remove('hosts', mikrotikId, id)

export function getHotspotErrorMessage(error: unknown) {
  if (axios.isAxiosError<{ message?: string; error?: string }>(error)) {
    const message = error.response?.data?.message ?? error.response?.data?.error
    if (typeof message === 'string' && message.length > 0) return message
  }

  return error instanceof Error
    ? error.message
    : 'An error occurred while managing Hotspot.'
}
