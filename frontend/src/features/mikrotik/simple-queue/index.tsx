import {
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
  type ReactNode,
} from 'react'
import { ThemeProvider } from '@mui/material'
import { Gauge, Pencil, Pin, Power, PowerOff, Trash2, Zap } from 'lucide-react'
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
  bucketToInput,
  formatBitsSide,
  formatBytesPair,
  formatLimit,
  formatTarget,
  limitToInput,
  pairToInput,
  pairSide,
  pairTotal,
  rateLevel,
  targetToInput,
  timeToInput,
  type RateLevel,
} from './format'
import {
  deleteSimpleQueue,
  getQueueParents,
  getQueueTypes,
  getPacketMarks,
  getSimpleQueueErrorMessage,
  getSimpleQueues,
  saveSimpleQueue,
  setSimpleQueueDisabled,
  type SimpleQueue,
  type SimpleQueueLive,
} from './simple-queue-api'

// Live updates arrive over one stream; the full list only repairs drift now and then.
const QUEUE_LIST_POLL_INTERVAL_MS = 30_000
const PRIORITIES = ['1', '2', '3', '4', '5', '6', '7', '8']
const RATE_PRESETS = [
  '64k',
  '128k',
  '256k',
  '384k',
  '512k',
  '768k',
  '1M',
  '2M',
  '3M',
  '4M',
  '5M',
  '10M',
]

// Form keys match the API keys so no mapping table is needed.
const formKeys = [
  'name',
  'target',
  'max_limit',
  'burst_limit',
  'burst_threshold',
  'burst_time',
  'comment',
  'packet_marks',
  'limit_at',
  'priority',
  'bucket_size',
  'queue_type',
  'parent',
] as const

type QueueForm = Record<(typeof formKeys)[number], string>

const emptyForm = Object.fromEntries(
  formKeys.map((key) => [key, ''])
) as QueueForm

// Nilai awal dialog Add, sama dengan bawaan RouterOS (0 = unlimited, burst time 0 detik).
const addDefaults: QueueForm = {
  ...emptyForm,
  max_limit: '0/0',
  burst_limit: '0/0',
  burst_threshold: '0/0',
  burst_time: '0/0',
  limit_at: '0/0',
  priority: '8/8',
  bucket_size: '0.100/0.100',
  queue_type: 'default-small/default-small',
  parent: 'none',
}

const formTabs: FormTab[] = [
  {
    value: 'general',
    label: 'General',
    keys: [
      'name',
      'target',
      'max_limit',
      'burst_limit',
      'burst_threshold',
      'burst_time',
      'comment',
    ],
  },
  {
    value: 'advanced',
    label: 'Advanced',
    keys: [
      'packet_marks',
      'limit_at',
      'priority',
      'bucket_size',
      'queue_type',
      'parent',
    ],
  },
]

const formFields: Field<QueueForm>[] = [
  { key: 'name', label: 'Name', placeholder: 'Queue name', required: true },
  {
    key: 'target',
    label: 'Target',
    placeholder: '192.168.1.0/24, 10.0.0.5/32',
    required: true,
  },
  {
    key: 'max_limit',
    label: 'Max Limit',
    placeholder: '5M/10M',
    type: 'rate',
    unlimited: true,
    presets: RATE_PRESETS,
  },
  {
    key: 'burst_limit',
    label: 'Burst Limit',
    placeholder: '10M/20M',
    type: 'rate',
    unlimited: true,
  },
  {
    key: 'burst_threshold',
    label: 'Burst Threshold',
    placeholder: '3M/6M',
    type: 'rate',
    unlimited: true,
  },
  { key: 'burst_time', label: 'Burst Time', placeholder: '20s', type: 'pair' },
  { key: 'comment', label: 'Comment', placeholder: 'Comment' },
  {
    key: 'packet_marks',
    label: 'Packet Marks',
    placeholder: '',
    type: 'multi',
  },
  {
    key: 'limit_at',
    label: 'Limit At',
    placeholder: '1M/2M',
    type: 'rate',
    unlimited: true,
    presets: RATE_PRESETS,
  },
  {
    key: 'priority',
    label: 'Priority',
    placeholder: '8',
    type: 'pair',
    options: PRIORITIES,
  },
  {
    key: 'bucket_size',
    label: 'Bucket Size',
    placeholder: '0.1',
    type: 'pair',
  },
  {
    key: 'queue_type',
    label: 'Queue Type',
    placeholder: 'Select queue type',
    type: 'pair',
  },
  { key: 'parent', label: 'Parent', placeholder: 'none' },
]

function queueToForm(queue: SimpleQueue): QueueForm {
  return {
    name: queue.name,
    target: targetToInput(queue.target),
    max_limit: limitToInput(queue.max_limit, true),
    burst_limit: limitToInput(queue.burst_limit, true),
    burst_threshold: limitToInput(queue.burst_threshold, true),
    burst_time: timeToInput(queue.burst_time),
    comment: queue.comment,
    packet_marks: queue.packet_marks,
    limit_at: limitToInput(queue.limit_at, true),
    priority: pairToInput(queue.priority),
    bucket_size: bucketToInput(queue.bucket_size),
    queue_type: pairToInput(queue.queue_type),
    parent: queue.parent || 'none',
  }
}

// Add sends only filled fields; edit sends only fields that differ (empty resets the value).
function changedInput(form: QueueForm, original?: QueueForm) {
  const input: Partial<QueueForm> = {}
  for (const key of formKeys) {
    const value = form[key].trim()
    if (original) {
      if (value !== original[key]) input[key] = value
    } else if (value !== '') {
      input[key] = value
    }
  }
  return input
}

function Tag({ children }: { children: string }) {
  return (
    <span className='rounded border border-destructive/40 px-1 text-[10px] font-medium text-destructive'>
      {children}
    </span>
  )
}

// Queue disabled: seluruh teks dibuat abu-abu samar seperti placeholder.
function Dim({ row, children }: { row: SimpleQueue; children: ReactNode }) {
  return (
    <span className={cn(row.disabled && 'text-muted-foreground/60')}>
      {children}
    </span>
  )
}

const RATE_SIGNALS: Record<RateLevel, { color: string; label: string }> = {
  low: { color: 'text-green-500', label: 'Low load' },
  medium: { color: 'text-yellow-500', label: 'Medium load' },
  high: { color: 'text-red-500', label: 'High load' },
}

// Ikon seperti Winbox: 3 batang mendatar rata kanan (pendek di atas, panjang di bawah).
// Semua batang menyala penuh dengan warna sesuai beban; queue disabled dibuat abu-abu samar.
const SIGNAL_BAR_WIDTHS = [10, 16, 22]
const SIGNAL_RIGHT_EDGE = 23

function RateSignal({ row }: { row: SimpleQueue }) {
  const signal = row.disabled
    ? { color: 'text-muted-foreground/60', label: 'Disabled' }
    : RATE_SIGNALS[rateLevel(row.rate, row.max_limit)]

  return (
    <svg
      role='img'
      aria-label={signal.label}
      viewBox='0 0 24 20'
      className={cn('h-5 w-6 shrink-0', signal.color)}
    >
      {SIGNAL_BAR_WIDTHS.map((width, index) => (
        <rect
          key={width}
          x={SIGNAL_RIGHT_EDGE - width}
          y={2 + index * 6}
          width={width}
          height={4}
          rx={1}
          fill='currentColor'
        />
      ))}
    </svg>
  )
}

// Baris dengan tingkat bersarang (0 = teratas, 1 = anak dari queue lain, ...).
type QueueRow = SimpleQueue & { depth: number }

// Menaruh tiap queue tepat di bawah parent-nya dengan urutan lain tetap seperti dari server.
// Queue yang parent-nya "none", bukan queue simple, atau tersaring jadi baris teratas.
function buildTree(items: SimpleQueue[]): QueueRow[] {
  const names = new Set(items.map((item) => item.name))
  const children = new Map<string, SimpleQueue[]>()
  const roots: SimpleQueue[] = []
  for (const item of items) {
    if (item.parent && item.parent !== item.name && names.has(item.parent)) {
      const siblings = children.get(item.parent)
      if (siblings) siblings.push(item)
      else children.set(item.parent, [item])
    } else {
      roots.push(item)
    }
  }

  const rows: QueueRow[] = []
  const seen = new Set<string>()
  const visit = (item: SimpleQueue, depth: number) => {
    if (seen.has(item.id)) return
    seen.add(item.id)
    rows.push({ ...item, depth })
    for (const child of children.get(item.name) ?? []) visit(child, depth + 1)
  }
  roots.forEach((item) => visit(item, 0))
  // Jaring pengaman untuk parent yang saling melingkar, yang kalau tidak akan menyembunyikan barisnya.
  items.forEach((item) => visit(item, 0))
  return rows
}

const columns: Column<QueueRow>[] = [
  {
    // Flag seperti Winbox: X = disabled, D = dynamic (bisa muncul bersamaan).
    header: 'Flags',
    align: 'center',
    hideHeader: true,
    narrow: true,
    cell: (row) => {
      const flags = [row.disabled && 'X', row.dynamic && 'D'].filter(Boolean)
      if (flags.length === 0) return null
      const title = [row.disabled && 'Disabled', row.dynamic && 'Dynamic']
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
      <span
        className='flex items-center gap-1.5'
        style={{ paddingInlineStart: row.depth * 20 }}
      >
        <RateSignal row={row} />
        <Dim row={row}>
          <span className='font-semibold'>{row.name}</span>
        </Dim>
        {row.invalid && !row.disabled && <Tag>Invalid</Tag>}
      </span>
    ),
    getSortValue: (row) => row.name,
  },
  {
    header: 'Target',
    cell: (row) => <Dim row={row}>{formatTarget(row.target)}</Dim>,
    getSortValue: (row) => row.target,
  },
  {
    header: 'Max Limit',
    cell: (row) => <Dim row={row}>{formatLimit(row.max_limit)}</Dim>,
    getSortValue: (row) => pairTotal(row.max_limit),
  },
  {
    header: 'Upload',
    cell: (row) => <Dim row={row}>{formatBitsSide(row.rate, 'up')}</Dim>,
    getSortValue: (row) => pairSide(row.rate, 'up'),
  },
  {
    header: 'Download',
    cell: (row) => <Dim row={row}>{formatBitsSide(row.rate, 'down')}</Dim>,
    getSortValue: (row) => pairSide(row.rate, 'down'),
  },
  {
    header: 'Bytes',
    cell: (row) => <Dim row={row}>{formatBytesPair(row.bytes)}</Dim>,
    getSortValue: (row) => pairTotal(row.bytes),
  },
  {
    header: 'Comment',
    hideEmpty: true,
    cell: (row) =>
      row.comment ? (
        <Dim row={row}>
          <span title={row.comment} className='block max-w-48 truncate'>
            {row.comment}
          </span>
        </Dim>
      ) : null,
    getSortValue: (row) => row.comment,
  },
]

type DialogMode = 'add' | 'edit' | 'view'

type QueueFilter = 'all' | 'dynamic' | 'static' | 'disabled'

const QUEUE_FILTERS: Record<QueueFilter, (queue: SimpleQueue) => boolean> = {
  all: () => true,
  dynamic: (queue) => queue.dynamic,
  static: (queue) => !queue.dynamic,
  disabled: (queue) => queue.disabled,
}

interface ConfirmState {
  title: string
  description: string
  actionLabel: string
  destructive?: boolean
  onConfirm: () => Promise<void>
}

export function SimpleQueueManagement() {
  const muiTheme = useMuiTheme()
  const [routers, setRouters] = useState<MikrotikRouter[]>([])
  const [routerId, setRouterId] = useState('')
  const [isLoadingRouters, setIsLoadingRouters] = useState(true)
  const [isFetching, setIsFetching] = useState(false)
  const [filter, setFilter] = useState<QueueFilter>('all')
  // Skeleton kartu hanya untuk muat pertama per router, bukan tiap polling.
  const [isLoadingData, setIsLoadingData] = useState(false)
  const [queues, setQueues] = useState<SimpleQueue[]>([])
  const [typeNames, setTypeNames] = useState<string[]>([])
  const [parentNames, setParentNames] = useState<string[]>([])
  const [markNames, setMarkNames] = useState<string[]>([])
  const [dialog, setDialog] = useState<{ open: boolean; mode: DialogMode }>({
    open: false,
    mode: 'add',
  })
  const [form, setForm] = useState<QueueForm>(emptyForm)
  const [editing, setEditing] = useState<SimpleQueue | null>(null)
  const [confirm, setConfirm] = useState<ConfirmState | null>(null)
  const [isConfirming, setIsConfirming] = useState(false)
  const dataRequestId = useRef(0)
  // When the last live event arrived; an older list response must not overwrite newer live rows.
  const liveAt = useRef(0)

  const loadRouters = useCallback(async () => {
    try {
      const result = await getMikrotikRouters()
      setRouters(result)
      setRouterId((current) =>
        result.some((router) => String(router.id) === current) ? current : ''
      )
    } catch (error) {
      toast.error(getSimpleQueueErrorMessage(error))
    } finally {
      setIsLoadingRouters(false)
    }
  }, [])

  const loadQueues = useCallback(
    async ({ silent = false }: { silent?: boolean } = {}) => {
      const requestId = ++dataRequestId.current
      if (!routerId) return
      const startedAt = performance.now()
      setIsFetching(true)
      try {
        const data = await getSimpleQueues(Number(routerId))
        // Drop stale responses (router switched or a newer request started).
        if (requestId !== dataRequestId.current) return
        // A background refresh that started before the latest live event is older than what is shown.
        if (silent && liveAt.current > startedAt) return
        setQueues(data)
      } catch (error) {
        // Polling failures are ignored so the table keeps its last data and no toast repeats.
        if (!silent && requestId === dataRequestId.current) {
          toast.error(getSimpleQueueErrorMessage(error))
        }
      } finally {
        if (requestId === dataRequestId.current) {
          setIsFetching(false)
          setIsLoadingData(false)
        }
      }
    },
    [routerId]
  )

  // Dropdown options rarely change, so they skip polling and fail quietly.
  const loadOptions = useCallback(async () => {
    if (!routerId) return
    const id = Number(routerId)
    const [types, parents, marks] = await Promise.all([
      getQueueTypes(id).catch(() => [] as string[]),
      getQueueParents(id).catch(() => [] as string[]),
      getPacketMarks(id).catch(() => [] as string[]),
    ])
    setTypeNames(types)
    setParentNames(parents)
    setMarkNames(marks)
  }, [routerId])

  useEffect(() => {
    const timer = window.setTimeout(() => void loadRouters(), 0)
    return () => window.clearTimeout(timer)
  }, [loadRouters])

  useEffect(() => {
    const timer = window.setTimeout(() => void loadOptions(), 0)
    return () => window.clearTimeout(timer)
  }, [loadOptions])

  useEffect(() => {
    const timer = window.setTimeout(() => void loadQueues(), 0)
    return () => window.clearTimeout(timer)
  }, [loadQueues])

  // Slow full refresh; rate and bytes come from the live stream below.
  useEffect(() => {
    if (!routerId) return

    let busy = false
    const timer = window.setInterval(() => {
      if (busy || document.hidden) return
      busy = true
      void loadQueues({ silent: true }).finally(() => {
        busy = false
      })
    }, QUEUE_LIST_POLL_INTERVAL_MS)

    return () => window.clearInterval(timer)
  }, [routerId, loadQueues])

  const applyLive = useCallback((live: SimpleQueueLive) => {
    if (!live.queues) return
    liveAt.current = performance.now()
    setQueues((current) =>
      applyDelta(current, live.queues, (queue) => queue.id)
    )
  }, [])
  useLiveStream<SimpleQueueLive>('/queue/simple/live', routerId, applyLive)

  const changeRouter = (value: string) => {
    setQueues([])
    setTypeNames([])
    setParentNames([])
    setMarkNames([])
    setIsFetching(Boolean(value))
    setIsLoadingData(Boolean(value))
    setRouterId(value)
  }

  const openAdd = () => {
    setEditing(null)
    setForm({ ...addDefaults })
    setDialog({ open: true, mode: 'add' })
    void loadOptions()
  }

  // Dynamic queues are managed by RouterOS, so they open read-only.
  const openQueue = (queue: SimpleQueue) => {
    setEditing(queue)
    setForm(queueToForm(queue))
    setDialog({ open: true, mode: queue.dynamic ? 'view' : 'edit' })
    void loadOptions()
  }

  const closeDialog = () => setDialog((state) => ({ ...state, open: false }))

  const submit = async (event: React.FormEvent) => {
    event.preventDefault()
    if (!routerId || dialog.mode === 'view') return
    if (!form.name.trim()) {
      toast.error('Name is required.')
      return
    }
    if (!form.target.trim()) {
      toast.error('Target is required.')
      return
    }

    const original = editing ? queueToForm(editing) : undefined
    const input = changedInput(form, original)
    if (editing && Object.keys(input).length === 0) {
      toast.info('No changes to save.')
      closeDialog()
      return
    }

    try {
      const result = await saveSimpleQueue(Number(routerId), input, editing?.id)
      toast.success(result.message)
      closeDialog()
      await loadQueues()
    } catch (error) {
      toast.error(getSimpleQueueErrorMessage(error))
    }
  }

  const confirmDelete = (queue: SimpleQueue) =>
    setConfirm({
      title: 'Delete Simple Queue?',
      description: `Queue "${queue.name}" will be permanently deleted. This action cannot be undone.`,
      actionLabel: 'Delete',
      destructive: true,
      onConfirm: async () => {
        const result = await deleteSimpleQueue(Number(routerId), queue.id)
        toast.success(result.message)
        await loadQueues()
      },
    })

  const confirmToggle = (queue: SimpleQueue) => {
    const disabled = !queue.disabled
    const action = disabled ? 'Disable' : 'Enable'
    setConfirm({
      title: `${action} Simple Queue?`,
      description: `Are you sure you want to ${action.toLowerCase()} "${queue.name}"?`,
      actionLabel: action,
      destructive: disabled,
      onConfirm: async () => {
        const result = await setSimpleQueueDisabled(
          Number(routerId),
          queue.id,
          disabled
        )
        toast.success(result.message)
        await loadQueues()
      },
    })
  }

  const confirmBulkDelete = (rows: SimpleQueue[]) => {
    // Dynamic queues cannot be deleted, so they are left out of the selection.
    const deletable = rows.filter((row) => !row.dynamic)
    if (deletable.length === 0) {
      toast.info('Dynamic queues cannot be deleted.')
      return
    }

    const skipped = rows.length - deletable.length
    setConfirm({
      title: `Delete ${deletable.length} Simple Queues?`,
      description:
        'The selected queues will be permanently deleted. This action cannot be undone.' +
        (skipped > 0 ? ` ${skipped} dynamic queue(s) will be skipped.` : ''),
      actionLabel: 'Delete',
      destructive: true,
      onConfirm: async () => {
        const { succeeded, failed, firstError } = await runBulk(
          deletable,
          (queue) => deleteSimpleQueue(Number(routerId), queue.id)
        )
        if (succeeded > 0) toast.success(`${succeeded} queue(s) deleted.`)
        if (failed > 0) {
          toast.error(
            `${failed} failed: ${getSimpleQueueErrorMessage(firstError)}`
          )
        }
        await loadQueues()
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
      toast.error(getSimpleQueueErrorMessage(error))
    } finally {
      setIsConfirming(false)
    }
  }

  const currentRouter = routers.find((router) => String(router.id) === routerId)

  const dynamicCount = queues.filter((queue) => queue.dynamic).length
  const disabledCount = queues.filter((queue) => queue.disabled).length

  // Anak yang parent-nya tersaring filter menjadi baris teratas (lihat buildTree).
  const visibleQueues = useMemo(
    () => buildTree(queues.filter(QUEUE_FILTERS[filter])),
    [queues, filter]
  )

  // Klik kartu yang sedang aktif mengembalikan ke semua queue.
  const toggleFilter = (next: QueueFilter) =>
    setFilter((current) => (current === next ? 'all' : next))

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

  const dialogTitle =
    dialog.mode === 'add'
      ? 'Add Simple Queue'
      : dialog.mode === 'edit'
        ? 'Edit Simple Queue'
        : 'Simple Queue (Dynamic)'

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
          <h1 className='text-2xl font-bold tracking-tight'>Simple Queue</h1>
        </div>

        <div className='mb-3 grid grid-cols-2 gap-2 sm:gap-3 lg:grid-cols-4'>
          <StatCard
            title='Total'
            value={queues.length}
            description='All queues'
            icon={Gauge}
            tone='blue'
            loading={isLoadingData}
            active={filter === 'all'}
            onClick={() => setFilter('all')}
          />
          <StatCard
            title='Dynamic'
            value={dynamicCount}
            description='Created by RouterOS'
            icon={Zap}
            tone='violet'
            loading={isLoadingData}
            active={filter === 'dynamic'}
            onClick={() => toggleFilter('dynamic')}
          />
          <StatCard
            title='Static'
            value={queues.length - dynamicCount}
            description='Manual queues'
            icon={Pin}
            tone='green'
            loading={isLoadingData}
            active={filter === 'static'}
            onClick={() => toggleFilter('static')}
          />
          <StatCard
            title='Disabled'
            value={disabledCount}
            description='Inactive queues'
            icon={PowerOff}
            tone='red'
            loading={isLoadingData}
            active={filter === 'disabled'}
            onClick={() => toggleFilter('disabled')}
          />
        </div>

        <ThemeProvider theme={muiTheme}>
          <DataTable
            columns={columns}
            rows={visibleQueues}
            isRefreshing={isFetching}
            onBulkDelete={confirmBulkDelete}
            bulkDeleteLabel='Delete selected'
            getRowId={(queue) => queue.id}
            addLabel='Add Queue'
            onAdd={currentRouter ? openAdd : undefined}
            toolbarBeforeAdd={routerSelect}
            actions={(queue) => (
              <>
                <Button
                  type='button'
                  size='icon'
                  variant='ghost'
                  className='size-6 p-0'
                  title={queue.disabled ? 'Enable' : 'Disable'}
                  aria-label={`${queue.disabled ? 'Enable' : 'Disable'} ${queue.name}`}
                  disabled={isConfirming}
                  onClick={() => confirmToggle(queue)}
                >
                  {queue.disabled ? (
                    <Power className='size-3.5' />
                  ) : (
                    <PowerOff className='size-3.5' />
                  )}
                </Button>
                <IconButton title='Edit' onClick={() => openQueue(queue)}>
                  <Pencil className='size-3.5' />
                </IconButton>
                <IconButton
                  title='Delete'
                  destructive
                  onClick={() => confirmDelete(queue)}
                >
                  <Trash2 className='size-3.5' />
                </IconButton>
              </>
            )}
          />
        </ThemeProvider>
      </Main>

      <FormDialog
        open={dialog.open}
        onOpenChange={(open) => setDialog((state) => ({ ...state, open }))}
        title={dialogTitle}
        idPrefix='simple-queue'
        fields={formFields.map((field) =>
          field.key === 'queue_type'
            ? { ...field, options: typeNames }
            : field.key === 'packet_marks'
              ? { ...field, options: ['no-mark', ...markNames] }
              : field.key === 'parent'
                ? { ...field, options: ['none', ...parentNames] }
                : field
        )}
        data={form}
        onChange={(data) => setForm(data as QueueForm)}
        onSubmit={submit}
        submitLabel={dialog.mode === 'edit' ? 'Save changes' : 'Add Queue'}
        tabs={formTabs}
        readOnly={dialog.mode === 'view'}
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
