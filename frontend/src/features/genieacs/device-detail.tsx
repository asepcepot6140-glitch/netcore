import {
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
  type ReactNode,
} from 'react'
import { useNavigate, useParams, useSearch } from '@tanstack/react-router'
import { ThemeProvider } from '@mui/material'
import {
  ArrowLeft,
  Loader2,
  Pencil,
  Plus,
  RefreshCw,
  Trash2,
} from 'lucide-react'
import { toast } from 'sonner'
import { getAuthErrorMessage } from '@/lib/auth-api'
import { Button } from '@/components/ui/button'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { ConfirmDialog } from '@/components/confirm-dialog'
import { MuiDataTable, type DataTableColumn } from '@/components/data-table'
import { Header } from '@/components/layout/header'
import { Main } from '@/components/layout/main'
import { LoadingOverlay } from '@/components/loading-overlay'
import { ProfileDropdown } from '@/components/profile-dropdown'
import { ThemeSwitch } from '@/components/theme-switch'
import {
  EditParametersDialog,
  type EditTarget,
  type ParameterField,
} from './edit-parameters-dialog'
import {
  addGenieACSWAN,
  deleteGenieACSWAN,
  getGenieACSDeviceLAN,
  getGenieACSDeviceWAN,
  getGenieACSDeviceWLAN,
  summonGenieACSDevice,
  updateGenieACSParameters,
  type GenieACSLAN,
  type GenieACSWAN,
  type GenieACSWLAN,
  type GenieACSWLANAssociatedDevice,
} from './genieacs-api'
import { useMuiTheme } from './use-mui-theme'

type Scope = 'wan' | 'lan' | 'wlan'

const IPOE_KIND = 'WANIPConnection'
const PPPOE_KIND = 'WANPPPConnection'

// Devices report either LAN1/SSID1 labels or full object paths; both become labels.
function lanBindLabels(raw: string) {
  return raw
    .split(',')
    .map((item) => item.trim())
    .filter(Boolean)
    .map((item) =>
      item
        .replace(/^.*\.LANEthernetInterfaceConfig\.(\d+)$/, 'LAN$1')
        .replace(/^.*\.WLANConfiguration\.(\d+)$/, 'SSID$1')
    )
}

function formatLanBind(raw: string) {
  return lanBindLabels(raw).join(', ') || '-'
}

function buildWanColumns(
  withCredentials: boolean
): DataTableColumn<GenieACSWAN>[] {
  return [
    { header: 'Enable', cell: (wan) => wan.enable || '-' },
    { header: 'Name', cell: (wan) => wan.conn_name || '-' },
    {
      header: 'Enable VLAN',
      cell: (wan) => (wan.enable_vlan ? 'true' : 'false'),
    },
    { header: 'VLAN', cell: (wan) => wan.vlan || '-' },
    { header: 'Conn Type', cell: (wan) => wan.conn_type || '-' },
    { header: 'Service', cell: (wan) => wan.service || '-' },
    ...(withCredentials
      ? [
          {
            header: 'Username',
            cell: (wan: GenieACSWAN) => wan.username || '-',
          },
          {
            header: 'Password',
            cell: (wan: GenieACSWAN) => (wan.password ? '********' : '-'),
          },
        ]
      : []),
    { header: 'NAT', cell: (wan) => wan.nat || '-' },
    { header: 'LanBind', cell: (wan) => formatLanBind(wan.lan_bind) },
    { header: 'Status', cell: (wan) => wan.status || '-' },
  ]
}

const ipoeColumns = buildWanColumns(false)
const pppoeColumns = buildWanColumns(true)

const portColumns: DataTableColumn<GenieACSLAN['ports'][number]>[] = [
  { header: 'Enabled', cell: (port) => port.enable || '-' },
  { header: 'Status', cell: (port) => port.status || '-' },
  { header: 'Link Speed', cell: (port) => port.link_speed || '-' },
  { header: 'L3 Enable', cell: (port) => port.l3_enable || '-' },
]

const hostColumns: DataTableColumn<GenieACSLAN['hosts'][number]>[] = [
  { header: 'IP Address', cell: (host) => host.ip_address || '-' },
  { header: 'Host Name', cell: (host) => host.host_name || '-' },
  { header: 'MAC Address', cell: (host) => host.mac_address || '-' },
  { header: 'Address Source', cell: (host) => host.address_source || '-' },
  { header: 'Interface Type', cell: (host) => host.interface_type || '-' },
  { header: 'Layer 2 Interface', cell: (host) => host.layer2_interface || '-' },
  { header: 'Active', cell: (host) => host.active || '-' },
]

const wlanColumns: DataTableColumn<GenieACSWLAN>[] = [
  { header: 'Enabled', cell: (wlan) => wlan.enable || '-' },
  {
    header: 'Broadcast',
    cell: (wlan) => wlan.ssid_advertisement_enabled || '-',
  },
  { header: 'SSID', cell: (wlan) => wlan.ssid || '-' },
  {
    header: 'Security',
    cell: (wlan) =>
      beaconTypeLabels[wlan.beacon_type] ?? (wlan.beacon_type || '-'),
  },
  { header: 'Password', cell: (wlan) => wlan.key_passphrase || '-' },
  {
    header: 'Channel',
    cell: (wlan) => {
      const channel = wlan.channel && wlan.channel !== '0' ? wlan.channel : ''
      if (wlan.auto_channel_enable === 'true')
        return channel ? `Auto (${channel})` : 'Auto'
      return wlan.channel || '-'
    },
  },
  {
    header: 'Width',
    cell: (wlan) => (wlan.channel_width ? `${wlan.channel_width} MHz` : '-'),
  },
]

const BYTE_UNITS = ['B', 'KB', 'MB', 'GB', 'TB']

function parseBytes(raw?: string | null) {
  const text = (raw ?? '').trim()
  const value = Number(text)
  return text && Number.isFinite(value) && value >= 0 ? value : null
}

function formatBytes(bytes: number) {
  let value = bytes
  let unit = 0
  while (value >= 1024 && unit < BYTE_UNITS.length - 1) {
    value /= 1024
    unit++
  }
  const digits = unit === 0 || value >= 100 ? 0 : value >= 10 ? 1 : 2
  return `${value.toFixed(digits)} ${BYTE_UNITS[unit]}`
}

function formatUsage(device: GenieACSWLANAssociatedDevice) {
  const received = parseBytes(device.bytes_received)
  const sent = parseBytes(device.bytes_sent)
  if (received === null && sent === null) return '-'
  return formatBytes((received ?? 0) + (sent ?? 0))
}

// Non-numeric values (e.g. a timestamp string from some vendors) are shown as reported.
function formatDuration(raw?: string | null) {
  const text = (raw ?? '').trim()
  const total = Number(text)
  if (!text) return '-'
  if (!Number.isFinite(total) || total < 0) return text

  const seconds = Math.floor(total)
  const days = Math.floor(seconds / 86400)
  const pad = (value: number) => String(value).padStart(2, '0')
  const clock = [
    Math.floor((seconds % 86400) / 3600),
    Math.floor((seconds % 3600) / 60),
    seconds % 60,
  ]
    .map(pad)
    .join(':')
  return days > 0 ? `${days}d ${clock}` : clock
}

const associatedDeviceColumns: DataTableColumn<GenieACSWLANAssociatedDevice>[] =
  [
    { header: 'Device Name', cell: (device) => device.name || '-' },
    { header: 'MAC Address', cell: (device) => device.mac_address || '-' },
    { header: 'IP Address', cell: (device) => device.ip_address || '-' },
    { header: 'RSSI', cell: (device) => device.rssi || '-' },
    { header: 'Bandwidth', cell: (device) => device.bandwidth || '-' },
    { header: 'Mode', cell: (device) => device.mode || '-' },
    { header: 'Radio', cell: (device) => device.radio || '-' },
    { header: 'Stay Time', cell: (device) => formatDuration(device.stay_time) },
    { header: 'Usage', cell: formatUsage },
  ]

const vlanDisabled = { key: 'enable_vlan', equals: 'false' }

const wanHeadFields: ParameterField[] = [
  { key: 'enable', label: 'Enable', type: 'boolean' },
  { key: 'conn_name', label: 'Name' },
  { key: 'enable_vlan', label: 'Enable VLAN', type: 'boolean' },
  { key: 'vlan', label: 'VLAN', type: 'number', hideWhen: vlanDisabled },
  { key: 'conn_type', label: 'Conn Type' },
  { key: 'service', label: 'Service' },
]

const bridgedPPPoE = { key: 'conn_type', equals: 'PPPoE_Bridged' }

const wanCredentialFields: ParameterField[] = [
  { key: 'username', label: 'Username', hideWhen: bridgedPPPoE },
  {
    key: 'password',
    label: 'Password',
    type: 'password',
    hideWhen: bridgedPPPoE,
  },
]

const wanTailFields: ParameterField[] = [
  { key: 'nat', label: 'NAT', type: 'boolean' },
  { key: 'lan_bind', label: 'LanBind', type: 'checklist' },
]

const ipoeFields = [...wanHeadFields, ...wanTailFields]
const pppoeFields = [...wanHeadFields, ...wanCredentialFields, ...wanTailFields]

const dhcpFields: Record<string, ParameterField> = {
  server_enable: {
    key: 'server_enable',
    label: 'DHCP Server',
    type: 'boolean',
  },
  min_address: { key: 'min_address', label: 'Min Address' },
  max_address: { key: 'max_address', label: 'Max Address' },
  subnet_mask: { key: 'subnet_mask', label: 'Subnet Mask' },
  ip_routers: { key: 'ip_routers', label: 'IP Routers' },
  dns_servers: { key: 'dns_servers', label: 'DNS Servers' },
}

const portFields: ParameterField[] = [
  { key: 'enable', label: 'Enabled', type: 'boolean' },
  { key: 'l3_enable', label: 'L3 Enable', type: 'boolean' },
]

const beaconTypeLabels: Record<string, string> = {
  Basic: 'Open',
  None: 'Open',
  WPA: 'WPA',
  '11i': 'WPA2',
  WPAand11i: 'WPA/WPA2',
}

// Open is Basic or None depending on vendor; show the one the dropdown offers so it is not listed twice.
function normalizeBeaconType(value: string, options: string[]) {
  const open = options.find((option) => beaconTypeLabels[option] === 'Open')
  return open && beaconTypeLabels[value] === 'Open' ? open : value
}

const channelWidthLabels: Record<string, string> = {
  '20': '20 MHz',
  '20/40': '20/40 MHz',
  '40': '40 MHz',
  '80': '80 MHz',
}

// Applies when the device reports no stricter password rule.
const MIN_WLAN_PASSWORD_LENGTH = 8

const wlanBasicKeys = [
  'enable',
  'ssid_advertisement_enabled',
  'ssid',
  'beacon_type',
  'key_passphrase',
]

const wlanFields: ParameterField[] = [
  { key: 'enable', label: 'Enabled', type: 'boolean' },
  {
    key: 'ssid_advertisement_enabled',
    label: ' Broadcast SSID',
    type: 'boolean',
  },
  { key: 'ssid', label: 'SSID' },
  { key: 'auto_channel_enable', label: 'Auto Channel', type: 'boolean' },
  {
    key: 'channel',
    label: 'Channel',
    type: 'number',
    disabledWhen: { key: 'auto_channel_enable', equals: 'true' },
  },
  { key: 'standard', label: 'Standard' },
  { key: 'channel_width', label: 'Channel Width', type: 'select' },
  { key: 'max_clients', label: 'Max Clients', type: 'number' },
  { key: 'beacon_type', label: 'Security' },
  { key: 'key_passphrase', label: 'Password', type: 'password' },
]

type DhcpRow = GenieACSLAN['dhcp']

const dhcpColumns: DataTableColumn<DhcpRow>[] = Object.entries(dhcpFields).map(
  ([key, field]) => ({
    header: field.label,
    cell: (row) => row[key as keyof GenieACSLAN['dhcp']] || '-',
  })
)

function EditButton({
  label,
  onClick,
  disabled,
}: {
  label: string
  onClick: () => void
  disabled?: boolean
}) {
  return (
    <Button
      variant='ghost'
      size='icon'
      className='h-7 w-7'
      title='Edit'
      aria-label={label}
      disabled={disabled}
      onClick={onClick}
    >
      <Pencil className='size-3.5' />
    </Button>
  )
}

function stringValues(source: object) {
  return Object.fromEntries(
    Object.entries(source).filter(([, value]) => typeof value === 'string')
  ) as Record<string, string>
}

function DetailTable<T extends object>({
  title,
  rows,
  columns,
  getRowId,
  actions,
  onSummon,
  summonLabel,
  isSummoning,
  onAdd,
  addLabel,
  isAdding,
  isBusy,
}: {
  title: string
  rows: T[]
  columns: DataTableColumn<T>[]
  getRowId: (row: T) => string | number
  actions?: (row: T) => ReactNode
  onSummon?: () => void
  summonLabel?: string
  isSummoning?: boolean
  onAdd?: () => void
  addLabel?: string
  isAdding?: boolean
  isBusy?: boolean
}) {
  const label = `Summon ${summonLabel ?? title}`
  return (
    <section className='space-y-2'>
      <div className='flex items-center gap-3'>
        <h2 className='text-lg font-semibold'>{title}</h2>
        {onSummon && (
          <Button
            size='sm'
            className='h-7 gap-2'
            title={label}
            aria-label={label}
            disabled={isBusy}
            onClick={onSummon}
          >
            {isSummoning ? (
              <Loader2 className='size-3.5 animate-spin' />
            ) : (
              <RefreshCw className='size-3.5' />
            )}
            Summon
          </Button>
        )}
      </div>
      <MuiDataTable
        columns={columns}
        rows={rows}
        getRowId={getRowId}
        actions={actions}
        toolbarBeforeAdd={
          onAdd && (
            <Button
              size='sm'
              className='h-7 w-full gap-2 sm:w-auto'
              title={addLabel}
              aria-label={addLabel}
              disabled={isBusy}
              onClick={onAdd}
            >
              {isAdding ? (
                <Loader2 className='size-3.5 animate-spin' />
              ) : (
                <Plus className='size-3.5' />
              )}
              {addLabel}
            </Button>
          )
        }
      />
    </section>
  )
}

export function GenieACSDeviceDetail() {
  const { deviceId } = useParams({
    from: '/_authenticated/genieacs/devices/$deviceId',
  })
  const { configId } = useSearch({
    from: '/_authenticated/genieacs/devices/$deviceId',
  })
  return (
    <GenieACSDeviceDetailContent
      key={`${configId}:${deviceId}`}
      configId={configId}
      deviceId={deviceId}
    />
  )
}

function GenieACSDeviceDetailContent({
  configId,
  deviceId,
}: {
  configId: number
  deviceId: string
}) {
  const [wan, setWan] = useState<GenieACSWAN[]>([])
  const [lan, setLan] = useState<GenieACSLAN | null>(null)
  const [wlan, setWlan] = useState<GenieACSWLAN[]>([])
  const [summoningScope, setSummoningScope] = useState<Scope | null>(null)
  const [activeTab, setActiveTab] = useState('wan')
  const [editTarget, setEditTarget] = useState<EditTarget | null>(null)
  const [isSaving, setIsSaving] = useState(false)
  const [deleteTarget, setDeleteTarget] = useState<GenieACSWAN | null>(null)
  const [isDeleting, setIsDeleting] = useState(false)
  const [addingKind, setAddingKind] = useState<string | null>(null)
  const requestId = useRef(0)
  const muiTheme = useMuiTheme()
  const navigate = useNavigate()
  const isBusy =
    summoningScope !== null || isSaving || isDeleting || addingKind !== null

  // Only the latest request may touch state, so stale responses are dropped.
  const loadTab = useCallback(
    async (tab: string) => {
      const id = ++requestId.current
      const isLatest = () => id === requestId.current
      try {
        if (tab === 'wan') {
          const data = await getGenieACSDeviceWAN(configId, deviceId)
          if (isLatest()) setWan(data ?? [])
        }
        if (tab === 'lan') {
          const data = await getGenieACSDeviceLAN(configId, deviceId)
          if (isLatest()) setLan(data ?? null)
        }
        if (tab === 'wlan') {
          const data = await getGenieACSDeviceWLAN(configId, deviceId)
          if (isLatest()) setWlan(data ?? [])
        }
      } catch (error) {
        if (isLatest()) toast.error(getAuthErrorMessage(error))
      }
    },
    [configId, deviceId]
  )

  // Reload only when the device applied the task; a queued task has nothing new to show yet.
  async function summonScope(scope: Scope) {
    if (summoningScope) return
    setSummoningScope(scope)
    try {
      const result = await summonGenieACSDevice(configId, deviceId, scope)
      if (result.applied) {
        toast.success(result.message)
        await loadTab(scope)
      } else {
        toast.error(result.message)
      }
    } catch (error) {
      toast.error(getAuthErrorMessage(error))
    } finally {
      setSummoningScope(null)
    }
  }

  function openEdit(
    title: string,
    endpoint: EditTarget['endpoint'],
    path: string,
    fields: ParameterField[],
    values: Record<string, string>,
    basicKeys?: string[]
  ) {
    setEditTarget({ title, endpoint, path, fields, values, basicKeys })
  }

  async function saveEdit(changed: Record<string, string>) {
    if (!editTarget || isSaving) return
    setIsSaving(true)
    try {
      const result = await updateGenieACSParameters(
        configId,
        deviceId,
        editTarget.endpoint,
        editTarget.path,
        changed
      )
      const scope = editTarget.endpoint.split('/')[0]
      setEditTarget(null)
      if (result.applied) {
        toast.success(result.message)
        await loadTab(scope)
      } else {
        toast.info(result.message)
      }
    } catch (error) {
      toast.error(getAuthErrorMessage(error))
    } finally {
      setIsSaving(false)
    }
  }

  async function addWan(kind: typeof IPOE_KIND | typeof PPPOE_KIND) {
    if (isBusy) return
    setAddingKind(kind)
    try {
      const result = await addGenieACSWAN(configId, deviceId, kind)
      if (result.applied) {
        toast.success(result.message)
        await loadTab('wan')
      } else {
        toast.info(result.message)
      }
    } catch (error) {
      toast.error(getAuthErrorMessage(error))
    } finally {
      setAddingKind(null)
    }
  }

  async function confirmDeleteWan() {
    if (!deleteTarget || isDeleting) return
    setIsDeleting(true)
    try {
      const result = await deleteGenieACSWAN(
        configId,
        deviceId,
        deleteTarget.path
      )
      setDeleteTarget(null)
      if (result.applied) {
        toast.success(result.message)
        await loadTab('wan')
      } else {
        toast.info(result.message)
      }
    } catch (error) {
      toast.error(getAuthErrorMessage(error))
    } finally {
      setIsDeleting(false)
    }
  }

  useEffect(() => {
    const id = ++requestId.current
    const isLatest = () => id === requestId.current
    const loadActiveTab = async () => {
      try {
        if (activeTab === 'wan') {
          const data = await getGenieACSDeviceWAN(configId, deviceId)
          if (isLatest()) setWan(data ?? [])
        } else if (activeTab === 'lan') {
          const data = await getGenieACSDeviceLAN(configId, deviceId)
          if (isLatest()) setLan(data ?? null)
        } else if (activeTab === 'wlan') {
          const data = await getGenieACSDeviceWLAN(configId, deviceId)
          if (isLatest()) setWlan(data ?? [])
        }
      } catch (error) {
        if (isLatest()) toast.error(getAuthErrorMessage(error))
      }
    }
    void loadActiveTab()
    return () => {
      if (isLatest()) requestId.current += 1
    }
  }, [activeTab, configId, deviceId])

  const ipoeRows = useMemo(
    () => wan.filter((row) => row.kind === IPOE_KIND),
    [wan]
  )
  const pppoeRows = useMemo(
    () => wan.filter((row) => row.kind === PPPOE_KIND),
    [wan]
  )

  const wanActions = (fields: ParameterField[], label: string) =>
    function renderWanActions(row: GenieACSWAN) {
      const name = row.conn_name || row.path
      return (
        <div className='flex items-center'>
          <EditButton
            label={`Edit ${name}`}
            disabled={isBusy}
            onClick={() =>
              openEdit(
                row.conn_name
                  ? `Edit ${label} - ${row.conn_name}`
                  : `Edit ${label}`,
                'wan',
                row.path,
                fields
                  .filter(
                    (field) =>
                      field.key !== 'lan_bind' ||
                      (row.lan_bind_options?.length ?? 0) > 0
                  )
                  .map((field) => {
                    if (field.key === 'conn_name' && row.name_locked)
                      return { ...field, readOnly: true }
                    if (field.key === 'lan_bind')
                      return {
                        ...field,
                        options: row.lan_bind_options ?? [],
                        disabledOptions: row.lan_bind_disabled ?? [],
                      }
                    const choices =
                      field.key === 'conn_type'
                        ? row.conn_type_options
                        : field.key === 'service'
                          ? row.service_options
                          : null
                    if (choices?.length)
                      return {
                        ...field,
                        type: 'select' as const,
                        options: choices,
                      }
                    return field
                  }),
                {
                  ...stringValues(row),
                  enable_vlan: String(row.enable_vlan),
                  lan_bind: lanBindLabels(row.lan_bind).join(','),
                }
              )
            }
          />
          <Button
            variant='ghost'
            size='icon'
            className='h-7 w-7 text-destructive hover:text-destructive'
            title='Delete'
            aria-label={`Delete ${name}`}
            disabled={isBusy}
            onClick={() => setDeleteTarget(row)}
          >
            <Trash2 className='size-3.5' />
          </Button>
        </div>
      )
    }

  const lanEntries = useMemo<DhcpRow[]>(() => (lan ? [lan.dhcp] : []), [lan])

  return (
    <>
      <LoadingOverlay open={summoningScope !== null} />
      <Header fixed>
        <div className='ms-auto flex items-center gap-2'>
          <ThemeSwitch />
          <ProfileDropdown />
        </div>
      </Header>
      <Main fixed fluid className='min-h-0 !overflow-y-auto'>
        <div className='mb-3 flex items-start gap-3'>
          <Button
            variant='outline'
            size='icon'
            aria-label='Back to devices'
            title='Back to devices'
            onClick={() =>
              void navigate({
                to: '/genieacs/devices',
                search: { configId },
              })
            }
          >
            <ArrowLeft className='size-4' />
          </Button>
          <div>
            <h1 className='text-2xl font-bold tracking-tight'>
              Device Details
            </h1>
            <p className='text-sm text-muted-foreground'>{deviceId}</p>
          </div>
        </div>
        <ThemeProvider theme={muiTheme}>
          <Tabs value={activeTab} onValueChange={setActiveTab}>
            <TabsList className='mb-3'>
              <TabsTrigger value='wan'>WAN</TabsTrigger>
              <TabsTrigger value='lan'>LAN</TabsTrigger>
              <TabsTrigger value='wlan'>WLAN</TabsTrigger>
            </TabsList>
            <TabsContent value='wan'>
              <div className='space-y-6'>
                <DetailTable
                  title='WAN IPoE'
                  rows={ipoeRows}
                  columns={ipoeColumns}
                  getRowId={(row) => row.path}
                  actions={wanActions(ipoeFields, 'IPoE')}
                  onSummon={() => void summonScope('wan')}
                  summonLabel='WAN'
                  isSummoning={summoningScope === 'wan'}
                  onAdd={() => void addWan(IPOE_KIND)}
                  addLabel='Add IPoE'
                  isAdding={addingKind === IPOE_KIND}
                  isBusy={isBusy}
                />
                <DetailTable
                  title='WAN PPPoE'
                  rows={pppoeRows}
                  columns={pppoeColumns}
                  getRowId={(row) => row.path}
                  actions={wanActions(pppoeFields, 'PPPoE')}
                  onAdd={() => void addWan(PPPOE_KIND)}
                  addLabel='Add PPPoE'
                  isAdding={addingKind === PPPOE_KIND}
                  isBusy={isBusy}
                />
              </div>
            </TabsContent>
            <TabsContent value='lan'>
              <div className='space-y-6'>
                <DetailTable
                  title='LAN Configuration'
                  rows={lanEntries}
                  columns={dhcpColumns}
                  getRowId={() => 'dhcp'}
                  actions={(row) => (
                    <EditButton
                      label='Edit LAN configuration'
                      disabled={isBusy}
                      onClick={() =>
                        openEdit(
                          'Edit LAN Configuration',
                          'lan/dhcp',
                          '',
                          Object.values(dhcpFields),
                          stringValues(row)
                        )
                      }
                    />
                  )}
                  onSummon={() => void summonScope('lan')}
                  summonLabel='LAN'
                  isSummoning={summoningScope === 'lan'}
                  isBusy={isBusy}
                />
                <DetailTable
                  title='LAN Ports'
                  rows={lan?.ports ?? []}
                  columns={portColumns}
                  getRowId={(row) => row.path}
                  actions={(row) => (
                    <EditButton
                      label='Edit port'
                      disabled={isBusy}
                      onClick={() =>
                        openEdit(
                          `Edit LAN Port ${row.path.split('.').pop()}`,
                          'lan/ports',
                          row.path,
                          portFields,
                          stringValues(row)
                        )
                      }
                    />
                  )}
                />
                <DetailTable
                  title='LAN Hosts'
                  rows={lan?.hosts ?? []}
                  columns={hostColumns}
                  getRowId={(row) => row.path}
                />
              </div>
            </TabsContent>
            <TabsContent value='wlan'>
              <div className='space-y-6'>
                <DetailTable
                  title='Wireless Networks'
                  rows={wlan}
                  columns={wlanColumns}
                  getRowId={(row) => row.path}
                  actions={(row) => (
                    <EditButton
                      label={`Edit ${row.ssid || `WLAN ${row.index}`}`}
                      disabled={isBusy}
                      onClick={() =>
                        openEdit(
                          `Edit WLAN - ${row.ssid || `WLAN ${row.index}`}`,
                          'wlan',
                          row.path,
                          wlanFields
                            .filter(
                              (field) =>
                                (field.key !== 'channel_width' ||
                                  row.channel_width_options?.length) &&
                                (field.key !== 'max_clients' || row.max_clients)
                            )
                            .map((field) => {
                              if (field.key === 'channel_width')
                                return {
                                  ...field,
                                  options: row.channel_width_options ?? [],
                                  optionLabels: channelWidthLabels,
                                  optionsBy: row.channel_width_by_standard
                                    ? {
                                        key: 'standard',
                                        map: row.channel_width_by_standard,
                                      }
                                    : undefined,
                                }
                              if (
                                field.key === 'standard' &&
                                row.standard_options?.length
                              )
                                return {
                                  ...field,
                                  type: 'select' as const,
                                  options: row.standard_options,
                                }
                              if (field.key === 'key_passphrase')
                                return {
                                  ...field,
                                  passwordPolicy: row.password_policy ?? {
                                    min_length: MIN_WLAN_PASSWORD_LENGTH,
                                  },
                                }
                              if (field.key === 'beacon_type')
                                return {
                                  ...field,
                                  type: 'select' as const,
                                  options: row.beacon_type_options ?? [],
                                  optionLabels: beaconTypeLabels,
                                }
                              if (
                                field.key === 'channel' &&
                                row.channel_options?.length
                              )
                                return {
                                  ...field,
                                  type: 'select' as const,
                                  options: row.channel_options,
                                }
                              return field
                            }),
                          {
                            ...stringValues(row),
                            beacon_type: normalizeBeaconType(
                              row.beacon_type,
                              row.beacon_type_options ?? []
                            ),
                          },
                          wlanBasicKeys
                        )
                      }
                    />
                  )}
                  onSummon={() => void summonScope('wlan')}
                  summonLabel='WLAN'
                  isSummoning={summoningScope === 'wlan'}
                  isBusy={isBusy}
                />
                {wlan
                  .filter(
                    (network) => (network.associated_devices ?? []).length > 0
                  )
                  .map((network) => (
                    <DetailTable
                      key={network.path}
                      title={`Associated Devices - ${network.ssid || `WLAN ${network.index}`}`}
                      rows={network.associated_devices ?? []}
                      columns={associatedDeviceColumns}
                      getRowId={(row) => row.path}
                    />
                  ))}
              </div>
            </TabsContent>
          </Tabs>
        </ThemeProvider>
        <EditParametersDialog
          target={editTarget}
          isSaving={isSaving}
          onClose={() => setEditTarget(null)}
          onSubmit={(changed) => void saveEdit(changed)}
        />
        <ConfirmDialog
          open={deleteTarget !== null}
          onOpenChange={(open) => !open && setDeleteTarget(null)}
          title='Delete WAN instance'
          desc={`Delete ${deleteTarget?.conn_name ? `"${deleteTarget.conn_name}"` : 'this WAN connection'}? The device will remove it.`}
          confirmText='Delete'
          destructive
          isLoading={isDeleting}
          handleConfirm={() => void confirmDeleteWan()}
        />
      </Main>
    </>
  )
}
