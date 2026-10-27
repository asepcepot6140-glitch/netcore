import {
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
  type ReactNode,
} from 'react'
import { ThemeProvider } from '@mui/material'
import {
  Activity,
  Eye,
  Network,
  Power,
  PowerOff,
  Unplug,
  Zap,
} from 'lucide-react'
import { toast } from 'sonner'
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
import {
  MuiDataTable as DataTable,
  type DataTableColumn as Column,
} from '@/components/data-table'
import { Header } from '@/components/layout/header'
import { Main } from '@/components/layout/main'
import { ProfileDropdown } from '@/components/profile-dropdown'
import { StatCard } from '@/components/stat-card'
import { ThemeSwitch } from '@/components/theme-switch'
import { IconButton } from '../components/icon-button'
import { useMuiTheme } from '../hooks/use-mui-theme'
import { getMikrotikRouters, type MikrotikRouter } from '../router/mikrotik-api'
import { formatBits, formatBytes } from '../simple-queue/format'
import {
  getInterfaceDetails,
  getInterfaceErrorMessage,
  getInterfaces,
  setInterfaceDisabled,
  streamInterfaceLive,
  StreamHttpError,
  type InterfaceDetail,
  type InterfaceLive,
  type NetInterface,
} from './interface-api'

const LIST_POLL_INTERVAL_MS = 30_000
const ALL_TYPES = 'all'

type CardFilter = 'all' | 'running' | 'dynamic' | 'down'

// Down = static and no link, including disabled ones; dynamic sessions vanish when they drop, so they are never "down".
const CARD_MATCHERS: Record<CardFilter, (item: NetInterface) => boolean> = {
  all: () => true,
  running: (item) => item.running,
  dynamic: (item) => item.dynamic,
  down: (item) => !item.running && !item.dynamic,
}

// Summary lines shown above the ethernet monitor output (same data as the table row).
function getBasicRows(item: NetInterface): InterfaceDetail[] {
  const rows: InterfaceDetail[] = [
    { key: 'type', value: item.type },
    { key: 'running', value: item.running ? 'yes' : 'no' },
    { key: 'disabled', value: item.disabled ? 'yes' : 'no' },
  ]
  if (item.parent) rows.push({ key: 'interface', value: item.parent })
  if (item.last_link_up) {
    rows.push({ key: 'last-link-up', value: item.last_link_up })
  }
  if (item.last_link_down) {
    rows.push({ key: 'last-link-down', value: item.last_link_down })
  }
  return rows
}

function DetailRow({ label, value }: { label: string; value: string }) {
  return (
    <>
      <dt className='text-muted-foreground'>{label}</dt>
      <dd className='min-w-0 font-medium break-words'>{value}</dd>
    </>
  )
}
// Row with its nesting level (0 = top level, 1 = child of an interface, ...).
type InterfaceRow = NetInterface & { depth: number }

// Puts every interface right below its parent (e.g. VLANs under ether1), keeping the router's order otherwise.
function buildTree(items: NetInterface[]): InterfaceRow[] {
  const names = new Set(items.map((item) => item.name))
  const children = new Map<string, NetInterface[]>()
  const roots: NetInterface[] = []
  for (const item of items) {
    if (item.parent && names.has(item.parent)) {
      children.set(item.parent, [...(children.get(item.parent) ?? []), item])
    } else {
      roots.push(item)
    }
  }

  const rows: InterfaceRow[] = []
  const seen = new Set<string>()
  const visit = (item: NetInterface, depth: number) => {
    if (seen.has(item.id)) return
    seen.add(item.id)
    rows.push({ ...item, depth })
    for (const child of children.get(item.name) ?? []) visit(child, depth + 1)
  }
  roots.forEach((item) => visit(item, 0))
  // Safety net for parent loops, which would otherwise hide their rows.
  items.forEach((item) => visit(item, 0))
  return rows
}

// Disabled interface: all text is dimmed like Simple Queue.
function Dim({ row, children }: { row: NetInterface; children: ReactNode }) {
  if (!children) return null
  return (
    <span className={cn(row.disabled && 'text-muted-foreground/60')}>
      {children}
    </span>
  )
}

const columns: Column<InterfaceRow>[] = [
  {
    // Flags like Winbox: X = disabled, D = dynamic, R = running.
    header: 'Flags',
    align: 'center',
    hideHeader: true,
    narrow: true,
    cell: (row) => {
      const flags = [
        row.disabled && 'X',
        row.dynamic && 'D',
        row.running && 'R',
      ].filter(Boolean)
      if (flags.length === 0) return null
      const title = [
        row.disabled && 'Disabled',
        row.dynamic && 'Dynamic',
        row.running && 'Running',
      ]
        .filter(Boolean)
        .join(', ')
      return (
        <Dim row={row}>
          <span title={title} className='font-semibold'>
            {flags.join('')}
          </span>
        </Dim>
      )
    },
  },
  {
    header: 'Name',
    cell: (row) => (
      <Dim row={row}>
        <span
          className='font-semibold'
          style={{ paddingInlineStart: row.depth * 20 }}
        >
          {row.name}
        </span>
      </Dim>
    ),
    getSortValue: (row) => row.name,
  },
  {
    header: 'Type',
    cell: (row) => <Dim row={row}>{row.type}</Dim>,
    getSortValue: (row) => row.type,
  },
  {
    header: 'Tx Rate',
    cell: (row) => <Dim row={row}>{formatBits(row.tx_rate)}</Dim>,
    getSortValue: (row) => row.tx_rate,
  },
  {
    header: 'Rx Rate',
    cell: (row) => <Dim row={row}>{formatBits(row.rx_rate)}</Dim>,
    getSortValue: (row) => row.rx_rate,
  },
  {
    header: 'Bytes (Tx / Rx)',
    cell: (row) => (
      <Dim row={row}>
        {`${formatBytes(row.tx_bytes)} / ${formatBytes(row.rx_bytes)}`}
      </Dim>
    ),
    getSortValue: (row) => row.tx_bytes + row.rx_bytes,
  },
  {
    // Negotiated link speed (10Mbps, 100Mbps, 1Gbps, 10Gbps); empty without link.
    header: 'Rate',
    hideEmpty: true,
    cell: (row) => <Dim row={row}>{row.rate}</Dim>,
    getSortValue: (row) => row.rate,
  },
  {
    header: 'Last Link Up',
    hideEmpty: true,
    cell: (row) => <Dim row={row}>{row.last_link_up}</Dim>,
    getSortValue: (row) => row.last_link_up,
  },
  {
    header: 'Last Link Down',
    hideEmpty: true,
    cell: (row) => <Dim row={row}>{row.last_link_down}</Dim>,
    getSortValue: (row) => row.last_link_down,
  },
]

interface ConfirmState {
  title: string
  description: string
  actionLabel: string
  destructive?: boolean
  onConfirm: () => Promise<void>
}

export function InterfaceManagement() {
  const muiTheme = useMuiTheme()
  const [routers, setRouters] = useState<MikrotikRouter[]>([])
  const [routerId, setRouterId] = useState('')
  const [isLoadingRouters, setIsLoadingRouters] = useState(true)
  const [isFetching, setIsFetching] = useState(false)
  const [interfaces, setInterfaces] = useState<NetInterface[]>([])
  const [typeFilter, setTypeFilter] = useState(ALL_TYPES)
  const [cardFilter, setCardFilter] = useState<CardFilter>('all')
  const [details, setDetails] = useState<{
    item: NetInterface
    rows: InterfaceDetail[]
    loading: boolean
  } | null>(null)
  const [confirm, setConfirm] = useState<ConfirmState | null>(null)
  const [isConfirming, setIsConfirming] = useState(false)
  const dataRequestId = useRef(0)
  // When the last live status arrived (performance.now), to tell whether it is newer than a list response.
  const liveStatusAt = useRef(0)
  const detailRequestId = useRef(0)

  const loadRouters = useCallback(async () => {
    try {
      const result = await getMikrotikRouters()
      setRouters(result)
      setRouterId((current) =>
        result.some((router) => String(router.id) === current) ? current : ''
      )
    } catch (error) {
      toast.error(getInterfaceErrorMessage(error))
    } finally {
      setIsLoadingRouters(false)
    }
  }, [])

  const loadInterfaces = useCallback(
    async ({
      silent = false,
      replace = false,
    }: { silent?: boolean; replace?: boolean } = {}) => {
      const requestId = ++dataRequestId.current
      if (!routerId) return
      const startedAt = performance.now()
      setIsFetching(true)
      try {
        const data = await getInterfaces(Number(routerId))
        // Drop stale responses (router switched or a newer request started).
        if (requestId !== dataRequestId.current) return
        setInterfaces((current) => {
          // A live status that arrived while this request was in flight is newer than the list
          // (the server never resends an unchanged status), so it must not be overwritten.
          // After an edit the list is the truth, so it replaces everything.
          if (replace || liveStatusAt.current < startedAt) return data
          const live = new Map(current.map((item) => [item.name, item]))
          return data.map((item) => {
            const known = live.get(item.name)
            return known
              ? {
                  ...item,
                  running: known.running,
                  disabled: known.disabled,
                  tx_bytes: known.tx_bytes,
                  rx_bytes: known.rx_bytes,
                  tx_rate: known.tx_rate,
                  rx_rate: known.rx_rate,
                  rate: known.rate,
                }
              : item
          })
        })
      } catch (error) {
        // Polling failures are ignored so the table keeps its last data and no toast repeats.
        if (!silent && requestId === dataRequestId.current) {
          toast.error(getInterfaceErrorMessage(error))
        }
      } finally {
        if (requestId === dataRequestId.current) setIsFetching(false)
      }
    },
    [routerId]
  )

  useEffect(() => {
    const timer = window.setTimeout(() => void loadRouters(), 0)
    return () => window.clearTimeout(timer)
  }, [loadRouters])

  useEffect(() => {
    const timer = window.setTimeout(() => void loadInterfaces(), 0)
    return () => window.clearTimeout(timer)
  }, [loadInterfaces])

  // One request can take over a second on large routers, so a new one never starts while the
  // previous is still running; hidden tabs do not poll at all.
  useEffect(() => {
    if (!routerId) return

    let busy = false
    const timer = window.setInterval(() => {
      if (busy || document.hidden) return
      busy = true
      void loadInterfaces({ silent: true }).finally(() => {
        busy = false
      })
    }, LIST_POLL_INTERVAL_MS)

    return () => window.clearInterval(timer)
  }, [routerId, loadInterfaces])

  // Realtime part: the server pushes rates (about every second) and status/bytes (about every two
  // seconds) over one long-lived request. The slower full refresh above only adds or removes
  // interfaces and updates link times and settings.
  useEffect(() => {
    if (!routerId) return

    let controller: AbortController | null = null
    let retryTimer: number | undefined
    let stopped = false

    const applyLive = ({ rates, status, changed }: InterfaceLive) => {
      // New or removed interfaces: reload the list now. The server closes this stream, and the
      // reconnect below starts one that also covers the new interfaces.
      if (changed) void loadInterfaces({ silent: true })
      if (rates.length === 0 && status.length === 0) return
      if (status.length > 0) liveStatusAt.current = performance.now()
      const ratesByName = new Map(rates.map((rate) => [rate.name, rate]))
      const statusByName = new Map(status.map((item) => [item.name, item]))
      setInterfaces((current) => {
        let changed = false
        const next = current.map((item) => {
          const rate = ratesByName.get(item.name)
          const state = statusByName.get(item.name)
          if (!rate && !state) return item

          const updated = { ...item }
          if (rate) {
            updated.tx_rate = rate.tx_rate
            updated.rx_rate = rate.rx_rate
          }
          if (state) {
            updated.running = state.running
            updated.disabled = state.disabled
            updated.tx_bytes = state.tx_bytes
            updated.rx_bytes = state.rx_bytes
            // Only ethernet interfaces report a link speed.
            if (item.ethernet) updated.rate = state.rate
            // A stopped interface carries no traffic, and the stream sends no more rates for it.
            if (!state.running) {
              updated.tx_rate = 0
              updated.rx_rate = 0
            }
          }

          const same = (Object.keys(updated) as (keyof NetInterface)[]).every(
            (key) => updated[key] === item[key]
          )
          if (same) return item
          changed = true
          return updated
        })
        return changed ? next : current
      })
    }

    const connect = async () => {
      if (stopped || document.hidden) return
      controller = new AbortController()
      let delay = 0
      try {
        // Returns when the server closes the stream (about once a minute): reconnect right away.
        await streamInterfaceLive(
          Number(routerId),
          applyLive,
          controller.signal
        )
      } catch (error) {
        if (stopped || controller.signal.aborted) return
        // Too many streams (429) waits longer; any other failure retries shortly.
        delay =
          error instanceof StreamHttpError && error.status === 429
            ? 15_000
            : 3_000
      }
      if (!stopped) retryTimer = window.setTimeout(() => void connect(), delay)
    }

    // A hidden tab drops its stream (no one is looking) and reconnects when it becomes visible.
    const onVisibilityChange = () => {
      if (document.hidden) {
        controller?.abort()
        window.clearTimeout(retryTimer)
      } else {
        window.clearTimeout(retryTimer)
        void connect()
      }
    }

    document.addEventListener('visibilitychange', onVisibilityChange)
    void connect()

    return () => {
      stopped = true
      document.removeEventListener('visibilitychange', onVisibilityChange)
      window.clearTimeout(retryTimer)
      controller?.abort()
    }
  }, [routerId, loadInterfaces])

  const changeRouter = (value: string) => {
    setInterfaces([])
    setTypeFilter(ALL_TYPES)
    setCardFilter('all')
    setIsFetching(Boolean(value))
    setRouterId(value)
  }

  // Show reads /interface/ethernet/monitor once; the dialog opens right away with a loading state.
  const showInterface = async (item: NetInterface) => {
    const requestId = ++detailRequestId.current
    setDetails({ item, rows: [], loading: true })
    try {
      const rows = await getInterfaceDetails(Number(routerId), item.id)
      if (requestId === detailRequestId.current) {
        setDetails({ item, rows, loading: false })
      }
    } catch (error) {
      if (requestId === detailRequestId.current) {
        setDetails({ item, rows: [], loading: false })
        toast.error(getInterfaceErrorMessage(error))
      }
    }
  }

  const closeDetails = () => {
    detailRequestId.current++
    setDetails(null)
  }
  const confirmToggle = (item: NetInterface) => {
    const disabled = !item.disabled
    const action = disabled ? 'Disable' : 'Enable'
    setConfirm({
      title: `${action} Interface?`,
      description: `Are you sure you want to ${action.toLowerCase()} "${item.name}"?`,
      actionLabel: action,
      destructive: disabled,
      onConfirm: async () => {
        const result = await setInterfaceDisabled(
          Number(routerId),
          item.id,
          disabled
        )
        toast.success(result.message)
        await loadInterfaces({ replace: true })
      },
    })
  }

  const handleConfirm = async () => {
    if (!confirm) return
    setIsConfirming(true)
    try {
      await confirm.onConfirm()
      setConfirm(null)
    } catch (error) {
      toast.error(getInterfaceErrorMessage(error))
    } finally {
      setIsConfirming(false)
    }
  }

  // Every type present on the router, so the dropdown always matches the real data.
  const types = useMemo(
    () => [...new Set(interfaces.map((item) => item.type))].sort(),
    [interfaces]
  )
  // A type that disappeared (e.g. last PPPoE client left) falls back to showing everything.
  const activeType = types.includes(typeFilter) ? typeFilter : ALL_TYPES

  // Card counts follow the type dropdown, so both filters agree on what is "total".
  const typeRows = useMemo(
    () =>
      activeType === ALL_TYPES
        ? interfaces
        : interfaces.filter((item) => item.type === activeType),
    [interfaces, activeType]
  )
  const counts = useMemo(
    () => ({
      all: typeRows.length,
      running: typeRows.filter(CARD_MATCHERS.running).length,
      dynamic: typeRows.filter(CARD_MATCHERS.dynamic).length,
      down: typeRows.filter(CARD_MATCHERS.down).length,
    }),
    [typeRows]
  )

  // Children whose parent is filtered out become top-level rows (see buildTree).
  const treeRows = useMemo(
    () => buildTree(typeRows.filter(CARD_MATCHERS[cardFilter])),
    [typeRows, cardFilter]
  )

  const toggleCard = (next: CardFilter) =>
    setCardFilter((current) => (current === next ? 'all' : next))

  const typeSelect = (
    <Select
      value={activeType}
      onValueChange={setTypeFilter}
      disabled={types.length === 0}
    >
      <SelectTrigger size='sm' className='h-7 w-full sm:w-43'>
        <SelectValue />
      </SelectTrigger>
      <SelectContent>
        <SelectItem value={ALL_TYPES}>All types</SelectItem>
        {types.map((type) => (
          <SelectItem key={type} value={type}>
            {type}
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  )

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
          <h1 className='text-2xl font-bold tracking-tight'>Interface</h1>
        </div>

        <div className='mb-3 grid grid-cols-2 gap-2 sm:gap-3 lg:grid-cols-4'>
          <StatCard
            title='Total'
            value={counts.all}
            description='All interfaces'
            icon={Network}
            tone='blue'
            loading={isFetching && interfaces.length === 0}
            active={cardFilter === 'all'}
            onClick={() => setCardFilter('all')}
          />
          <StatCard
            title='Running'
            value={counts.running}
            description='Link up'
            icon={Activity}
            tone='green'
            loading={isFetching && interfaces.length === 0}
            active={cardFilter === 'running'}
            onClick={() => toggleCard('running')}
          />
          <StatCard
            title='Dynamic'
            value={counts.dynamic}
            description='Created by RouterOS'
            icon={Zap}
            tone='violet'
            loading={isFetching && interfaces.length === 0}
            active={cardFilter === 'dynamic'}
            onClick={() => toggleCard('dynamic')}
          />
          <StatCard
            title='Down'
            value={counts.down}
            description='No link / disabled'
            icon={Unplug}
            tone='red'
            loading={isFetching && interfaces.length === 0}
            active={cardFilter === 'down'}
            onClick={() => toggleCard('down')}
          />
        </div>

        <ThemeProvider theme={muiTheme}>
          <DataTable
            columns={columns}
            rows={treeRows}
            showIndex={false}
            isRefreshing={isFetching}
            getRowId={(item) => item.id}
            toolbarBeforeAdd={
              <>
                {routerSelect}
                {typeSelect}
              </>
            }
            actions={(item) => (
              <>
                <Button
                  type='button'
                  size='icon'
                  variant='ghost'
                  className='size-6 p-0'
                  title={item.disabled ? 'Enable' : 'Disable'}
                  aria-label={`${item.disabled ? 'Enable' : 'Disable'} ${item.name}`}
                  disabled={isConfirming || item.dynamic}
                  onClick={() => confirmToggle(item)}
                >
                  {item.disabled ? (
                    <Power className='size-3.5' />
                  ) : (
                    <PowerOff className='size-3.5' />
                  )}
                </Button>
                <IconButton
                  title='Show'
                  onClick={() => void showInterface(item)}
                >
                  <Eye className='size-3.5' />
                </IconButton>
              </>
            )}
          />
        </ThemeProvider>
      </Main>

      <Dialog
        open={details !== null}
        onOpenChange={(o) => !o && closeDetails()}
      >
        <DialogContent
          className='max-h-[84vh] overflow-y-auto sm:max-w-lg'
          showCloseButton={false}
        >
          <DialogHeader>
            <DialogTitle>{details?.item.name}</DialogTitle>
            <DialogDescription className='sr-only'>
              Interface details
            </DialogDescription>
          </DialogHeader>
          {details?.loading ? (
            <p className='text-sm text-muted-foreground'>Loading...</p>
          ) : (
            <dl className='grid grid-cols-[auto_1fr] gap-x-4 gap-y-1 text-sm'>
              {details &&
                getBasicRows(details.item).map(({ key, value }) => (
                  <DetailRow key={key} label={key} value={value} />
                ))}
              {details?.rows.map(({ key, value }) => (
                <DetailRow key={key} label={key} value={value} />
              ))}
            </dl>
          )}
          {details && !details.loading && details.rows.length === 0 && (
            <p className='text-sm text-muted-foreground'>
              No more details for this interface.
            </p>
          )}
          <DialogFooter>
            <Button variant='outline' onClick={closeDetails}>
              Close
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
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
