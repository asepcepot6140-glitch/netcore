import axios from 'axios'
import { useAuthStore } from '@/stores/auth-store'
import { apiBaseURL } from '@/lib/auth-api'

export interface MikrotikRouter {
  id: number
  name: string
  host: string
  username: string
  port: number
}

export interface CreateMikrotikInput {
  name: string
  host: string
  username: string
  password: string
  port: number
}

interface APIResponse<T> {
  data: T
}

interface CreateMikrotikResponse {
  message: string
  data: MikrotikRouter
}

function authorizationHeaders() {
  const accessToken = useAuthStore.getState().auth.accessToken
  if (!accessToken) {
    throw new Error('Session not found. Please sign in again.')
  }

  return { Authorization: `Bearer ${accessToken}` }
}

export async function getMikrotikRouters() {
  const response = await axios.get<APIResponse<MikrotikRouter[]>>(
    `${apiBaseURL}/mikrotiks`,
    { headers: authorizationHeaders() }
  )
  return response.data.data
}

export async function createMikrotikRouter(input: CreateMikrotikInput) {
  const response = await axios.post<CreateMikrotikResponse>(
    `${apiBaseURL}/mikrotiks`,
    input,
    { headers: authorizationHeaders() }
  )
  return response.data
}

export async function updateMikrotikRouter(
  id: number,
  input: Partial<CreateMikrotikInput>
) {
  const response = await axios.put<CreateMikrotikResponse>(
    `${apiBaseURL}/mikrotiks/${id}`,
    input,
    { headers: authorizationHeaders() }
  )
  return response.data
}

export async function deleteMikrotikRouter(id: number) {
  const response = await axios.delete<{ message: string }>(
    `${apiBaseURL}/mikrotiks/${id}`,
    { headers: authorizationHeaders() }
  )
  return response.data
}
