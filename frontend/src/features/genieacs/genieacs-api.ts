import axios from 'axios'
import { useAuthStore } from '@/stores/auth-store'
import { apiBaseURL } from '@/lib/auth-api'

export interface GenieACSConfig {
  id: number
  name: string
  host: string
  port: number
}

export interface GenieACSInput {
  name: string
  host: string
  port: number
}

// Users only deal with the bare host; the backend detects http/https by itself.
export function stripHostScheme(host: string) {
  return host
    .trim()
    .replace(/^https?:\/\//i, '')
    .replace(/\/+$/, '')
}

export interface GenieACSDevice {
  id: string
  serial_number: string
  manufacturer: string
  product_class: string
  last_inform: string | null
  registered: string | null
  status: string
  pppoe_username: string
  rx_power: string
  ssid: string
}

export interface GenieACSWAN {
  path: string
  conn_device: string
  conn_instance: string
  kind: string
  enable: string
  conn_name: string
  enable_vlan: boolean
  vlan: string
  conn_type: string
  service: string
  username: string
  password: string
  nat: string
  lan_bind: string
  lan_bind_options: string[]
  lan_bind_disabled: string[]
  conn_type_options: string[] | null
  service_options: string[] | null
  name_locked: boolean
  status: string
}

export interface GenieACSLAN {
  dhcp: {
    server_enable: string
    min_address: string
    max_address: string
    subnet_mask: string
    ip_routers: string
    dns_servers: string
  }
  ports: {
    path: string
    enable: string
    status: string
    link_speed: string
    l3_enable: string
  }[]
  hosts: {
    path: string
    ip_address: string
    mac_address: string
    host_name: string
    address_source: string
    interface_type: string
    layer2_interface: string
    active: string
    lease_time_remaining: string
  }[]
}

export interface GenieACSWLAN {
  path: string
  index: string
  name: string
  ssid: string
  enable: string
  channel: string
  channel_options: string[] | null
  channel_width: string
  channel_width_options: string[] | null
  channel_width_by_standard?: Record<string, string[]>
  max_clients: string
  auto_channel_enable: string
  operating_frequency_band: string
  standard: string
  standard_options: string[] | null
  transmit_power: string
  beacon_type: string
  beacon_type_options: string[]
  key_passphrase: string
  password_policy?: PasswordPolicy
  ssid_advertisement_enabled: string
  wmm_enable: string
  associated_device_number_of_entries: string
  associated_devices: GenieACSWLANAssociatedDevice[]
}

export interface PasswordPolicy {
  min_length: number
  max_length: number
  specials: string
}

export interface GenieACSWLANAssociatedDevice {
  path: string
  name: string
  mac_address: string
  ip_address: string
  rssi: string
  bandwidth: string
  mode: string
  radio: string
  stay_time: string
  bytes_received: string
  bytes_sent: string
}

function authorizationHeaders() {
  const accessToken = useAuthStore.getState().auth.accessToken
  if (!accessToken) {
    throw new Error('Login session not found. Please sign in again.')
  }

  return { Authorization: `Bearer ${accessToken}` }
}

interface APIResponse<T> {
  data: T
}

interface MessageResponse<T> {
  message: string
  data: T
}

export async function getGenieACSConfigs() {
  const response = await axios.get<APIResponse<GenieACSConfig[]>>(
    `${apiBaseURL}/genieacs`,
    { headers: authorizationHeaders() }
  )
  return response.data.data
}

export async function createGenieACSConfig(input: GenieACSInput) {
  const response = await axios.post<MessageResponse<GenieACSConfig>>(
    `${apiBaseURL}/genieacs`,
    input,
    { headers: authorizationHeaders() }
  )
  return response.data
}

export async function updateGenieACSConfig(id: number, input: GenieACSInput) {
  const response = await axios.put<MessageResponse<GenieACSConfig>>(
    `${apiBaseURL}/genieacs/${id}`,
    input,
    { headers: authorizationHeaders() }
  )
  return response.data
}

export async function deleteGenieACSConfig(id: number) {
  const response = await axios.delete<{ message: string }>(
    `${apiBaseURL}/genieacs/${id}`,
    { headers: authorizationHeaders() }
  )
  return response.data
}

export async function getGenieACSDevices(id: number) {
  const response = await axios.get<
    APIResponse<GenieACSDevice[]> & { total: number }
  >(`${apiBaseURL}/devices/${id}`, { headers: authorizationHeaders() })
  return response.data
}

export async function summonGenieACSDevice(
  configId: number,
  deviceId: string,
  scope?: 'wan' | 'lan' | 'wlan'
) {
  const response = await axios.post<{ message: string; applied: boolean }>(
    `${apiBaseURL}/devices/${configId}/${encodeURIComponent(deviceId)}/summon`,
    {},
    {
      headers: authorizationHeaders(),
      ...(scope ? { params: { scope } } : {}),
    }
  )
  return response.data
}

export async function deleteGenieACSDevice(configId: number, deviceId: string) {
  const response = await axios.delete<{ message: string }>(
    `${apiBaseURL}/devices/${configId}/${encodeURIComponent(deviceId)}`,
    { headers: authorizationHeaders() }
  )
  return response.data
}

export type GenieACSEditTarget = 'wan' | 'lan/dhcp' | 'lan/ports' | 'wlan'

// applied=false means GenieACS queued the change because the device did not answer.
export async function updateGenieACSParameters(
  configId: number,
  deviceId: string,
  target: GenieACSEditTarget,
  path: string,
  values: Record<string, string>
) {
  const response = await axios.put<{ message: string; applied: boolean }>(
    `${apiBaseURL}/devices/${configId}/${encodeURIComponent(deviceId)}/${target}`,
    { path, values },
    { headers: authorizationHeaders() }
  )
  return response.data
}

// applied=false means GenieACS queued the add because the device did not answer.
export async function addGenieACSWAN(
  configId: number,
  deviceId: string,
  kind: 'WANIPConnection' | 'WANPPPConnection'
) {
  const response = await axios.post<{ message: string; applied: boolean }>(
    `${apiBaseURL}/devices/${configId}/${encodeURIComponent(deviceId)}/wan`,
    { kind },
    { headers: authorizationHeaders() }
  )
  return response.data
}

// applied=false means GenieACS queued the delete because the device did not answer.
export async function deleteGenieACSWAN(
  configId: number,
  deviceId: string,
  path: string
) {
  const response = await axios.delete<{ message: string; applied: boolean }>(
    `${apiBaseURL}/devices/${configId}/${encodeURIComponent(deviceId)}/wan`,
    { data: { path }, headers: authorizationHeaders() }
  )
  return response.data
}

export async function getGenieACSDeviceWAN(configId: number, deviceId: string) {
  const response = await axios.get<
    APIResponse<GenieACSWAN[]> & { total: number }
  >(`${apiBaseURL}/devices/${configId}/${encodeURIComponent(deviceId)}/wan`, {
    headers: authorizationHeaders(),
  })
  return response.data.data
}

export async function getGenieACSDeviceLAN(configId: number, deviceId: string) {
  const response = await axios.get<APIResponse<GenieACSLAN>>(
    `${apiBaseURL}/devices/${configId}/${encodeURIComponent(deviceId)}/lan`,
    { headers: authorizationHeaders() }
  )
  return response.data.data
}

export async function getGenieACSDeviceWLAN(
  configId: number,
  deviceId: string
) {
  const response = await axios.get<
    APIResponse<GenieACSWLAN[]> & { total: number }
  >(`${apiBaseURL}/devices/${configId}/${encodeURIComponent(deviceId)}/wlan`, {
    headers: authorizationHeaders(),
  })
  return response.data.data
}
