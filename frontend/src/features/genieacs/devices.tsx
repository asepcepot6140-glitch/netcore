import { useCallback, useEffect, useRef, useState } from 'react'
import { useNavigate, useSearch } from '@tanstack/react-router'
import { ThemeProvider } from '@mui/material'
import { Eye, RefreshCw, Trash2 } from 'lucide-react'
import { toast } from 'sonner'
import { getAuthErrorMessage } from '@/lib/auth-api'
import { runBulk } from '@/lib/run-bulk'
import { Button } from '@/components/ui/button'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { ConfirmDialog } from '@/components/confirm-dialog'
import { MuiDataTable, type DataTableColumn } from '@/components/data-table'
import { Header } from '@/components/layout/header'
import { Main } from '@/components/layout/main'
import { LoadingOverlay } from '@/components/loading-overlay'
import { ProfileDropdown } from '@/components/profile-dropdown'
import { ThemeSwitch } from '@/components/theme-switch'
import { isStaleDevice, useNow } from './device-freshness'
import { DeviceStats } from './device-stats'
import {
  deleteGenieACSDevice,
  getGenieACSConfigs,
  getGenieACSDevices,
  summonGenieACSDevice,
  type GenieACSConfig,
  type GenieACSDevice,
} from './genieacs-api'
import { useMuiTheme } from './use-mui-theme'

const DEVICES_POLL_INTERVAL_MS = 5_000

const columns: DataTableColumn<GenieACSDevice>[] = [
  {
    header: 'Status',
    cell: (device) => (
      <span
        className={
          device.status === 'online'
            ? 'text-xs font-medium text-emerald-600 dark:text-emerald-400'
            : 'text-xs font-medium text-muted-foreground'
        }
      >
        {device.status}
      </span>
    ),
    getSortValue: (device) => device.status,
  },
  {
    header: 'Username',
    cell: (device) => device.pppoe_username || '-',
    getSortValue: (device) => device.pppoe_username,
  },
  {
    header: 'RX Power',
    cell: (device) =>
      device.rx_power && device.rx_power !== 'N/A'
        ? `${device.rx_power} dBm`
        : 'N/A',
  },
  { header: 'SSID', cell: (device) => device.ssid || '-' },
  { header: 'Product Class', cell: (device) => device.product_class || '-' },
  {
    header: 'Serial Number',
    cell: (device) => device.serial_number || device.id,
  },
]

export function GenieACSDevices() {
  const navigate = useNavigate()
  const { configId } = useSearch({
    from: '/_authenticated/genieacs/devices',
  })
  const [configs, setConfigs] = useState<GenieACSConfig[]>([])
  const [selectedId, setSelectedId] = useState(configId ? String(configId) : '')
  const [productClassFilter, setProductClassFilter] = useState('all')
  const [statusFilter, setStatusFilter] = useState('all')
  const [devices, setDevices] = useState<GenieACSDevice[]>([])
  const [isLoadingConfigs, setIsLoadingConfigs] = useState(true)
  const [isLoadingDevices, setIsLoadingDevices] = useState(false)
  const [isFetching, setIsFetching] = useState(false)
  const [summoningDeviceId, setSummoningDeviceId] = useState<string | null>(
    null
  )
  const deviceRequestId = useRef(0)
  const [deviceToDelete, setDeviceToDelete] = useState<GenieACSDevice | null>(
    null
  )
  const [isDeleting, setIsDeleting] = useState(false)
  const [devicesToDelete, setDevicesToDelete] = useState<GenieACSDevice[]>([])
  const muiTheme = useMuiTheme()
  const now = useNow()
  // Skeleton/loading state only when there is no data yet;
  // on refresh keep the current data and only spin the refresh icon.
  const isInitialLoading =
    isLoadingConfigs || (isLoadingDevices && devices.length === 0)
  const productClasses = Array.from(
    new Set(devices.map((device) => device.product_class).filter(Boolean))
  ).sort((first, second) => first.localeCompare(second))
  const productClassDevices = devices.filter(
    (device) =>
      productClassFilter === 'all' ||
      device.product_class === productClassFilter
  )
  const filteredDevices = productClassDevices.filter((device) => {
    if (statusFilter === 'all') return true
    if (statusFilter === 'stale') return isStaleDevice(device, now)
    return device.status === statusFilter
  })

  const loadConfigs = useCallback(async () => {
    try {
      const result = await getGenieACSConfigs()
      setConfigs(result)
      setSelectedId((current) =>
        result.some((config) => String(config.id) === current) ? current : ''
      )
    } catch (error) {
      toast.error(getAuthErrorMessage(error))
    } finally {
      setIsLoadingConfigs(false)
    }
  }, [])

  const loadDevices = useCallback(
    async ({ silent = false }: { silent?: boolean } = {}) => {
      const requestId = ++deviceRequestId.current
      if (!selectedId) return
      setIsFetching(true)
      try {
        const result = await getGenieACSDevices(Number(selectedId))
        if (requestId === deviceRequestId.current) setDevices(result.data)
      } catch (error) {
        // Polling failures are ignored so the table keeps its last data
        // and no toast is spammed every interval.
        if (!silent && requestId === deviceRequestId.current) {
          setDevices([])
          toast.error(getAuthErrorMessage(error))
        }
      } finally {
        if (requestId === deviceRequestId.current) {
          setIsLoadingDevices(false)
          setIsFetching(false)
        }
      }
    },
    [selectedId]
  )

  async function summonDevice(device: GenieACSDevice) {
    if (!selectedId || summoningDeviceId) return
    setSummoningDeviceId(device.id)
    try {
      const result = await summonGenieACSDevice(Number(selectedId), device.id)
      if (result.applied) {
        toast.success(result.message)
        setIsLoadingDevices(true)
        await loadDevices()
      } else {
        toast.error(result.message)
      }
    } catch (error) {
      toast.error(getAuthErrorMessage(error))
    } finally {
      setSummoningDeviceId(null)
    }
  }

  async function confirmDeleteDevice() {
    if (!selectedId || !deviceToDelete || isDeleting) return
    setIsDeleting(true)
    try {
      const result = await deleteGenieACSDevice(
        Number(selectedId),
        deviceToDelete.id
      )
      toast.success(result.message)
      setDeviceToDelete(null)
      setIsLoadingDevices(true)
      await loadDevices()
    } catch (error) {
      toast.error(getAuthErrorMessage(error))
    } finally {
      setIsDeleting(false)
    }
  }

  async function confirmBulkDelete() {
    if (!selectedId || devicesToDelete.length === 0 || isDeleting) return
    setIsDeleting(true)
    try {
      const { succeeded, failed, firstError } = await runBulk(
        devicesToDelete,
        (device) => deleteGenieACSDevice(Number(selectedId), device.id)
      )
      if (succeeded > 0) toast.success(`${succeeded} device(s) deleted.`)
      if (failed > 0) {
        toast.error(
          `${failed} device(s) failed to delete: ${getAuthErrorMessage(firstError)}`
        )
      }
      setDevicesToDelete([])
      setIsLoadingDevices(true)
      await loadDevices()
    } finally {
      setIsDeleting(false)
    }
  }

  useEffect(() => {
    const timer = window.setTimeout(() => void loadConfigs(), 0)
    return () => window.clearTimeout(timer)
  }, [loadConfigs])

  useEffect(() => {
    const timer = window.setTimeout(() => void loadDevices(), 0)
    return () => window.clearTimeout(timer)
  }, [loadDevices])

  // Realtime polling: only while a GenieACS is selected, no caching
  useEffect(() => {
    if (!selectedId) return

    let busy = false
    const timer = window.setInterval(() => {
      // Skip while the tab is hidden or the previous poll is still running
      if (busy || document.hidden) return
      busy = true
      void loadDevices({ silent: true }).finally(() => {
        busy = false
      })
    }, DEVICES_POLL_INTERVAL_MS)

    return () => window.clearInterval(timer)
  }, [selectedId, loadDevices])

  return (
    <>
      <LoadingOverlay open={summoningDeviceId !== null} />
      <Header fixed>
        <div className='ms-auto flex items-center gap-2'>
          <ThemeSwitch />
          <ProfileDropdown />
        </div>
      </Header>
      <Main fixed fluid className='min-h-0 !overflow-y-auto'>
        <div className='mb-3'>
          <h1 className='text-2xl font-bold tracking-tight'>
            GenieACS Devices
          </h1>
        </div>
        <DeviceStats
          devices={productClassDevices}
          now={now}
          loading={isInitialLoading}
          statusFilter={statusFilter}
          onStatusFilterChange={setStatusFilter}
        />
        <ThemeProvider theme={muiTheme}>
          <MuiDataTable
            columns={columns}
            rows={filteredDevices}
            getRowId={(device) => device.id}
            isLoading={isInitialLoading}
            isRefreshing={isFetching}
            loadingLabel='Loading devices...'
            onBulkDelete={setDevicesToDelete}
            bulkDeleteLabel='Delete selected'
            toolbarAfterAdd={
              <>
                <Select
                  value={selectedId}
                  onValueChange={(value) => {
                    setDevices([])
                    setIsLoadingDevices(Boolean(value))
                    setSelectedId(value)
                    void navigate({
                      to: '/genieacs/devices',
                      search: {},
                    })
                  }}
                  disabled={isLoadingConfigs || configs.length === 0}
                >
                  <SelectTrigger size='sm' className='h-7 w-full sm:w-43'>
                    <SelectValue
                      placeholder={
                        isLoadingConfigs
                          ? 'Loading GenieACS...'
                          : 'Select GenieACS'
                      }
                    />
                  </SelectTrigger>
                  <SelectContent>
                    {configs.map((config) => (
                      <SelectItem key={config.id} value={String(config.id)}>
                        {config.name}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
                <Select
                  value={productClassFilter}
                  onValueChange={setProductClassFilter}
                >
                  <SelectTrigger size='sm' className='h-7 w-full sm:w-40'>
                    <SelectValue placeholder='Filter by product class' />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value='all'>All product classes</SelectItem>
                    {productClasses.map((productClass) => (
                      <SelectItem key={productClass} value={productClass}>
                        {productClass}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
                <Select value={statusFilter} onValueChange={setStatusFilter}>
                  <SelectTrigger size='sm' className='h-7 w-full sm:w-32'>
                    <SelectValue placeholder='Filter by status' />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value='all'>All status</SelectItem>
                    <SelectItem value='online'>Online</SelectItem>
                    <SelectItem value='offline'>Offline</SelectItem>
                    <SelectItem value='stale'>No inform</SelectItem>
                  </SelectContent>
                </Select>
              </>
            }
            actions={(device) => (
              <>
                <Button
                  variant='ghost'
                  size='icon'
                  className='h-7 w-7'
                  title='View device details'
                  aria-label={`View details for ${device.serial_number || device.id}`}
                  onClick={() =>
                    void navigate({
                      to: '/genieacs/devices/$deviceId',
                      params: { deviceId: device.id },
                      search: { configId: Number(selectedId) },
                    })
                  }
                >
                  <Eye className='size-3.5' />
                </Button>
                <Button
                  variant='ghost'
                  size='icon'
                  className='h-7 w-7'
                  title='Summon device'
                  aria-label={`Summon ${device.serial_number || device.id}`}
                  disabled={summoningDeviceId !== null}
                  onClick={() => void summonDevice(device)}
                >
                  <RefreshCw className='size-3.5' />
                </Button>
                <Button
                  variant='ghost'
                  size='icon'
                  className='h-7 w-7 text-destructive hover:text-destructive'
                  title='Delete device'
                  aria-label={`Delete ${device.serial_number || device.id}`}
                  onClick={() => setDeviceToDelete(device)}
                >
                  <Trash2 className='size-3.5' />
                </Button>
              </>
            )}
          />
        </ThemeProvider>
        <ConfirmDialog
          open={deviceToDelete !== null}
          onOpenChange={(open) =>
            !open && !isDeleting && setDeviceToDelete(null)
          }
          title='Delete device?'
          desc={`This removes ${deviceToDelete?.serial_number || deviceToDelete?.id} from GenieACS. The device will register again on its next inform.`}
          confirmText='Delete'
          destructive
          isLoading={isDeleting}
          handleConfirm={() => void confirmDeleteDevice()}
        />
        <ConfirmDialog
          open={devicesToDelete.length > 0}
          onOpenChange={(open) =>
            !open && !isDeleting && setDevicesToDelete([])
          }
          title={`Delete ${devicesToDelete.length} devices?`}
          desc='The selected devices will be removed from GenieACS. Each device will register again on its next inform.'
          confirmText='Delete'
          destructive
          isLoading={isDeleting}
          handleConfirm={() => void confirmBulkDelete()}
        />
      </Main>
    </>
  )
}
