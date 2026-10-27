import { useCallback, useEffect, useRef, useState } from 'react'
import { ThemeProvider } from '@mui/material'
import {
  Laptop,
  Layers,
  Pencil,
  Power,
  PowerOff,
  Trash2,
  Users,
  Wifi,
} from 'lucide-react'
import { toast } from 'sonner'
import { runBulk } from '@/lib/run-bulk'
import { cn } from '@/lib/utils'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Tabs, TabsContent } from '@/components/ui/tabs'
import {
  MuiDataTable as DataTable,
  type DataTableColumn as Column,
} from '@/components/data-table'
import { Header } from '@/components/layout/header'
import { Main } from '@/components/layout/main'
import { ProfileDropdown } from '@/components/profile-dropdown'
import { StatCard } from '@/components/stat-card'
import { ThemeSwitch } from '@/components/theme-switch'
import { FormDialog, type Field, type FormTab } from '../components/form-dialog'
import { IconButton } from '../components/icon-button'
import { applyDelta, useLiveStream } from '../hooks/use-live-stream'
import { useMuiTheme } from '../hooks/use-mui-theme'
import { getMikrotikRouters, type MikrotikRouter } from '../router/mikrotik-api'
import {
  deleteHotspotHost,
  deleteHotspotProfile,
  deleteHotspotUser,
  getHotspotActive,
  getHotspotErrorMessage,
  getHotspotHosts,
  getHotspotProfiles,
  getHotspotServers,
  getHotspotPools,
  getHotspotQueues,
  getHotspotQueueTypes,
  getHotspotUsers,
  kickHotspotActive,
  saveHotspotProfile,
  saveHotspotUser,
  setHotspotUserDisabled,
  type HotspotActive,
  type HotspotHost,
  type HotspotLive,
  type HotspotProfile,
  type HotspotUser,
} from './hotspot-api'

type HotspotView = 'user' | 'profile' | 'active' | 'host'

function percent(part: number, total: number) {
  return total === 0 ? '0%' : `${Math.round((part / total) * 100)}%`
}

// Live updates arrive over one stream; the full refresh only repairs drift and reloads profiles.
const HOTSPOT_LIST_POLL_INTERVAL_MS = 30_000

// Form keys match the API keys so no mapping table is needed.
const userKeys = [
  'server',
  'name',
  'password',
  'address',
  'mac_address',
  'profile',
  'comment',
  'limit_uptime',
  'limit_bytes_in',
  'limit_bytes_out',
  'limit_bytes_total',
] as const

const userGeneralKeys = [
  'server',
  'name',
  'password',
  'address',
  'mac_address',
  'profile',
  'comment',
]
const userLimitKeys = [
  'limit_uptime',
  'limit_bytes_in',
  'limit_bytes_out',
  'limit_bytes_total',
]
const userTabs = [
  { value: 'general', label: 'General', keys: userGeneralKeys },
  { value: 'limits', label: 'Limits', keys: userLimitKeys },
]

const profileKeys = [
  'name',
  'address_pool',
  'shared_users',
  'rate_limit',
  'session_timeout',
  'idle_timeout',
  'keepalive_timeout',
  'status_autorefresh',
  'add_mac_cookie',
  'mac_cookie_timeout',
  'address_list',
  'insert_queue_before',
  'parent_queue',
  'queue_type',
  'transparent_proxy',
  'on_login',
  'on_logout',
  'comment',
] as const

const profileGeneralKeys = [
  'name',
  'address_pool',
  'shared_users',
  'rate_limit',
  'add_mac_cookie',
  'mac_cookie_timeout',
  'transparent_proxy',
  'comment',
]
const profileQueueKeys = ['insert_queue_before', 'parent_queue', 'queue_type']
const profileScriptKeys = ['on_login', 'on_logout']

const profileTabs: FormTab[] = [
  { value: 'general', label: 'General', keys: profileGeneralKeys },
  { value: 'queue', label: 'Queue', keys: profileQueueKeys },
  { value: 'scripts', label: 'Scripts', keys: profileScriptKeys },
]

type UserForm = Record<(typeof userKeys)[number], string>
type ProfileForm = Record<(typeof profileKeys)[number], string>

interface ConfirmState {
  title: string
  description: string
  actionLabel: string
  destructive?: boolean
  onConfirm: () => Promise<void>
}

const emptyUser = Object.fromEntries(
  userKeys.map((key) => [key, ''])
) as UserForm

const emptyProfile = Object.fromEntries(
  profileKeys.map((key) => [key, ''])
) as ProfileForm

// RouterOS tidak menampilkan address-pool bila tidak diset, di form ditampilkan sebagai "none".
function profileToForm(profile: HotspotProfile): ProfileForm {
  const form = pickForm(profileKeys, profile)
  return { ...form, address_pool: form.address_pool || 'none' }
}

function formatBytes(value: string) {
  const bytes = Number(value)
  if (!Number.isFinite(bytes) || bytes <= 0) return '-'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  const exponent = Math.min(
    Math.floor(Math.log(bytes) / Math.log(1024)),
    units.length - 1
  )
  const scaled = bytes / 1024 ** exponent
  return `${scaled.toFixed(exponent === 0 ? 0 : 2)} ${units[exponent]}`
}

type Rated<T> = T & { tx_rate: number | null; rx_rate: number | null }

// Rate (bit/detik) dihitung server dan datang lewat stream; daftar penuh tidak membawanya,
// jadi nilai yang sudah ada dipertahankan per id. TX = bytes-out, RX = bytes-in.
function keepRates<T extends { id: string }>(
  rows: T[],
  current: Rated<T>[]
): Rated<T>[] {
  const known = new Map(current.map((row) => [row.id, row]))
  return rows.map((row) => ({
    ...row,
    tx_rate: known.get(row.id)?.tx_rate ?? null,
    rx_rate: known.get(row.id)?.rx_rate ?? null,
  }))
}

function formatRate(value: number | null) {
  if (value === null) return '-'
  const units = ['bps', 'kbps', 'Mbps', 'Gbps']
  const exponent = Math.min(
    value >= 1 ? Math.floor(Math.log(value) / Math.log(1000)) : 0,
    units.length - 1
  )
  const scaled = value / 1000 ** exponent
  return `${scaled.toFixed(exponent === 0 ? 0 : 2)} ${units[exponent]}`
}

function rateColumns<T extends Rated<unknown>>(): Column<T>[] {
  return [
    {
      header: 'TX Rate',
      cell: (row) => formatRate(row.tx_rate),
      getSortValue: (row) => row.tx_rate,
    },
    {
      header: 'RX Rate',
      cell: (row) => formatRate(row.rx_rate),
      getSortValue: (row) => row.rx_rate,
    },
  ]
}

function bytesColumns<
  T extends { bytes_in: string; bytes_out: string },
>(): Column<T>[] {
  return [
    {
      header: 'Bytes In',
      cell: (row) => formatBytes(row.bytes_in),
      getSortValue: (row) => Number(row.bytes_in) || 0,
    },
    {
      header: 'Bytes Out',
      cell: (row) => formatBytes(row.bytes_out),
      getSortValue: (row) => Number(row.bytes_out) || 0,
    },
  ]
}

// Shows the stored byte count as 10G / 500M when it divides evenly, so edit matches how users type it.
function bytesToInput(value: string) {
  const bytes = Number(value)
  if (!Number.isSafeInteger(bytes) || bytes <= 0) return value ?? ''
  const units = ['', 'K', 'M', 'G', 'T']
  let exponent = 0
  while (exponent < units.length - 1 && bytes % 1024 ** (exponent + 1) === 0) {
    exponent++
  }
  return `${bytes / 1024 ** exponent}${units[exponent]}`
}

function userToForm(user: HotspotUser): UserForm {
  return {
    ...pickForm(userKeys, user),
    limit_bytes_in: bytesToInput(user.limit_bytes_in),
    limit_bytes_out: bytesToInput(user.limit_bytes_out),
    limit_bytes_total: bytesToInput(user.limit_bytes_total),
  }
}

// Add sends only filled fields; edit sends only fields that differ (empty clears the value).
// A blank password on edit is ignored: hotspot users cannot be left without a password.
function changedInput<K extends string>(
  keys: readonly K[],
  form: Record<K, string>,
  original?: Record<K, string>
) {
  const input: Partial<Record<K, string>> = {}
  for (const key of keys) {
    const value = (form[key] ?? '').trim()
    if (original) {
      if (key === 'password' && value === '') continue
      if (value !== (original[key] ?? '')) input[key] = value
    } else if (value !== '') {
      input[key] = value
    }
  }
  return input
}

function pickForm<K extends string>(
  keys: readonly K[],
  source: Record<K, string>
) {
  return Object.fromEntries(
    keys.map((key) => [key, source[key] ?? ''])
  ) as Record<K, string>
}

// User disabled: flag X di kolom pertama dan seluruh teks baris dibuat abu-abu samar, seperti Simple Queue.
function dimDisabled(columns: Column<HotspotUser>[]): Column<HotspotUser>[] {
  return columns.map((column) => ({
    ...column,
    cell: (row) => {
      const content = column.cell(row)
      if (!content) return content
      return (
        <span className={cn(row.disabled && 'text-muted-foreground/60')}>
          {content}
        </span>
      )
    },
  }))
}

const userColumns: Column<HotspotUser>[] = dimDisabled([
  {
    header: 'Flags',
    align: 'center',
    hideHeader: true,
    narrow: true,
    cell: (row) =>
      row.disabled ? (
        <span title='Disabled' className='font-semibold'>
          X
        </span>
      ) : null,
  },
  {
    header: 'Name',
    cell: (row) => <span className='font-semibold'>{row.name}</span>,
    getSortValue: (row) => row.name,
  },
  { header: 'Password', cell: (row) => row.password || '-' },
  { header: 'Profile', cell: (row) => row.profile || '-' },
  { header: 'Server', cell: (row) => row.server || '-' },
  { header: 'Uptime', cell: (row) => row.uptime || '-' },
  {
    header: 'Bytes In',
    cell: (row) => formatBytes(row.bytes_in),
    getSortValue: (row) => Number(row.bytes_in) || 0,
  },
  {
    header: 'Bytes Out',
    cell: (row) => formatBytes(row.bytes_out),
    getSortValue: (row) => Number(row.bytes_out) || 0,
  },
  { header: 'Comment', cell: (row) => row.comment || '-' },
])

const profileColumns: Column<HotspotProfile>[] = [
  {
    header: 'Name',
    cell: (row) => <span className='font-semibold'>{row.name}</span>,
    getSortValue: (row) => row.name,
  },
  { header: 'Rate Limit', cell: (row) => row.rate_limit || '-' },
  { header: 'Shared Users', cell: (row) => row.shared_users || '-' },
  {
    header: 'Insert Queue Before',
    cell: (row) => row.insert_queue_before || '-',
  },
  { header: 'Parent Queue', cell: (row) => row.parent_queue || '-' },
]

const activeColumns: Column<Rated<HotspotActive>>[] = [
  {
    header: 'User',
    cell: (row) => <span className='font-semibold'>{row.user}</span>,
    getSortValue: (row) => row.user,
  },
  { header: 'Address', cell: (row) => row.address || '-' },
  { header: 'MAC Address', cell: (row) => row.mac_address || '-' },
  { header: 'Server', cell: (row) => row.server || '-' },
  { header: 'Login By', cell: (row) => row.login_by || '-' },
  { header: 'Uptime', cell: (row) => row.uptime || '-' },
  { header: 'Time Left', cell: (row) => row.session_time_left || '-' },
  ...rateColumns<Rated<HotspotActive>>(),
  ...bytesColumns<HotspotActive>(),
]

const hostColumns: Column<Rated<HotspotHost>>[] = [
  {
    // Flag seperti Winbox: A = authorized, P = bypassed, H = DHCP, D = dynamic.
    header: 'Flags',
    align: 'center',
    hideHeader: true,
    narrow: true,
    cell: (row) => {
      const flags = [
        row.authorized && ['A', 'Authorized'],
        row.bypassed && ['P', 'Bypassed'],
        row.dhcp && ['H', 'DHCP'],
        row.dynamic && ['D', 'Dynamic'],
      ].filter((flag) => flag !== false)
      if (flags.length === 0) return null
      return (
        <span
          title={flags.map(([, name]) => name).join(', ')}
          className='font-semibold'
        >
          {flags.map(([letter]) => letter).join('')}
        </span>
      )
    },
  },
  {
    header: 'MAC Address',
    cell: (row) => <span className='font-semibold'>{row.mac_address}</span>,
    getSortValue: (row) => row.mac_address,
  },
  { header: 'Address', cell: (row) => row.address || '-' },
  { header: 'To Address', cell: (row) => row.to_address || '-' },
  { header: 'Server', cell: (row) => row.server || '-' },
  ...rateColumns<Rated<HotspotHost>>(),
  ...bytesColumns<HotspotHost>(),
  { header: 'Comment', cell: (row) => row.comment || '-' },
]

const userFields: Field<UserForm>[] = [
  {
    key: 'server',
    label: 'Server',
    placeholder: 'Select server',
    required: true,
  },
  { key: 'name', label: 'Name', placeholder: 'Username', required: true },
  {
    key: 'password',
    label: 'Password',
    placeholder: 'Password',
    required: true,
  },
  { key: 'address', label: 'Address', placeholder: '0.0.0.0' },
  {
    key: 'mac_address',
    label: 'MAC Address',
    placeholder: '00:00:00:00:00:00',
  },
  {
    key: 'profile',
    label: 'Profile',
    placeholder: 'Select profile',
    required: true,
  },
  { key: 'comment', label: 'Comment', placeholder: 'Comment' },
  {
    key: 'limit_uptime',
    label: 'Limit Uptime',
    placeholder: 'e.g. 30m, 1h, 1d, 1w, 1mo',
  },
  {
    key: 'limit_bytes_in',
    label: 'Limit Bytes In',
    placeholder: 'e.g. 500M, 10G',
  },
  {
    key: 'limit_bytes_out',
    label: 'Limit Bytes Out',
    placeholder: 'e.g. 500M, 10G',
  },
  {
    key: 'limit_bytes_total',
    label: 'Limit Bytes Total',
    placeholder: 'e.g. 500M, 10G',
  },
]

const profileFields: Field<ProfileForm>[] = [
  { key: 'name', label: 'Name', placeholder: 'Profile Name', required: true },
  {
    key: 'address_pool',
    label: 'Address Pool',
    placeholder: 'Select pool',
  },
  { key: 'shared_users', label: 'Shared Users', placeholder: '1' },
  {
    key: 'rate_limit',
    label: 'Rate Limit (rx/tx)',
    placeholder: '5M/5M',
    type: 'rate',
  },
  {
    key: 'add_mac_cookie',
    label: 'Add MAC Cookie',
    placeholder: '',
    type: 'checkbox',
  },
  {
    key: 'mac_cookie_timeout',
    label: 'MAC Cookie Timeout',
    placeholder: '3d 00:00:00',
    disabled: (data) => data.add_mac_cookie !== 'yes',
  },
  {
    key: 'transparent_proxy',
    label: 'Transparent Proxy',
    placeholder: '',
    type: 'checkbox',
  },
  { key: 'comment', label: 'Comment', placeholder: 'Comment' },
  {
    key: 'insert_queue_before',
    label: 'Insert Queue Before',
    placeholder: 'Select queue',
  },
  { key: 'parent_queue', label: 'Parent Queue', placeholder: 'none' },
  { key: 'queue_type', label: 'Queue Type', placeholder: 'Select queue type' },
  {
    key: 'on_login',
    label: 'On Login',
    placeholder: ':put "user logged in"',
    type: 'code',
  },
  {
    key: 'on_logout',
    label: 'On Logout',
    placeholder: ':put "user logged out"',
    type: 'code',
  },
]

export function HotspotManagement() {
  const muiTheme = useMuiTheme()
  const [routers, setRouters] = useState<MikrotikRouter[]>([])
  const [routerId, setRouterId] = useState('')
  const [isLoadingRouters, setIsLoadingRouters] = useState(true)
  const [view, setView] = useState<HotspotView>('user')
  const [isLoadingData, setIsLoadingData] = useState(false)
  const [isFetching, setIsFetching] = useState(false)
  const [users, setUsers] = useState<HotspotUser[]>([])
  const [profiles, setProfiles] = useState<HotspotProfile[]>([])
  const [activeSessions, setActiveSessions] = useState<Rated<HotspotActive>[]>(
    []
  )
  const [hosts, setHosts] = useState<Rated<HotspotHost>[]>([])
  const [serverNames, setServerNames] = useState<string[]>([])
  const [poolNames, setPoolNames] = useState<string[]>([])
  const [queueNames, setQueueNames] = useState<string[]>([])
  const [queueTypeNames, setQueueTypeNames] = useState<string[]>([])
  const [userProfileFilter, setUserProfileFilter] = useState('all')
  const [userDialog, setUserDialog] = useState({ open: false, editing: false })
  const [userForm, setUserForm] = useState<UserForm>(emptyUser)
  const [editingUser, setEditingUser] = useState<HotspotUser | null>(null)
  const [profileDialog, setProfileDialog] = useState({
    open: false,
    editing: false,
  })
  const [profileForm, setProfileForm] = useState<ProfileForm>(emptyProfile)
  const [editingProfile, setEditingProfile] = useState<HotspotProfile | null>(
    null
  )
  const [confirm, setConfirm] = useState<ConfirmState | null>(null)
  const [isConfirming, setIsConfirming] = useState(false)
  const dataRequestId = useRef(0)
  // When the last live event arrived; an older full refresh must not overwrite newer live rows.
  const liveAt = useRef(0)

  const loadRouters = useCallback(async () => {
    try {
      const result = await getMikrotikRouters()
      setRouters(result)
      setRouterId((current) =>
        result.some((router) => String(router.id) === current) ? current : ''
      )
    } catch (error) {
      toast.error(getHotspotErrorMessage(error))
    } finally {
      setIsLoadingRouters(false)
    }
  }, [])

  const loadRouterData = useCallback(
    async ({ silent = false }: { silent?: boolean } = {}) => {
      const requestId = ++dataRequestId.current
      if (!routerId) return
      const startedAt = performance.now()
      setIsFetching(true)
      try {
        const id = Number(routerId)
        const [userData, profileData, activeData, hostData, serverData] =
          await Promise.all([
            getHotspotUsers(id),
            getHotspotProfiles(id),
            getHotspotActive(id),
            getHotspotHosts(id),
            getHotspotServers(id),
          ])
        // Drop stale responses (router switched or a newer request started).
        if (requestId !== dataRequestId.current) return
        setProfiles(profileData)
        setServerNames(serverData)
        // A background refresh that started before the latest live event is older than what is shown.
        if (!(silent && liveAt.current > startedAt)) {
          setUsers(userData)
          setActiveSessions((current) => keepRates(activeData, current))
          setHosts((current) => keepRates(hostData, current))
        }
      } catch (error) {
        // Polling failures are ignored so tables keep their last data and no toast repeats.
        if (!silent && requestId === dataRequestId.current) {
          toast.error(getHotspotErrorMessage(error))
        }
      } finally {
        if (requestId === dataRequestId.current) {
          setIsLoadingData(false)
          setIsFetching(false)
        }
      }
    },
    [routerId]
  )

  // Opsi dropdown profile (pool/queue) jarang berubah, jadi tidak ikut polling dan gagal dengan tenang.
  const loadProfileOptions = useCallback(async () => {
    if (!routerId) return
    const id = Number(routerId)
    const [pools, queues, queueTypes] = await Promise.all([
      getHotspotPools(id).catch(() => [] as string[]),
      getHotspotQueues(id).catch(() => [] as string[]),
      getHotspotQueueTypes(id).catch(() => [] as string[]),
    ])
    setPoolNames(pools)
    setQueueNames(queues)
    setQueueTypeNames(queueTypes)
  }, [routerId])

  useEffect(() => {
    const timer = window.setTimeout(() => void loadRouters(), 0)
    return () => window.clearTimeout(timer)
  }, [loadRouters])

  useEffect(() => {
    const timer = window.setTimeout(() => void loadProfileOptions(), 0)
    return () => window.clearTimeout(timer)
  }, [loadProfileOptions])

  useEffect(() => {
    const timer = window.setTimeout(() => void loadRouterData(), 0)
    return () => window.clearTimeout(timer)
  }, [loadRouterData])

  // Realtime polling: only while a MikroTik is selected, no caching.
  useEffect(() => {
    if (!routerId) return

    let busy = false
    const timer = window.setInterval(() => {
      if (busy || document.hidden) return
      busy = true
      void loadRouterData({ silent: true }).finally(() => {
        busy = false
      })
    }, HOTSPOT_LIST_POLL_INTERVAL_MS)

    return () => window.clearInterval(timer)
  }, [routerId, loadRouterData])

  const applyLive = useCallback((live: HotspotLive) => {
    if (!live.users && !live.active && !live.hosts) return
    liveAt.current = performance.now()
    setUsers((current) =>
      live.users ? applyDelta(current, live.users, (row) => row.id) : current
    )
    setActiveSessions((current) =>
      live.active ? applyDelta(current, live.active, (row) => row.id) : current
    )
    setHosts((current) =>
      live.hosts ? applyDelta(current, live.hosts, (row) => row.id) : current
    )
  }, [])
  useLiveStream<HotspotLive>('/hotspot/live', routerId, applyLive)

  const changeRouter = (value: string) => {
    setUsers([])
    setProfiles([])
    setActiveSessions([])
    setHosts([])
    liveAt.current = 0
    setServerNames([])
    setPoolNames([])
    setQueueNames([])
    setQueueTypeNames([])
    setUserProfileFilter('all')
    setIsLoadingData(Boolean(value))
    setRouterId(value)
  }

  const openAddUser = () => {
    setEditingUser(null)
    setUserForm({
      ...emptyUser,
      server: 'all',
      profile:
        profiles.find((profile) => profile.name === 'default')?.name ?? '',
    })
    setUserDialog({ open: true, editing: false })
  }

  const openEditUser = (user: HotspotUser) => {
    setEditingUser(user)
    setUserForm(userToForm(user))
    setUserDialog({ open: true, editing: true })
  }

  const submitUser = async (event: React.FormEvent) => {
    event.preventDefault()
    if (!routerId) return
    // Select fields are not covered by native `required`, so every required field is checked here.
    const missing = [
      ['server', 'Server'],
      ['name', 'Name'],
      ['profile', 'Profile'],
      ...(editingUser ? [] : [['password', 'Password']]),
    ].find(([key]) => !userForm[key as keyof UserForm].trim())
    if (missing) {
      toast.error(`${missing[1]} is required.`)
      return
    }

    const original = editingUser ? userToForm(editingUser) : undefined
    const input = changedInput(userKeys, userForm, original)
    if (editingUser && Object.keys(input).length === 0) {
      toast.info('No changes to save.')
      setUserDialog({ open: false, editing: false })
      return
    }

    try {
      const result = await saveHotspotUser(
        Number(routerId),
        input,
        editingUser?.id
      )
      toast.success(result.message)
      setUserDialog({ open: false, editing: false })
      await loadRouterData()
    } catch (error) {
      toast.error(getHotspotErrorMessage(error))
    }
  }

  const openAddProfile = () => {
    setEditingProfile(null)
    // Bawaan RouterOS keduanya "yes"; checkbox menampilkan nilai yang akan dikirim.
    setProfileForm({
      ...emptyProfile,
      address_pool: 'none',
      add_mac_cookie: 'yes',
      transparent_proxy: 'yes',
    })
    setProfileDialog({ open: true, editing: false })
    void loadProfileOptions()
  }

  const openEditProfile = (profile: HotspotProfile) => {
    setEditingProfile(profile)
    setProfileForm(profileToForm(profile))
    setProfileDialog({ open: true, editing: true })
    void loadProfileOptions()
  }

  const submitProfile = async (event: React.FormEvent) => {
    event.preventDefault()
    if (!routerId) return
    if (!profileForm.name.trim()) {
      toast.error('Name is required.')
      return
    }

    const original = editingProfile ? profileToForm(editingProfile) : undefined
    const input = changedInput(profileKeys, profileForm, original)
    if (editingProfile && Object.keys(input).length === 0) {
      toast.info('No changes to save.')
      setProfileDialog({ open: false, editing: false })
      return
    }

    try {
      const result = await saveHotspotProfile(
        Number(routerId),
        input,
        editingProfile?.id
      )
      toast.success(result.message)
      setProfileDialog({ open: false, editing: false })
      await loadRouterData()
    } catch (error) {
      toast.error(getHotspotErrorMessage(error))
    }
  }

  const confirmDeleteUser = (user: HotspotUser) =>
    setConfirm({
      title: 'Delete Hotspot User?',
      description: `User "${user.name}" will be permanently deleted. This action cannot be undone.`,
      actionLabel: 'Delete',
      destructive: true,
      onConfirm: async () => {
        const result = await deleteHotspotUser(Number(routerId), user.id)
        toast.success(result.message)
        await loadRouterData()
      },
    })

  const confirmToggleUser = (user: HotspotUser) => {
    const disabled = !user.disabled
    const action = disabled ? 'Disable' : 'Enable'
    setConfirm({
      title: `${action} Hotspot User?`,
      description: `Are you sure you want to ${action.toLowerCase()} "${user.name}"?`,
      actionLabel: action,
      destructive: disabled,
      onConfirm: async () => {
        const result = await setHotspotUserDisabled(
          Number(routerId),
          user.id,
          disabled
        )
        toast.success(result.message)
        await loadRouterData()
      },
    })
  }

  const confirmDeleteProfile = (profile: HotspotProfile) =>
    setConfirm({
      title: 'Delete Hotspot Profile?',
      description: `Profile "${profile.name}" will be permanently deleted. This action cannot be undone.`,
      actionLabel: 'Delete',
      destructive: true,
      onConfirm: async () => {
        const result = await deleteHotspotProfile(Number(routerId), profile.id)
        toast.success(result.message)
        await loadRouterData()
      },
    })

  const confirmKick = (session: HotspotActive) =>
    setConfirm({
      title: 'Disconnect user?',
      description: `Session "${session.user}" (${session.address}) will be disconnected.`,
      actionLabel: 'Disconnect',
      destructive: true,
      onConfirm: async () => {
        const result = await kickHotspotActive(Number(routerId), session.id)
        toast.success(result.message)
        await loadRouterData()
      },
    })

  const confirmDeleteHost = (host: HotspotHost) =>
    setConfirm({
      title: 'Delete Hotspot Host?',
      description: `Host "${host.mac_address}" will be removed. A device that is still connected will be detected again.`,
      actionLabel: 'Delete',
      destructive: true,
      onConfirm: async () => {
        const result = await deleteHotspotHost(Number(routerId), host.id)
        toast.success(result.message)
        await loadRouterData()
      },
    })

  // Shared bulk flow: confirm once, run per item, then report successes and failures together.
  const confirmBulk = <T,>(
    items: T[],
    options: {
      title: string
      description: string
      actionLabel?: string
      doneLabel: string
      task: (item: T) => Promise<unknown>
    }
  ) =>
    setConfirm({
      title: options.title,
      description: options.description,
      actionLabel: options.actionLabel ?? 'Delete',
      destructive: true,
      onConfirm: async () => {
        const { succeeded, failed, firstError } = await runBulk(
          items,
          options.task
        )
        if (succeeded > 0) {
          toast.success(`${succeeded} ${options.doneLabel}.`)
        }
        if (failed > 0) {
          toast.error(`${failed} failed: ${getHotspotErrorMessage(firstError)}`)
        }
        await loadRouterData()
      },
    })

  const confirmBulkDeleteUsers = (rows: HotspotUser[]) =>
    confirmBulk(rows, {
      title: `Delete ${rows.length} Hotspot Users?`,
      description:
        'The selected users will be permanently deleted. This action cannot be undone.',
      doneLabel: 'user(s) deleted',
      task: (user) => deleteHotspotUser(Number(routerId), user.id),
    })

  const confirmBulkDeleteProfiles = (rows: HotspotProfile[]) =>
    confirmBulk(rows, {
      title: `Delete ${rows.length} Hotspot Profiles?`,
      description:
        'The selected profiles will be permanently deleted. This action cannot be undone.',
      doneLabel: 'profile(s) deleted',
      task: (profile) => deleteHotspotProfile(Number(routerId), profile.id),
    })

  const confirmBulkKick = (rows: HotspotActive[]) =>
    confirmBulk(rows, {
      title: `Disconnect ${rows.length} users?`,
      description: 'The selected sessions will be disconnected.',
      actionLabel: 'Disconnect',
      doneLabel: 'session(s) disconnected',
      task: (session) => kickHotspotActive(Number(routerId), session.id),
    })

  const confirmBulkDeleteHosts = (rows: HotspotHost[]) =>
    confirmBulk(rows, {
      title: `Delete ${rows.length} Hotspot Hosts?`,
      description:
        'The selected hosts will be removed. Devices that are still connected will be detected again.',
      doneLabel: 'host(s) deleted',
      task: (host) => deleteHotspotHost(Number(routerId), host.id),
    })

  const handleConfirm = async () => {
    if (!confirm) return
    setIsConfirming(true)
    try {
      await confirm.onConfirm()
      setConfirm(null)
    } catch (error) {
      toast.error(getHotspotErrorMessage(error))
    } finally {
      setIsConfirming(false)
    }
  }

  const profileOptions = profiles.map((profile) => profile.name)
  // "all" adalah nilai bawaan RouterOS untuk user yang berlaku di semua server.
  const serverOptions = ['all', ...serverNames]
  const filteredUsers =
    userProfileFilter === 'all'
      ? users
      : users.filter((user) => user.profile === userProfileFilter)
  const currentRouter = routers.find((router) => String(router.id) === routerId)

  const routerSelect = (
    <Select
      value={routerId}
      onValueChange={changeRouter}
      disabled={isLoadingRouters || routers.length === 0}
    >
      <SelectTrigger size='sm' className='h-7 w-full sm:w-43'>
        <SelectValue
          placeholder={
            isLoadingRouters ? 'Loading MikroTik...' : 'Select MikroTik'
          }
        />
      </SelectTrigger>
      <SelectContent>
        {routers.map((router) => (
          <SelectItem key={router.id} value={String(router.id)}>
            {router.name}
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  )

  return (
    <>
      <Header fixed>
        <div className='ms-auto flex items-center gap-2'>
          <ThemeSwitch />
          <ProfileDropdown />
        </div>
      </Header>
      <Main fixed fluid className='min-h-0 !overflow-y-auto'>
        <div className='mb-3'>
          <h1 className='text-2xl font-bold tracking-tight'>
            Hotspot Management
          </h1>
        </div>

        <div className='mb-3 grid grid-cols-2 gap-2 sm:gap-3 lg:grid-cols-4'>
          <StatCard
            title='User'
            value={users.length}
            description='Hotspot users'
            icon={Users}
            tone='blue'
            loading={isLoadingData}
            active={view === 'user'}
            onClick={() => setView('user')}
          />
          <StatCard
            title='Active'
            value={activeSessions.length}
            description={`${percent(activeSessions.length, users.length)} of users`}
            icon={Wifi}
            tone='green'
            loading={isLoadingData}
            active={view === 'active'}
            onClick={() => setView('active')}
          />
          <StatCard
            title='Host'
            value={hosts.length}
            description='Detected devices'
            icon={Laptop}
            tone='red'
            loading={isLoadingData}
            active={view === 'host'}
            onClick={() => setView('host')}
          />
          <StatCard
            title='Profile'
            value={profiles.length}
            description='User profiles'
            icon={Layers}
            tone='violet'
            loading={isLoadingData}
            active={view === 'profile'}
            onClick={() => setView('profile')}
          />
        </div>

        <ThemeProvider theme={muiTheme}>
          <Tabs
            value={view}
            onValueChange={(value) => setView(value as HotspotView)}
            className='w-full'
          >
            <TabsContent value='user'>
              <DataTable
                columns={userColumns}
                rows={filteredUsers}
                isRefreshing={isFetching}
                onBulkDelete={confirmBulkDeleteUsers}
                bulkDeleteLabel='Delete selected'
                getRowId={(user) => user.id}
                addLabel='Add User'
                onAdd={currentRouter ? openAddUser : undefined}
                toolbarBeforeAdd={
                  <>
                    {routerSelect}
                    <Select
                      value={userProfileFilter}
                      onValueChange={setUserProfileFilter}
                    >
                      <SelectTrigger size='sm' className='h-7 w-full sm:w-43'>
                        <SelectValue placeholder='Filter by profile' />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value='all'>All profiles</SelectItem>
                        {profileOptions.map((profile) => (
                          <SelectItem key={profile} value={profile}>
                            {profile}
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                  </>
                }
                actions={(user) => (
                  <>
                    <Button
                      type='button'
                      size='icon'
                      variant='ghost'
                      className='size-6 p-0'
                      title={user.disabled ? 'Enable' : 'Disable'}
                      aria-label={`${user.disabled ? 'Enable' : 'Disable'} ${user.name}`}
                      disabled={isConfirming}
                      onClick={() => confirmToggleUser(user)}
                    >
                      {user.disabled ? (
                        <Power className='size-3.5' />
                      ) : (
                        <PowerOff className='size-3.5' />
                      )}
                    </Button>
                    <IconButton title='Edit' onClick={() => openEditUser(user)}>
                      <Pencil className='size-3.5' />
                    </IconButton>
                    <IconButton
                      title='Delete'
                      destructive
                      onClick={() => confirmDeleteUser(user)}
                    >
                      <Trash2 className='size-3.5' />
                    </IconButton>
                  </>
                )}
              />
            </TabsContent>
            <TabsContent value='active'>
              <DataTable
                columns={activeColumns}
                rows={activeSessions}
                isRefreshing={isFetching}
                onBulkDelete={confirmBulkKick}
                bulkDeleteLabel='Disconnect selected'
                getRowId={(session) => session.id}
                toolbarBeforeAdd={routerSelect}
                actions={(session) => (
                  <IconButton
                    title='Disconnect / Kick'
                    destructive
                    onClick={() => confirmKick(session)}
                  >
                    <Trash2 className='size-3.5' />
                  </IconButton>
                )}
              />
            </TabsContent>
            <TabsContent value='host'>
              <DataTable
                columns={hostColumns}
                rows={hosts}
                isRefreshing={isFetching}
                onBulkDelete={confirmBulkDeleteHosts}
                bulkDeleteLabel='Delete selected'
                getRowId={(host) => host.id}
                toolbarBeforeAdd={routerSelect}
                actions={(host) => (
                  <IconButton
                    title='Delete'
                    destructive
                    onClick={() => confirmDeleteHost(host)}
                  >
                    <Trash2 className='size-3.5' />
                  </IconButton>
                )}
              />
            </TabsContent>
            <TabsContent value='profile'>
              <DataTable
                columns={profileColumns}
                rows={profiles}
                isRefreshing={isFetching}
                onBulkDelete={confirmBulkDeleteProfiles}
                bulkDeleteLabel='Delete selected'
                getRowId={(profile) => profile.id}
                addLabel='Add Profile'
                onAdd={currentRouter ? openAddProfile : undefined}
                toolbarBeforeAdd={routerSelect}
                actions={(profile) => (
                  <>
                    <IconButton
                      title='Edit'
                      onClick={() => openEditProfile(profile)}
                    >
                      <Pencil className='size-3.5' />
                    </IconButton>
                    <IconButton
                      title='Delete'
                      destructive
                      onClick={() => confirmDeleteProfile(profile)}
                    >
                      <Trash2 className='size-3.5' />
                    </IconButton>
                  </>
                )}
              />
            </TabsContent>
          </Tabs>
        </ThemeProvider>
      </Main>

      <FormDialog
        open={userDialog.open}
        onOpenChange={(open) => setUserDialog((state) => ({ ...state, open }))}
        title={userDialog.editing ? 'Edit Hotspot User' : 'Add Hotspot User'}
        idPrefix='hotspot-user'
        fields={userFields.map((field) =>
          field.key === 'profile'
            ? { ...field, options: profileOptions }
            : field.key === 'server'
              ? { ...field, options: serverOptions }
              : field.key === 'password' && userDialog.editing
                ? { ...field, required: false }
                : field
        )}
        data={userForm}
        onChange={(data) => setUserForm(data as UserForm)}
        onSubmit={submitUser}
        submitLabel={userDialog.editing ? 'Save changes' : 'Add User'}
        tabs={userTabs}
      />
      <FormDialog
        open={profileDialog.open}
        onOpenChange={(open) =>
          setProfileDialog((state) => ({ ...state, open }))
        }
        title={
          profileDialog.editing ? 'Edit Hotspot Profile' : 'Add Hotspot Profile'
        }
        idPrefix='hotspot-profile'
        fields={profileFields.map((field) =>
          field.key === 'address_pool'
            ? { ...field, options: ['none', ...poolNames] }
            : field.key === 'insert_queue_before'
              ? { ...field, options: queueNames }
              : field.key === 'parent_queue'
                ? { ...field, options: queueNames }
                : field.key === 'queue_type'
                  ? { ...field, options: queueTypeNames }
                  : field
        )}
        data={profileForm}
        onChange={(data) => setProfileForm(data as ProfileForm)}
        onSubmit={submitProfile}
        submitLabel={profileDialog.editing ? 'Save changes' : 'Add Profile'}
        tabs={profileTabs}
        wide
      />
      <Dialog
        open={confirm !== null}
        onOpenChange={(open) => !open && setConfirm(null)}
      >
        <DialogContent className='sm:max-w-md' showCloseButton={false}>
          <DialogHeader>
            <DialogTitle>{confirm?.title}</DialogTitle>
            <DialogDescription>{confirm?.description}</DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button
              variant='outline'
              onClick={() => setConfirm(null)}
              disabled={isConfirming}
            >
              Cancel
            </Button>
            <Button
              variant={confirm?.destructive ? 'destructive' : 'default'}
              onClick={() => void handleConfirm()}
              disabled={isConfirming}
            >
              {isConfirming ? 'Processing...' : confirm?.actionLabel}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  )
}
