import axios from 'axios'
import { useAuthStore } from '@/stores/auth-store'
import { apiBaseURL } from '@/lib/auth-api'
import type { Delta } from '../hooks/use-live-stream'

export interface SimpleQueue {
  id: string
  name: string
  target: string
  max_limit: string
  burst_limit: string
  burst_threshold: string
  burst_time: string
  comment: string
  packet_marks: string
  limit_at: string
  priority: string
  bucket_size: string
  queue_type: string
  parent: string
  disabled: boolean
  dynamic: boolean
  invalid: boolean
  // "upload/download" in bps and bytes.
  rate: string
  bytes: string
}

// Edit sends only changed fields; an empty string resets the value on the router.
export type SimpleQueueInput = Partial<
  Omit<
    SimpleQueue,
    'id' | 'dynamic' | 'invalid' | 'rate' | 'bytes' | 'disabled'
  >
> & { disabled?: boolean }

interface DataResponse<T> {
  data: T
}

// Pushed by the server every few seconds; only contains rows that changed.
export interface SimpleQueueLive {
  queues?: Delta<SimpleQueue>
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
    `${apiBaseURL}/queue/${path}`,
    requestConfig(mikrotikId)
  )
  return response.data.data
}

export const getSimpleQueues = (mikrotikId: number) =>
  list<SimpleQueue>('simple', mikrotikId)

export const getQueueTypes = (mikrotikId: number) =>
  list<string>('types', mikrotikId)

export const getQueueParents = (mikrotikId: number) =>
  list<string>('parents', mikrotikId)

export const getPacketMarks = (mikrotikId: number) =>
  list<string>('packet-marks', mikrotikId)

export async function saveSimpleQueue(
  mikrotikId: number,
  input: SimpleQueueInput,
  id?: string
) {
  const config = requestConfig(mikrotikId)
  const response = id
    ? await axios.put<MessageResponse>(
        `${apiBaseURL}/queue/simple/${encodeURIComponent(id)}`,
        input,
        config
      )
    : await axios.post<MessageResponse>(
        `${apiBaseURL}/queue/simple`,
        input,
        config
      )
  return response.data
}

export const setSimpleQueueDisabled = (
  mikrotikId: number,
  id: string,
  disabled: boolean
) => saveSimpleQueue(mikrotikId, { disabled }, id)

export async function deleteSimpleQueue(mikrotikId: number, id: string) {
  const response = await axios.delete<MessageResponse>(
    `${apiBaseURL}/queue/simple/${encodeURIComponent(id)}`,
    requestConfig(mikrotikId)
  )
  return response.data
}

export function getSimpleQueueErrorMessage(error: unknown) {
  if (axios.isAxiosError<{ message?: string; error?: string }>(error)) {
    const message = error.response?.data?.message ?? error.response?.data?.error
    if (typeof message === 'string' && message.length > 0) return message
  }

  return error instanceof Error
    ? error.message
    : 'An error occurred while managing Simple Queue.'
}
