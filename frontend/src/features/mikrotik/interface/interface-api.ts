import axios from 'axios'
import { useAuthStore } from '@/stores/auth-store'
import { apiBaseURL } from '@/lib/auth-api'

export interface NetInterface {
  id: string
  name: string
  type: string
  running: boolean
  disabled: boolean
  dynamic: boolean
  // Parent interface name (VLAN only); empty when the interface stands alone.
  parent: string // Bits per second and bytes.
  tx_rate: number
  rx_rate: number
  tx_bytes: number
  rx_bytes: number
  // Only ethernet interfaces have speed, auto negotiation, and duplex settings.
  ethernet: boolean
  // Configured speed (used when auto negotiation is off).
  speed: string
  // Negotiated link speed such as "1Gbps"; empty when there is no link.
  rate: string
  auto_negotiation: boolean
  full_duplex: boolean
  last_link_up: string
  last_link_down: string
}

// Edit sends only changed fields.
export type InterfaceInput = Partial<
  Pick<
    NetInterface,
    'name' | 'auto_negotiation' | 'speed' | 'full_duplex' | 'disabled'
  >
>

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

export async function getInterfaces(mikrotikId: number) {
  const response = await axios.get<DataResponse<NetInterface[]>>(
    `${apiBaseURL}/interfaces`,
    requestConfig(mikrotikId)
  )
  return response.data.data
}

// Live data pushed by the server: rates about once per second, status about every two seconds.
// Both only contain interfaces that changed since the previous event.
export interface InterfaceRate {
  name: string
  tx_rate: number
  rx_rate: number
}

export interface InterfaceStatus {
  name: string
  running: boolean
  disabled: boolean
  tx_bytes: number
  rx_bytes: number
  // Negotiated link speed such as "1Gbps"; empty for non-ethernet or when there is no link.
  rate: string
}

export interface InterfaceLive {
  rates: InterfaceRate[]
  status: InterfaceStatus[]
  // The interface list on the router changed; the server closes the stream right after.
  changed: boolean
}

// Reads the Server-Sent Events stream with fetch (EventSource cannot send the Authorization header).
// Resolves when the server ends the stream (it does so about once a minute) so the caller can
// reconnect; rejects on HTTP errors or network failures.
export async function streamInterfaceLive(
  mikrotikId: number,
  onLive: (live: InterfaceLive) => void,
  signal: AbortSignal
) {
  const { headers, params } = requestConfig(mikrotikId)
  const response = await fetch(
    `${apiBaseURL}/interfaces/live?mikrotik_id=${params.mikrotik_id}`,
    { headers: { ...headers, Accept: 'text/event-stream' }, signal }
  )
  if (!response.ok || !response.body) {
    throw new StreamHttpError(response.status)
  }

  const reader = response.body.pipeThrough(new TextDecoderStream()).getReader()
  let buffer = ''
  for (;;) {
    const { value, done } = await reader.read()
    if (done) return
    buffer += value

    // Events are separated by a blank line; keep the unfinished tail for the next chunk.
    const events = buffer.split('\n\n')
    buffer = events.pop() ?? ''
    for (const event of events) {
      const data = event
        .split('\n')
        .filter((line) => line.startsWith('data:'))
        .map((line) => line.slice(5).trim())
        .join('')
      if (!data) continue
      try {
        onLive(JSON.parse(data) as InterfaceLive)
      } catch {
        // A malformed event is skipped; the next one carries fresh data.
      }
    }
  }
}

export class StreamHttpError extends Error {
  status: number

  constructor(status: number) {
    super(`Live stream failed with status ${status}`)
    this.status = status
  }
}

export async function updateInterface(
  mikrotikId: number,
  id: string,
  input: InterfaceInput
) {
  const response = await axios.put<MessageResponse>(
    `${apiBaseURL}/interfaces/${encodeURIComponent(id)}`,
    input,
    requestConfig(mikrotikId)
  )
  return response.data
}

export const setInterfaceDisabled = (
  mikrotikId: number,
  id: string,
  disabled: boolean
) => updateInterface(mikrotikId, id, { disabled })

// One line of /interface/ethernet/monitor, in the order RouterOS prints it.
export interface InterfaceDetail {
  key: string
  value: string
}

export async function getInterfaceDetails(mikrotikId: number, id: string) {
  const response = await axios.get<DataResponse<InterfaceDetail[]>>(
    `${apiBaseURL}/interfaces/${encodeURIComponent(id)}/monitor`,
    requestConfig(mikrotikId)
  )
  return response.data.data
}

export function getInterfaceErrorMessage(error: unknown) {
  if (axios.isAxiosError<{ message?: string; error?: string }>(error)) {
    const message = error.response?.data?.message ?? error.response?.data?.error
    if (typeof message === 'string' && message.length > 0) return message
  }

  return error instanceof Error
    ? error.message
    : 'An error occurred while managing interfaces.'
}
