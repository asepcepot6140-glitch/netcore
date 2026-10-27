import axios from 'axios'
import { useAuthStore } from '@/stores/auth-store'
import { apiBaseURL } from '@/lib/auth-api'
import type { Delta } from '../hooks/use-live-stream'

export interface PppoeSecret {
  id: string
  disabled: boolean
  name: string
  service: string
  caller_id: string
  password: string
  profile: string
  remote_address: string
  local_address: string
  last_logged_out: string
  last_caller_id: string
  last_disconnect_reason: string
  comment: string
}

export interface PppoeProfile {
  name: string
  local_address: string
  remote_address: string
  only_one: string
  rate_limit: string
  insert_queue_before: string
  parent_queue: string
  address_list: string
  on_up: string
  on_down: string
  comment: string
}

export interface PppoeActiveConnection {
  id: string
  name: string
  service: string
  caller_id: string
  address: string
  uptime: string
  encoding: string
  session_id: string
  limit_bytes_in: string
  limit_bytes_out: string
  radius: boolean
}

export interface PppoePool {
  name: string
  ranges: string
}

// Edit sends only changed fields, so everything except the create-time basics stays optional.
export type PppoeSecretInput = {
  name?: string
  password?: string
  service?: string
  profile?: string
  local_address?: string
  remote_address?: string
  comment?: string
}

// Edit sends only changed fields; an empty string clears the value on the router.
export type PppoeProfileInput = Partial<PppoeProfile>

interface DataResponse<T> {
  data: T
}

// Pushed by the server every few seconds; each table only holds rows that changed.
export interface PppoeLive {
  secrets?: Delta<PppoeSecret>
  offline?: Delta<PppoeSecret>
  active?: Delta<PppoeActiveConnection>
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

export async function getPppoeSecrets(mikrotikId: number) {
  const response = await axios.get<DataResponse<PppoeSecret[]>>(
    `${apiBaseURL}/pppoe/secrets`,
    requestConfig(mikrotikId)
  )
  return response.data.data
}

export async function savePppoeSecret(
  mikrotikId: number,
  input: PppoeSecretInput,
  id?: string
) {
  const config = requestConfig(mikrotikId)
  const response = id
    ? await axios.put<MessageResponse>(
        `${apiBaseURL}/pppoe/secrets/${encodeURIComponent(id)}`,
        input,
        config
      )
    : await axios.post<MessageResponse>(
        `${apiBaseURL}/pppoe/secrets`,
        input,
        config
      )
  return response.data
}

export async function deletePppoeSecret(mikrotikId: number, id: string) {
  const response = await axios.delete<MessageResponse>(
    `${apiBaseURL}/pppoe/secrets/${encodeURIComponent(id)}`,
    requestConfig(mikrotikId)
  )
  return response.data
}

export async function setPppoeSecretDisabled(
  mikrotikId: number,
  id: string,
  disabled: boolean
) {
  const response = await axios.put<MessageResponse>(
    `${apiBaseURL}/pppoe/secrets/${encodeURIComponent(id)}`,
    { disabled },
    requestConfig(mikrotikId)
  )
  return response.data
}

export async function getPppoeProfiles(mikrotikId: number) {
  const response = await axios.get<DataResponse<PppoeProfile[]>>(
    `${apiBaseURL}/pppoe/profiles`,
    requestConfig(mikrotikId)
  )
  return response.data.data
}

export async function savePppoeProfile(
  mikrotikId: number,
  input: PppoeProfileInput,
  name?: string
) {
  const config = requestConfig(mikrotikId)
  const response = name
    ? await axios.put<MessageResponse>(
        `${apiBaseURL}/pppoe/profiles/${encodeURIComponent(name)}`,
        input,
        config
      )
    : await axios.post<MessageResponse>(
        `${apiBaseURL}/pppoe/profiles`,
        input,
        config
      )
  return response.data
}

export async function deletePppoeProfile(mikrotikId: number, name: string) {
  const response = await axios.delete<MessageResponse>(
    `${apiBaseURL}/pppoe/profiles/${encodeURIComponent(name)}`,
    requestConfig(mikrotikId)
  )
  return response.data
}

export async function getPppoeActive(mikrotikId: number) {
  const response = await axios.get<DataResponse<PppoeActiveConnection[]>>(
    `${apiBaseURL}/pppoe/active`,
    requestConfig(mikrotikId)
  )
  return response.data.data
}

export async function disconnectPppoeActive(mikrotikId: number, id: string) {
  const response = await axios.delete<MessageResponse>(
    `${apiBaseURL}/pppoe/active/${encodeURIComponent(id)}`,
    requestConfig(mikrotikId)
  )
  return response.data
}

export async function getPppoeOffline(mikrotikId: number) {
  const response = await axios.get<DataResponse<PppoeSecret[]>>(
    `${apiBaseURL}/pppoe/offline`,
    requestConfig(mikrotikId)
  )
  return response.data.data
}

export async function getPppoePools(mikrotikId: number) {
  const response = await axios.get<DataResponse<PppoePool[]>>(
    `${apiBaseURL}/pppoe/pools`,
    requestConfig(mikrotikId)
  )
  return response.data.data
}

export async function getPppoeQueueNames(mikrotikId: number) {
  const response = await axios.get<DataResponse<string[]>>(
    `${apiBaseURL}/pppoe/queues`,
    requestConfig(mikrotikId)
  )
  return response.data.data
}

export function getPppoeErrorMessage(error: unknown) {
  if (axios.isAxiosError<{ message?: string; error?: string }>(error)) {
    const message = error.response?.data?.message ?? error.response?.data?.error
    if (typeof message === 'string' && message.length > 0) return message
  }

  return error instanceof Error
    ? error.message
    : 'An error occurred while managing PPPoE.'
}
