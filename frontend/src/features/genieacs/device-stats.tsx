import { useMemo } from 'react'
import { Clock, Server, Wifi, WifiOff } from 'lucide-react'
import { StatCard } from '@/components/stat-card'
import { isStaleDevice } from './device-freshness'
import type { GenieACSDevice } from './genieacs-api'

function percent(part: number, total: number) {
  return total === 0 ? '0%' : `${Math.round((part / total) * 100)}%`
}

type DeviceStatsProps = {
  devices: GenieACSDevice[]
  now: number
  loading: boolean
  statusFilter: string
  onStatusFilterChange: (status: string) => void
}

export function DeviceStats({
  devices,
  now,
  loading,
  statusFilter,
  onStatusFilterChange,
}: DeviceStatsProps) {
  const stats = useMemo(() => {
    const online = devices.filter((device) => device.status === 'online').length
    return {
      total: devices.length,
      online,
      offline: devices.length - online,
      stale: devices.filter((device) => isStaleDevice(device, now)).length,
    }
  }, [devices, now])

  // Clicking the active card clears the filter.
  const toggleStatus = (status: string) =>
    onStatusFilterChange(statusFilter === status ? 'all' : status)

  return (
    <div className='mb-3 grid grid-cols-2 gap-2 sm:gap-3 lg:grid-cols-4'>
      <StatCard
        title='Total Devices'
        value={stats.total}
        description='All devices'
        icon={Server}
        tone='blue'
        loading={loading}
        active={statusFilter === 'all'}
        onClick={() => onStatusFilterChange('all')}
      />
      <StatCard
        title='Online'
        value={stats.online}
        description={`${percent(stats.online, stats.total)} of total`}
        icon={Wifi}
        tone='green'
        loading={loading}
        active={statusFilter === 'online'}
        onClick={() => toggleStatus('online')}
      />
      <StatCard
        title='Offline'
        value={stats.offline}
        description={`${percent(stats.offline, stats.total)} of total`}
        icon={WifiOff}
        tone='red'
        loading={loading}
        active={statusFilter === 'offline'}
        onClick={() => toggleStatus('offline')}
      />
      <StatCard
        title='No Inform'
        value={stats.stale}
        description='Needs attention'
        icon={Clock}
        tone='violet'
        loading={loading}
        active={statusFilter === 'stale'}
        onClick={() => toggleStatus('stale')}
      />
    </div>
  )
}
