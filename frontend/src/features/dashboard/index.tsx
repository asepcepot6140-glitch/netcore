import { useQuery } from '@tanstack/react-query'
import { Activity, Network, Server, WifiOff } from 'lucide-react'
import { cn } from '@/lib/utils'
import { Badge } from '@/components/ui/badge'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { Header } from '@/components/layout/header'
import { Main } from '@/components/layout/main'
import { ProfileDropdown } from '@/components/profile-dropdown'
import { StatCard } from '@/components/stat-card'
import { ThemeSwitch } from '@/components/theme-switch'
import {
  getGenieACSConfigs,
  getGenieACSDevices,
  stripHostScheme,
  type GenieACSConfig,
} from '../genieacs/genieacs-api'
import {
  getPppoeActive,
  getPppoeErrorMessage,
  getPppoeSecrets,
} from '../mikrotik/pppoe/pppoe-api'
import {
  getMikrotikRouters,
  type MikrotikRouter,
} from '../mikrotik/router/mikrotik-api'

type RouterStats = {
  router: MikrotikRouter
  total: number
  active: number
  offline: number
  disabled: number
  error?: string
}

type AcsStats = {
  config: GenieACSConfig
  total: number
  online: number
  offline: number
  error?: string
}

type Overview = {
  routers: RouterStats[]
  acs: AcsStats[]
  routersError?: string
  acsError?: string
}

async function loadRouterStats(router: MikrotikRouter): Promise<RouterStats> {
  try {
    const [secrets, active] = await Promise.all([
      getPppoeSecrets(router.id),
      getPppoeActive(router.id),
    ])
    const activeNames = new Set(active.map((item) => item.name))
    const offline = secrets.filter((item) => !activeNames.has(item.name))

    return {
      router,
      total: secrets.length,
      active: active.length,
      offline: offline.length,
      disabled: secrets.filter((item) => item.disabled).length,
    }
  } catch (error) {
    return {
      router,
      total: 0,
      active: 0,
      offline: 0,
      disabled: 0,
      error: getPppoeErrorMessage(error),
    }
  }
}

async function loadAcsStats(config: GenieACSConfig): Promise<AcsStats> {
  try {
    const { data } = await getGenieACSDevices(config.id)
    const online = data.filter((device) => device.status === 'online').length

    return {
      config,
      total: data.length,
      online,
      offline: data.length - online,
    }
  } catch (error) {
    return {
      config,
      total: 0,
      online: 0,
      offline: 0,
      error: getPppoeErrorMessage(error),
    }
  }
}

// One unreachable router/ACS must not fail the whole dashboard.
async function loadOverview(): Promise<Overview> {
  const [routersResult, configsResult] = await Promise.allSettled([
    getMikrotikRouters(),
    getGenieACSConfigs(),
  ])

  const [routers, acs] = await Promise.all([
    routersResult.status === 'fulfilled'
      ? Promise.all(routersResult.value.map(loadRouterStats))
      : [],
    configsResult.status === 'fulfilled'
      ? Promise.all(configsResult.value.map(loadAcsStats))
      : [],
  ])

  return {
    routers,
    acs,
    routersError:
      routersResult.status === 'rejected'
        ? getPppoeErrorMessage(routersResult.reason)
        : undefined,
    acsError:
      configsResult.status === 'rejected'
        ? getPppoeErrorMessage(configsResult.reason)
        : undefined,
  }
}

function sum<T>(items: T[], pick: (item: T) => number) {
  return items.reduce((total, item) => total + pick(item), 0)
}

function ErrorBadge({ message }: { message: string }) {
  return (
    <Badge variant='destructive' className='max-w-64 truncate' title={message}>
      {message}
    </Badge>
  )
}

const DASHBOARD_POLL_INTERVAL_MS = 5_000

export function Dashboard() {
  // Realtime polling without cache: data is dropped as soon as the page unmounts.
  const { data, isLoading, isError, error } = useQuery({
    queryKey: ['dashboard', 'overview'],
    queryFn: loadOverview,
    refetchInterval: DASHBOARD_POLL_INTERVAL_MS,
    gcTime: 0,
  })

  const routers = data?.routers ?? []
  const acs = data?.acs ?? []
  const okRouters = routers.filter((item) => !item.error)
  const okAcs = acs.filter((item) => !item.error)
  const unreachable =
    routers.filter((item) => item.error).length +
    acs.filter((item) => item.error).length

  return (
    <>
      <Header>
        <div className='ms-auto flex items-center gap-2'>
          <ThemeSwitch />
          <ProfileDropdown />
        </div>
      </Header>

      <Main fluid className='space-y-3'>
        <div className='flex items-center justify-between'>
          <h1 className='text-2xl font-bold tracking-tight'>Dashboard</h1>
        </div>

        {isError && (
          <Card className='border-destructive'>
            <CardHeader>
              <CardTitle>Failed to load dashboard</CardTitle>
              <CardDescription>{getPppoeErrorMessage(error)}</CardDescription>
            </CardHeader>
          </Card>
        )}

        <div className='grid grid-cols-2 gap-2 sm:gap-3 lg:grid-cols-4'>
          <StatCard
            title='Router Mikrotik'
            icon={Network}
            tone='blue'
            value={routers.length}
            description={
              unreachable > 0
                ? `${unreachable} device(s) unreachable`
                : 'Registered'
            }
            loading={isLoading}
          />
          <StatCard
            title='Active PPPoE'
            icon={Activity}
            tone='green'
            value={sum(okRouters, (item) => item.active)}
            description={`of ${sum(okRouters, (item) => item.total)} accounts`}
            loading={isLoading}
          />
          <StatCard
            title='PPPoE Offline'
            icon={WifiOff}
            tone='red'
            value={sum(okRouters, (item) => item.offline)}
            description={`${sum(okRouters, (item) => item.disabled)} accounts disabled`}
            loading={isLoading}
          />
          <StatCard
            title='GenieACS Devices'
            icon={Server}
            tone='violet'
            value={`${sum(okAcs, (item) => item.online)} / ${sum(okAcs, (item) => item.total)}`}
            description='Online / total'
            loading={isLoading}
          />
        </div>

        <div className='grid grid-cols-1 gap-4 lg:grid-cols-2'>
          <Card className='gap-2'>
            <CardHeader className='pb-0'>
              <CardTitle>Router Summary</CardTitle>
              <CardDescription>
                {data?.routersError ?? 'PPPoE accounts per router.'}
              </CardDescription>
            </CardHeader>
            <CardContent>
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Router</TableHead>
                    <TableHead className='text-end'>Total</TableHead>
                    <TableHead className='text-end'>Active</TableHead>
                    <TableHead className='text-end'>Offline</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {isLoading && (
                    <TableRow>
                      <TableCell colSpan={4}>
                        <Skeleton className='h-5 w-full' />
                      </TableCell>
                    </TableRow>
                  )}
                  {!isLoading && routers.length === 0 && (
                    <TableRow>
                      <TableCell
                        colSpan={4}
                        className='text-center text-muted-foreground'
                      >
                        No routers yet.
                      </TableCell>
                    </TableRow>
                  )}
                  {routers.map((item) => (
                    <TableRow key={item.router.id}>
                      <TableCell>
                        <div className='font-medium'>{item.router.name}</div>
                        <div className='text-xs text-muted-foreground'>
                          {item.router.host}:{item.router.port}
                        </div>
                      </TableCell>
                      {item.error ? (
                        <TableCell colSpan={3} className='text-end'>
                          <ErrorBadge message={item.error} />
                        </TableCell>
                      ) : (
                        <>
                          <TableCell className='text-end'>
                            {item.total}
                          </TableCell>
                          <TableCell className='text-end font-medium text-emerald-600 dark:text-emerald-400'>
                            {item.active}
                          </TableCell>
                          <TableCell
                            className={cn(
                              'text-end font-medium',
                              item.offline > 0 &&
                                'text-red-600 dark:text-red-400'
                            )}
                          >
                            {item.offline}
                          </TableCell>
                        </>
                      )}
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </CardContent>
          </Card>

          <Card className='gap-2'>
            <CardHeader className='pb-0'>
              <CardTitle>GenieACS Summary</CardTitle>
              <CardDescription>
                {data?.acsError ?? 'Device status per ACS server.'}
              </CardDescription>
            </CardHeader>
            <CardContent>
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Server</TableHead>
                    <TableHead className='text-end'>Total</TableHead>
                    <TableHead className='text-end'>Online</TableHead>
                    <TableHead className='text-end'>Offline</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {isLoading && (
                    <TableRow>
                      <TableCell colSpan={4}>
                        <Skeleton className='h-5 w-full' />
                      </TableCell>
                    </TableRow>
                  )}
                  {!isLoading && acs.length === 0 && (
                    <TableRow>
                      <TableCell
                        colSpan={4}
                        className='text-center text-muted-foreground'
                      >
                        No GenieACS servers yet.
                      </TableCell>
                    </TableRow>
                  )}
                  {acs.map((item) => (
                    <TableRow key={item.config.id}>
                      <TableCell>
                        <div className='font-medium'>{item.config.name}</div>
                        <div className='text-xs text-muted-foreground'>
                          {stripHostScheme(item.config.host)}:{item.config.port}
                        </div>
                      </TableCell>
                      {item.error ? (
                        <TableCell colSpan={3} className='text-end'>
                          <ErrorBadge message={item.error} />
                        </TableCell>
                      ) : (
                        <>
                          <TableCell className='text-end'>
                            {item.total}
                          </TableCell>
                          <TableCell className='text-end font-medium text-emerald-600 dark:text-emerald-400'>
                            {item.online}
                          </TableCell>
                          <TableCell
                            className={cn(
                              'text-end font-medium',
                              item.offline > 0 &&
                                'text-red-600 dark:text-red-400'
                            )}
                          >
                            {item.offline}
                          </TableCell>
                        </>
                      )}
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </CardContent>
          </Card>
        </div>
      </Main>
    </>
  )
}
