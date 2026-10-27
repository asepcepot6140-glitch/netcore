import { useCallback, useEffect, useRef, useState } from 'react'
import { ThemeProvider } from '@mui/material'
import {
  Layers,
  Pencil,
  Power,
  PowerOff,
  Trash2,
  Users,
  Wifi,
  WifiOff,
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
import { FormDialog, type Field } from '../components/form-dialog'
import { IconButton } from '../components/icon-button'
import { applyDelta, useLiveStream, type Delta } from '../hooks/use-live-stream'
import { useMuiTheme } from '../hooks/use-mui-theme'
import { getMikrotikRouters, type MikrotikRouter } from '../router/mikrotik-api'
import {
  deletePppoeProfile,
  deletePppoeSecret,
  disconnectPppoeActive,
  getPppoeActive,
  getPppoeErrorMessage,
  getPppoeOffline,
  getPppoePools,
  getPppoeProfiles,
  getPppoeQueueNames,
  getPppoeSecrets,
  savePppoeProfile,
  savePppoeSecret,
  setPppoeSecretDisabled,
  type PppoeActiveConnection,
  type PppoeLive,
  type PppoeProfile,
  type PppoeProfileInput,
  type PppoePool,
  type PppoeSecret,
  type PppoeSecretInput,
} from './pppoe-api'

interface SecretRow extends PppoeSecret {
  index: number
  flags: string
  localAddress: string
  remoteAddress: string
  callerId: string
  lastLoggedOut: string
  lastCallerId: string
  lastDisconnectReason: string
}

interface ActiveRow extends PppoeActiveConnection {
  index: number
  flags: string
  callerId: string
  sessionId: string
}

interface ProfileRow {
  index: number
  flags: string
  name: string
  localAddress: string
  remoteAddress: string
  rateLimit: string
  onlyOne: string
  insertQueueBefore: string
  parentQueue: string
  addressList: string
  onUp: string
  onDown: string
  comment: string
}

type PppoeView = 'secret' | 'online' | 'offline' | 'profile'

// Live updates arrive over one stream; the full refresh only repairs drift and reloads profiles.
const PPPOE_LIST_POLL_INTERVAL_MS = 30_000

// Applies a live delta (raw API rows) to a table of display rows and renumbers them, since the
// row number depends on the position.
function applyLiveRows<
  R extends { id: string },
  T extends R & { index: number },
>(
  current: T[],
  delta: Delta<R> | undefined,
  toRow: (raw: R, index: number) => T
): T[] {
  if (!delta) return current
  const mapped = { ...delta, upsert: delta.upsert.map((raw) => toRow(raw, 0)) }
  const next = applyDelta(current, mapped, (row) => row.id)
  if (next === current) return current
  return next.map((row, index) =>
    row.index === index ? row : { ...row, index }
  )
}

function percent(part: number, total: number) {
  return total === 0 ? '0%' : `${Math.round((part / total) * 100)}%`
}

interface ConfirmState {
  title: string
  description: string
  actionLabel: string
  destructive?: boolean
  onConfirm: () => Promise<void>
}

const emptySecret: Partial<SecretRow> = {
  name: '',
  password: '',
  service: 'pppoe',
  profile: '',
  local_address: '',
  remote_address: '',
  localAddress: '',
  remoteAddress: '',
  caller_id: '',
  last_logged_out: '',
  last_caller_id: '',
  last_disconnect_reason: '',
  comment: '',
  id: '',
  disabled: false,
  index: 0,
  flags: '',
  callerId: '',
  lastLoggedOut: '',
  lastCallerId: '',
  lastDisconnectReason: '',
}

const emptyProfile: Partial<ProfileRow> = {
  name: '',
  localAddress: '',
  remoteAddress: '',
  rateLimit: '',
  onlyOne: '',
  insertQueueBefore: '',
  parentQueue: '',
  addressList: '',
  onUp: '',
  onDown: '',
  comment: '',
}

const serviceOptions = ['any', 'pppoe', 'pptp', 'l2tp', 'sstp', 'ovpn', 'async']

function lastLoggedOutSortValue(value: string) {
  const match = value.match(
    /^([a-z]{3})\/(\d{1,2})\/(\d{4})(?:\s+(\d{1,2}:\d{2}:\d{2}))?$/i
  )
  if (!match) return null

  const month = [
    'jan',
    'feb',
    'mar',
    'apr',
    'may',
    'jun',
    'jul',
    'aug',
    'sep',
    'oct',
    'nov',
    'dec',
  ].indexOf(match[1].toLowerCase())
  if (month < 0) return null

  const [hours = '0', minutes = '0', seconds = '0'] = (
    match[4] ?? '00:00:00'
  ).split(':')
  return new Date(
    Number(match[3]),
    month,
    Number(match[2]),
    Number(hours),
    Number(minutes),
    Number(seconds)
  ).getTime()
}

// Secret disabled: flag X di kolom pertama dan seluruh teks baris dibuat abu-abu samar, seperti Simple Queue.
function dimDisabled(columns: Column<SecretRow>[]): Column<SecretRow>[] {
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

const secretColumns: Column<SecretRow>[] = dimDisabled([
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
  { header: 'Service', cell: (row) => row.service },
  { header: 'Password', cell: (row) => row.password },
  { header: 'Profile', cell: (row) => row.profile },
  { header: 'Local Address', cell: (row) => row.localAddress || '-' },
  { header: 'Remote Address', cell: (row) => row.remoteAddress || '-' },
  {
    header: 'Last Logged Out',
    cell: (row) => row.lastLoggedOut || '-',
    getSortValue: (row) => lastLoggedOutSortValue(row.lastLoggedOut),
  },
  { header: 'Last Caller ID', cell: (row) => row.lastCallerId || '-' },
  {
    header: 'Last Disconnect',
    cell: (row) => row.lastDisconnectReason || '-',
  },
])

const sessionColumns: Column<ActiveRow>[] = [
  {
    header: 'Name',
    cell: (row) => <span className='font-semibold'>{row.name}</span>,
  },
  { header: 'Service', cell: (row) => row.service },
  { header: 'Caller ID', cell: (row) => row.callerId },
  { header: 'Address', cell: (row) => row.address },
  { header: 'Uptime', cell: (row) => row.uptime },
  { header: 'Session ID', cell: (row) => row.sessionId },
]

const profileColumns: Column<ProfileRow>[] = [
  {
    header: 'Name',
    cell: (row) => <span className='font-semibold'>{row.name}</span>,
  },
  { header: 'Local Address', cell: (row) => row.localAddress || '-' },
  { header: 'Remote Address', cell: (row) => row.remoteAddress || '-' },
  { header: 'Rate Limit', cell: (row) => row.rateLimit || '-' },
]

const secretFields: Field<SecretRow>[] = [
  { key: 'name', label: 'Name', placeholder: 'Username', required: true },
  { key: 'password', label: 'Password', placeholder: 'Password' },
  {
    key: 'service',
    label: 'Service',
    placeholder: 'Select service',
    required: true,
    options: serviceOptions,
  },
  { key: 'profile', label: 'Profile', placeholder: 'Select profile' },
  { key: 'localAddress', label: 'Local Address', placeholder: '0.0.0.0' },
  { key: 'remoteAddress', label: 'Remote Address', placeholder: '0.0.0.0' },
  { key: 'comment', label: 'Comment', placeholder: 'Comment' },
]

const profileFields: Field<ProfileRow>[] = [
  { key: 'name', label: 'Name', placeholder: 'Profile Name', required: true },
  { key: 'rateLimit', label: 'Rate Limit', placeholder: '7M/7M' },
  { key: 'localAddress', label: 'Local Address', placeholder: '0.0.0.0' },
  { key: 'remoteAddress', label: 'Remote Address', placeholder: 'pool-name' },
  {
    key: 'onlyOne',
    label: 'Only One',
    placeholder: 'Select setting',
    options: ['yes', 'no'],
  },
  {
    key: 'insertQueueBefore',
    label: 'Insert Queue Before',
    placeholder: 'Select queue',
  },
  {
    key: 'parentQueue',
    label: 'Parent Queue',
    placeholder: 'Select parent queue',
  },
  { key: 'addressList', label: 'Address List', placeholder: 'Address list' },
  { key: 'onUp', label: 'On Up', placeholder: 'Script' },
  { key: 'onDown', label: 'On Down', placeholder: 'Script' },
  { key: 'comment', label: 'Comment', placeholder: 'Comment' },
]

// Only fields that differ from the loaded secret; an emptied field is sent as "" to clear it.
function changedSecretInput(
  original: SecretRow | undefined,
  form: Partial<SecretRow>
): PppoeSecretInput {
  const input: PppoeSecretInput = {}
  const name = form.name?.trim() ?? ''
  if (name && name !== original?.name) input.name = name
  if (form.password && form.password !== original?.password) {
    input.password = form.password
  }
  if (form.service && form.service !== original?.service) {
    input.service = form.service
  }
  if (form.profile && form.profile !== original?.profile) {
    input.profile = form.profile
  }
  const local = form.localAddress?.trim() ?? ''
  if (local !== (original?.localAddress ?? '')) input.local_address = local
  const remote = form.remoteAddress?.trim() ?? ''
  if (remote !== (original?.remoteAddress ?? '')) input.remote_address = remote
  const comment = form.comment ?? ''
  if (comment !== (original?.comment ?? '')) input.comment = comment
  return input
}

function secretRow(secret: PppoeSecret, index: number): SecretRow {
  return {
    ...secret,
    index,
    flags: secret.disabled ? 'X' : '',
    localAddress: secret.local_address,
    remoteAddress: secret.remote_address,
    callerId: secret.caller_id,
    lastLoggedOut: secret.last_logged_out,
    lastCallerId: secret.last_caller_id,
    lastDisconnectReason: secret.last_disconnect_reason,
  }
}

function profileRow(profile: PppoeProfile, index: number): ProfileRow {
  return {
    index,
    flags: profile.name === 'default' ? '*' : '',
    name: profile.name,
    localAddress: profile.local_address,
    remoteAddress: profile.remote_address,
    rateLimit: profile.rate_limit,
    onlyOne: profile.only_one,
    insertQueueBefore: profile.insert_queue_before,
    parentQueue: profile.parent_queue,
    addressList: profile.address_list,
    onUp: profile.on_up,
    onDown: profile.on_down,
    comment: profile.comment,
  }
}

function activeRow(
  connection: PppoeActiveConnection,
  index: number
): ActiveRow {
  return {
    ...connection,
    index,
    flags: '',
    callerId: connection.caller_id,
    sessionId: connection.session_id,
  }
}

// Maps form keys to API keys, excluding name which is handled separately.
const profileApiKeys = {
  localAddress: 'local_address',
  remoteAddress: 'remote_address',
  onlyOne: 'only_one',
  rateLimit: 'rate_limit',
  insertQueueBefore: 'insert_queue_before',
  parentQueue: 'parent_queue',
  addressList: 'address_list',
  onUp: 'on_up',
  onDown: 'on_down',
  comment: 'comment',
} as const satisfies Partial<Record<keyof ProfileRow, keyof PppoeProfile>>

// Add sends name plus filled fields; edit sends only fields that differ (empty clears).
function profileInput(
  form: Partial<ProfileRow>,
  original?: ProfileRow
): PppoeProfileInput {
  const input: PppoeProfileInput = {}
  const name = form.name?.trim() ?? ''
  if (!original || name !== original.name) input.name = name

  for (const [formKey, apiKey] of Object.entries(profileApiKeys) as [
    keyof typeof profileApiKeys,
    (typeof profileApiKeys)[keyof typeof profileApiKeys],
  ][]) {
    const value = (form[formKey] ?? '').trim()
    if (original ? value !== (original[formKey] ?? '') : value !== '') {
      input[apiKey] = value
    }
  }
  return input
}

export function PppoeSecrets() {
  const muiTheme = useMuiTheme()
  const [routers, setRouters] = useState<MikrotikRouter[]>([])
  const [routerId, setRouterId] = useState('')
  const [isLoadingRouters, setIsLoadingRouters] = useState(true)
  const [view, setView] = useState<PppoeView>('secret')
  const [isLoadingData, setIsLoadingData] = useState(false)
  const [isFetching, setIsFetching] = useState(false)
  const [secrets, setSecrets] = useState<SecretRow[]>([])
  const [secretProfileFilter, setSecretProfileFilter] = useState('all')
  const [offlineSecrets, setOfflineSecrets] = useState<SecretRow[]>([])
  const [activeSessions, setActiveSessions] = useState<ActiveRow[]>([])
  const [profiles, setProfiles] = useState<ProfileRow[]>([])
  const [ipPools, setIpPools] = useState<PppoePool[]>([])
  const [queueNames, setQueueNames] = useState<string[]>([])
  const [secretDialog, setSecretDialog] = useState({
    open: false,
    editing: false,
  })
  const [secretForm, setSecretForm] = useState<Partial<SecretRow>>(emptySecret)
  const [editingSecretId, setEditingSecretId] = useState<string | null>(null)
  const [profileDialog, setProfileDialog] = useState({
    open: false,
    editing: false,
  })
  const [profileForm, setProfileForm] =
    useState<Partial<ProfileRow>>(emptyProfile)
  const [editingProfileName, setEditingProfileName] = useState<string | null>(
    null
  )
  const [confirm, setConfirm] = useState<ConfirmState | null>(null)
  const [isConfirming, setIsConfirming] = useState(false)
  const dataRequestId = useRef(0)
  const fullLoads = useRef(0)
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
      toast.error(getPppoeErrorMessage(error))
    } finally {
      setIsLoadingRouters(false)
    }
  }, [])

  const loadRouterData = useCallback(
    async ({ silent = false }: { silent?: boolean } = {}) => {
      const requestId = ++dataRequestId.current
      if (!routerId) return
      const startedAt = performance.now()
      if (!silent) fullLoads.current++
      setIsFetching(true)
      try {
        const id = Number(routerId)
        const [secretData, offlineData, activeData, profileData] =
          await Promise.all([
            getPppoeSecrets(id),
            getPppoeOffline(id),
            getPppoeActive(id),
            getPppoeProfiles(id),
          ])
        // Pools and queues are only needed by the form dialogs,
        // so polling skips them to keep the router load light.
        const [poolData, queues] = silent
          ? [undefined, undefined]
          : await Promise.all([getPppoePools(id), getPppoeQueueNames(id)])
        if (requestId !== dataRequestId.current) return
        // A background refresh that started before the latest live event is older than what is shown.
        if (!(silent && liveAt.current > startedAt)) {
          setSecrets(secretData.map(secretRow))
          setOfflineSecrets(offlineData.map(secretRow))
          setActiveSessions(activeData.map(activeRow))
        }
        setProfiles(profileData.map(profileRow))
        if (poolData) setIpPools(poolData)
        if (queues) setQueueNames(queues)
      } catch (error) {
        // Polling failures are ignored so the tables keep their last data
        // and no toast is spammed every interval.
        if (!silent && requestId === dataRequestId.current) {
          toast.error(getPppoeErrorMessage(error))
        }
      } finally {
        if (!silent) fullLoads.current--
        if (requestId === dataRequestId.current) {
          setIsLoadingData(false)
          setIsFetching(false)
        }
      }
    },
    [routerId]
  )

  useEffect(() => {
    const timer = window.setTimeout(() => void loadRouters(), 0)
    return () => window.clearTimeout(timer)
  }, [loadRouters])

  useEffect(() => {
    const timer = window.setTimeout(() => void loadRouterData(), 0)
    return () => window.clearTimeout(timer)
  }, [loadRouterData])

  // Slow full refresh; secrets, offline and active rows come from the live stream below.
  useEffect(() => {
    if (!routerId) return

    let busy = false
    const timer = window.setInterval(() => {
      // Skip while the tab is hidden, a poll is running, or a full load
      // (which also fetches pools/queues) is still in flight
      if (busy || document.hidden || fullLoads.current > 0) return
      busy = true
      void loadRouterData({ silent: true }).finally(() => {
        busy = false
      })
    }, PPPOE_LIST_POLL_INTERVAL_MS)

    return () => window.clearInterval(timer)
  }, [routerId, loadRouterData])

  const applyLive = useCallback((live: PppoeLive) => {
    if (!live.secrets && !live.offline && !live.active) return
    liveAt.current = performance.now()
    setSecrets((current) => applyLiveRows(current, live.secrets, secretRow))
    setOfflineSecrets((current) =>
      applyLiveRows(current, live.offline, secretRow)
    )
    setActiveSessions((current) =>
      applyLiveRows(current, live.active, activeRow)
    )
  }, [])
  useLiveStream<PppoeLive>('/pppoe/live', routerId, applyLive)

  const openAddSecret = () => {
    setEditingSecretId(null)
    setSecretForm({
      ...emptySecret,
      profile:
        profiles.find((profile) => profile.name === 'default')?.name ?? '',
    })
    setSecretDialog({ open: true, editing: false })
  }

  const openEditSecret = (secret: SecretRow) => {
    setEditingSecretId(secret.id)
    setSecretForm({ ...secret })
    setSecretDialog({ open: true, editing: true })
  }

  const submitSecret = async (event: React.FormEvent) => {
    event.preventDefault()
    if (!routerId) return
    if (!secretDialog.editing && !secretForm.password) {
      toast.error('Password is required.')
      return
    }

    try {
      const input = secretDialog.editing
        ? changedSecretInput(
            [...secrets, ...offlineSecrets].find(
              (secret) => secret.id === editingSecretId
            ),
            secretForm
          )
        : {
            name: secretForm.name?.trim() ?? '',
            ...(secretForm.password ? { password: secretForm.password } : {}),
            service: secretForm.service || 'pppoe',
            profile: secretForm.profile || 'default',
            ...(secretForm.localAddress
              ? { local_address: secretForm.localAddress }
              : {}),
            ...(secretForm.remoteAddress
              ? { remote_address: secretForm.remoteAddress }
              : {}),
            ...(secretForm.comment ? { comment: secretForm.comment } : {}),
          }

      if (secretDialog.editing && Object.keys(input).length === 0) {
        toast.info('No changes to save.')
        setSecretDialog({ open: false, editing: false })
        return
      }

      const result = await savePppoeSecret(
        Number(routerId),
        input,
        editingSecretId ?? undefined
      )
      toast.success(result.message)
      setSecretDialog({ open: false, editing: false })
      await loadRouterData()
    } catch (error) {
      toast.error(getPppoeErrorMessage(error))
    }
  }

  const openAddProfile = () => {
    setEditingProfileName(null)
    setProfileForm(emptyProfile)
    setProfileDialog({ open: true, editing: false })
  }

  const openEditProfile = (profile: ProfileRow) => {
    setEditingProfileName(profile.name)
    setProfileForm({ ...profile })
    setProfileDialog({ open: true, editing: true })
  }

  const submitProfile = async (event: React.FormEvent) => {
    event.preventDefault()
    if (!routerId) return

    const original = editingProfileName
      ? profiles.find((profile) => profile.name === editingProfileName)
      : undefined
    if (!profileForm.name?.trim()) {
      toast.error('Name is required.')
      return
    }
    const input = profileInput(profileForm, original)
    if (original && Object.keys(input).length === 0) {
      toast.info('No changes to save.')
      setProfileDialog({ open: false, editing: false })
      return
    }

    try {
      const result = await savePppoeProfile(
        Number(routerId),
        input,
        editingProfileName ?? undefined
      )
      toast.success(result.message)
      setProfileDialog({ open: false, editing: false })
      await loadRouterData()
    } catch (error) {
      toast.error(getPppoeErrorMessage(error))
    }
  }

  const confirmDeleteSecret = (secret: SecretRow) =>
    setConfirm({
      title: 'Delete PPP Secret?',
      description: `Secret "${secret.name}" will be permanently deleted. This action cannot be undone.`,
      actionLabel: 'Delete',
      destructive: true,
      onConfirm: async () => {
        const result = await deletePppoeSecret(Number(routerId), secret.id)
        toast.success(result.message)
        await loadRouterData()
      },
    })

  const confirmToggleSecret = (secret: SecretRow) => {
    const disabled = !secret.disabled
    const action = disabled ? 'Disable' : 'Enable'
    setConfirm({
      title: `${action} PPP Secret?`,
      description: `Are you sure you want to ${action.toLowerCase()} "${secret.name}"?`,
      actionLabel: action,
      destructive: disabled,
      onConfirm: async () => {
        const result = await setPppoeSecretDisabled(
          Number(routerId),
          secret.id,
          disabled
        )
        toast.success(result.message)
        await loadRouterData()
      },
    })
  }

  const toggleSecretDisabled = async (secret: SecretRow) => {
    if (!routerId || isConfirming) return
    confirmToggleSecret(secret)
  }

  const confirmDeleteProfile = (profile: ProfileRow) =>
    setConfirm({
      title: 'Delete PPP Profile?',
      description: `Profile "${profile.name}" will be permanently deleted. This action cannot be undone.`,
      actionLabel: 'Delete',
      destructive: true,
      onConfirm: async () => {
        const result = await deletePppoeProfile(Number(routerId), profile.name)
        toast.success(result.message)
        await loadRouterData()
      },
    })

  const confirmDisconnect = (session: ActiveRow) =>
    setConfirm({
      title: 'Disconnect user?',
      description: `Session "${session.name}" (${session.address}) will be disconnected.`,
      actionLabel: 'Disconnect',
      destructive: true,
      onConfirm: async () => {
        const result = await disconnectPppoeActive(Number(routerId), session.id)
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
      actionLabel: string
      doneLabel: string
      task: (item: T) => Promise<unknown>
    }
  ) =>
    setConfirm({
      title: options.title,
      description: options.description,
      actionLabel: options.actionLabel,
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
          toast.error(`${failed} failed: ${getPppoeErrorMessage(firstError)}`)
        }
        await loadRouterData()
      },
    })

  const confirmBulkDeleteSecrets = (rows: SecretRow[]) =>
    confirmBulk(rows, {
      title: `Delete ${rows.length} PPP Secrets?`,
      description:
        'The selected secrets will be permanently deleted. This action cannot be undone.',
      actionLabel: 'Delete',
      doneLabel: 'secret(s) deleted',
      task: (secret) => deletePppoeSecret(Number(routerId), secret.id),
    })

  const confirmBulkDeleteProfiles = (rows: ProfileRow[]) =>
    confirmBulk(rows, {
      title: `Delete ${rows.length} PPP Profiles?`,
      description:
        'The selected profiles will be permanently deleted. This action cannot be undone.',
      actionLabel: 'Delete',
      doneLabel: 'profile(s) deleted',
      task: (profile) => deletePppoeProfile(Number(routerId), profile.name),
    })

  const confirmBulkDisconnect = (rows: ActiveRow[]) =>
    confirmBulk(rows, {
      title: `Disconnect ${rows.length} users?`,
      description: 'The selected sessions will be disconnected.',
      actionLabel: 'Disconnect',
      doneLabel: 'session(s) disconnected',
      task: (session) => disconnectPppoeActive(Number(routerId), session.id),
    })

  const handleConfirm = async () => {
    if (!confirm) return
    setIsConfirming(true)
    try {
      await confirm.onConfirm()
      setConfirm(null)
    } catch (error) {
      toast.error(getPppoeErrorMessage(error))
    } finally {
      setIsConfirming(false)
    }
  }

  const profileOptions = profiles.map((profile) => profile.name)
  const ipPoolOptions = ipPools.map((pool) => pool.name)
  const insertQueueOptions = ['bottom', 'first', ...queueNames]
  const parentQueueOptions = ['none', ...queueNames]
  const filteredSecrets =
    secretProfileFilter === 'all'
      ? secrets
      : secrets.filter((secret) => secret.profile === secretProfileFilter)
  const currentRouter = routers.find((router) => String(router.id) === routerId)

  const changeRouter = (value: string) => {
    setSecrets([])
    setOfflineSecrets([])
    setActiveSessions([])
    setProfiles([])
    setIpPools([])
    setQueueNames([])
    setSecretProfileFilter('all')
    setIsLoadingData(Boolean(value))
    setRouterId(value)
  }

  // Dipakai di toolbar semua tabel agar MikroTik bisa dipilih dari tab mana pun.
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
            PPPoE Management
          </h1>
        </div>

        <div className='mb-3 grid grid-cols-2 gap-2 sm:gap-3 lg:grid-cols-4'>
          <StatCard
            title='Secret'
            value={secrets.length}
            description='All PPP secrets'
            icon={Users}
            tone='blue'
            loading={isLoadingData}
            active={view === 'secret'}
            onClick={() => setView('secret')}
          />
          <StatCard
            title='Online'
            value={activeSessions.length}
            description={`${percent(activeSessions.length, secrets.length)} of secrets`}
            icon={Wifi}
            tone='green'
            loading={isLoadingData}
            active={view === 'online'}
            onClick={() => setView('online')}
          />
          <StatCard
            title='Offline'
            value={offlineSecrets.length}
            description={`${percent(offlineSecrets.length, secrets.length)} of secrets`}
            icon={WifiOff}
            tone='red'
            loading={isLoadingData}
            active={view === 'offline'}
            onClick={() => setView('offline')}
          />
          <StatCard
            title='Profile'
            value={profiles.length}
            description='PPP profiles'
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
            onValueChange={(value) => setView(value as PppoeView)}
            className='w-full'
          >
            <TabsContent value='secret'>
              <DataTable
                columns={secretColumns}
                rows={filteredSecrets}
                isRefreshing={isFetching}
                onBulkDelete={confirmBulkDeleteSecrets}
                bulkDeleteLabel='Delete selected'
                getRowId={(secret) => secret.id}
                addLabel='Add Secret'
                onAdd={currentRouter ? openAddSecret : undefined}
                toolbarBeforeAdd={
                  <>
                    {routerSelect}
                    <Select
                      value={secretProfileFilter}
                      onValueChange={setSecretProfileFilter}
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
                actions={(secret) => (
                  <>
                    <Button
                      type='button'
                      size='icon'
                      variant='ghost'
                      className='size-6 p-0'
                      title={secret.disabled ? 'Enable' : 'Disable'}
                      aria-label={`${secret.disabled ? 'Enable' : 'Disable'} ${secret.name}`}
                      disabled={isConfirming}
                      onClick={() => void toggleSecretDisabled(secret)}
                    >
                      {secret.disabled ? (
                        <Power className='size-3.5' />
                      ) : (
                        <PowerOff className='size-3.5' />
                      )}
                    </Button>
                    <IconButton
                      title='Edit'
                      onClick={() => openEditSecret(secret)}
                    >
                      <Pencil className='size-3.5' />
                    </IconButton>
                    <IconButton
                      title='Delete'
                      destructive
                      onClick={() => confirmDeleteSecret(secret)}
                    >
                      <Trash2 className='size-3.5' />
                    </IconButton>
                  </>
                )}
              />
            </TabsContent>
            <TabsContent value='online'>
              <DataTable
                columns={sessionColumns}
                rows={activeSessions}
                isRefreshing={isFetching}
                onBulkDelete={confirmBulkDisconnect}
                bulkDeleteLabel='Disconnect selected'
                getRowId={(session) => session.id}
                toolbarBeforeAdd={routerSelect}
                actions={(session) => (
                  <IconButton
                    title='Disconnect / Kick'
                    destructive
                    onClick={() => confirmDisconnect(session)}
                  >
                    <Trash2 className='size-3.5' />
                  </IconButton>
                )}
              />
            </TabsContent>
            <TabsContent value='offline'>
              <DataTable
                columns={secretColumns}
                rows={offlineSecrets}
                isRefreshing={isFetching}
                onBulkDelete={confirmBulkDeleteSecrets}
                bulkDeleteLabel='Delete selected'
                getRowId={(secret) => secret.id}
                toolbarBeforeAdd={routerSelect}
                actions={(secret) => (
                  <>
                    <Button
                      type='button'
                      size='icon'
                      variant='ghost'
                      className='size-6 p-0'
                      title={secret.disabled ? 'Enable' : 'Disable'}
                      aria-label={`${secret.disabled ? 'Enable' : 'Disable'} ${secret.name}`}
                      disabled={isConfirming}
                      onClick={() => void toggleSecretDisabled(secret)}
                    >
                      {secret.disabled ? (
                        <Power className='size-3.5' />
                      ) : (
                        <PowerOff className='size-3.5' />
                      )}
                    </Button>
                    <IconButton
                      title='Edit'
                      onClick={() => openEditSecret(secret)}
                    >
                      <Pencil className='size-3.5' />
                    </IconButton>
                    <IconButton
                      title='Delete'
                      destructive
                      onClick={() => confirmDeleteSecret(secret)}
                    >
                      <Trash2 className='size-3.5' />
                    </IconButton>
                  </>
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
                getRowId={(profile) => profile.name}
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
        open={secretDialog.open}
        onOpenChange={(open) =>
          setSecretDialog((state) => ({ ...state, open }))
        }
        title={secretDialog.editing ? 'Edit PPP Secret' : 'Add PPP Secret'}
        idPrefix='secret'
        fields={secretFields.map((field) =>
          field.key === 'profile'
            ? { ...field, options: profileOptions }
            : field
        )}
        data={secretForm}
        onChange={setSecretForm}
        onSubmit={submitSecret}
        submitLabel={secretDialog.editing ? 'Save changes' : 'Add Secret'}
        passwordOptional={secretDialog.editing}
        basicKeys={['name', 'password', 'service', 'profile']}
      />
      <FormDialog
        open={profileDialog.open}
        onOpenChange={(open) =>
          setProfileDialog((state) => ({ ...state, open }))
        }
        title={profileDialog.editing ? 'Edit PPP Profile' : 'Add PPP Profile'}
        idPrefix='profile'
        fields={profileFields.map((field) =>
          field.key === 'remoteAddress'
            ? {
                ...field,
                placeholder:
                  ipPoolOptions.length > 0
                    ? 'Select IP pool'
                    : 'No IP pools available',
                options: ipPoolOptions,
              }
            : field.key === 'insertQueueBefore'
              ? { ...field, options: insertQueueOptions }
              : field.key === 'parentQueue'
                ? { ...field, options: parentQueueOptions }
                : field
        )}
        data={profileForm}
        onChange={setProfileForm}
        onSubmit={submitProfile}
        submitLabel={profileDialog.editing ? 'Save changes' : 'Add Profile'}
        basicKeys={[
          'name',
          'rateLimit',
          'localAddress',
          'remoteAddress',
          'onlyOne',
          'insertQueueBefore',
          'parentQueue',
        ]}
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
