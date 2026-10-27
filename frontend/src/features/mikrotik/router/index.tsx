import { useCallback, useEffect, useMemo, useState } from 'react'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { ThemeProvider, createTheme } from '@mui/material'
import { Pencil, Trash2 } from 'lucide-react'
import { toast } from 'sonner'
import { getAuthErrorMessage } from '@/lib/auth-api'
import { Button } from '@/components/ui/button'
import { MuiDataTable, type DataTableColumn } from '@/components/data-table'
import { Header } from '@/components/layout/header'
import { Main } from '@/components/layout/main'
import { ProfileDropdown } from '@/components/profile-dropdown'
import { ThemeSwitch } from '@/components/theme-switch'
import { DeleteRouterDialog, RouterDialog } from './components/dialog'
import {
  defaultValues,
  routerSchema,
  type RouterFormValues,
} from './components/dialog-schema'
import {
  createMikrotikRouter,
  deleteMikrotikRouter,
  getMikrotikRouters,
  type MikrotikRouter,
  updateMikrotikRouter,
} from './mikrotik-api'

function useIsDark() {
  const [dark, setDark] = useState(() =>
    document.documentElement.classList.contains('dark')
  )

  useEffect(() => {
    const element = document.documentElement
    const observer = new MutationObserver(() =>
      setDark(element.classList.contains('dark'))
    )
    observer.observe(element, { attributes: true, attributeFilter: ['class'] })
    return () => observer.disconnect()
  }, [])

  return dark
}

function useMuiTheme() {
  const dark = useIsDark()
  return useMemo(
    () =>
      createTheme({
        palette: { mode: dark ? 'dark' : 'light' },
        typography: { fontFamily: 'inherit' },
      }),
    [dark]
  )
}

const routerColumns: DataTableColumn<MikrotikRouter>[] = [
  {
    header: 'Name',
    cell: (router) => <span className='font-semibold'>{router.name}</span>,
  },
  { header: 'IP Address', cell: (router) => router.host },
  { header: 'Username', cell: (router) => router.username },
  { header: 'Port', cell: (router) => router.port, align: 'right' },
]

export function MikrotikRouters() {
  const muiTheme = useMuiTheme()
  const [routers, setRouters] = useState<MikrotikRouter[]>([])
  const [isLoadingRouters, setIsLoadingRouters] = useState(true)
  const [isSaving, setIsSaving] = useState(false)
  const [isDialogOpen, setIsDialogOpen] = useState(false)
  const [editingRouter, setEditingRouter] = useState<MikrotikRouter | null>(
    null
  )
  const [deletingRouter, setDeletingRouter] = useState<MikrotikRouter | null>(
    null
  )
  const form = useForm<RouterFormValues>({
    resolver: zodResolver(routerSchema),
    defaultValues,
  })

  const loadRouters = useCallback(async () => {
    try {
      setRouters(await getMikrotikRouters())
    } catch (error) {
      toast.error(getAuthErrorMessage(error))
    } finally {
      setIsLoadingRouters(false)
    }
  }, [])

  useEffect(() => {
    const timer = window.setTimeout(() => void loadRouters(), 0)
    return () => window.clearTimeout(timer)
  }, [loadRouters])

  async function onSubmit(values: RouterFormValues) {
    if (!editingRouter && !values.password) {
      form.setError('password', {
        type: 'manual',
        message: 'Password is required.',
      })
      return
    }

    setIsSaving(true)
    try {
      const { password, ...routerFields } = values
      const result = editingRouter
        ? await updateMikrotikRouter(editingRouter.id, {
            ...routerFields,
            port: Number(values.port),
            ...(password ? { password } : {}),
          })
        : await createMikrotikRouter({
            ...routerFields,
            password,
            port: Number(values.port),
          })
      toast.success(result.message)
      form.reset(defaultValues)
      setIsDialogOpen(false)
      setEditingRouter(null)
      setIsLoadingRouters(true)
      await loadRouters()
    } catch (error) {
      toast.error(getAuthErrorMessage(error))
    } finally {
      setIsSaving(false)
    }
  }

  function openAddDialog() {
    setEditingRouter(null)
    form.reset(defaultValues)
    setIsDialogOpen(true)
  }

  function openEditDialog(router: MikrotikRouter) {
    setEditingRouter(router)
    form.reset({
      name: router.name,
      host: router.host,
      username: router.username,
      password: '',
      port: String(router.port),
    })
    setIsDialogOpen(true)
  }

  async function confirmDeleteRouter() {
    if (!deletingRouter) return

    try {
      const result = await deleteMikrotikRouter(deletingRouter.id)
      toast.success(result.message)
      setDeletingRouter(null)
      setIsLoadingRouters(true)
      await loadRouters()
    } catch (error) {
      toast.error(getAuthErrorMessage(error))
    }
  }

  return (
    <>
      <Header>
        <div className='ms-auto flex items-center gap-2'>
          <ThemeSwitch />
          <ProfileDropdown />
        </div>
      </Header>
      <Main fluid>
        <div className='flex flex-wrap items-start justify-between gap-4'>
          <div className='space-y-1'>
            <h1 className='text-2xl font-bold tracking-tight md:text-3xl'>
              MikroTik Routers
            </h1>
          </div>
        </div>

        <ThemeProvider theme={muiTheme}>
          <div className='mt-3'>
            <MuiDataTable
              columns={routerColumns}
              rows={routers}
              getRowId={(router) => router.id}
              isLoading={isLoadingRouters}
              loadingLabel='Loading routers...'
              addLabel='Add MikroTik'
              onAdd={openAddDialog}
              actions={(router) => (
                <>
                  <Button
                    type='button'
                    variant='ghost'
                    size='icon'
                    className='h-7 w-7'
                    aria-label={`Edit ${router.name}`}
                    title='Edit'
                    onClick={() => openEditDialog(router)}
                  >
                    <Pencil className='size-3.5' />
                  </Button>
                  <Button
                    type='button'
                    variant='ghost'
                    size='icon'
                    className='h-7 w-7 text-destructive hover:text-destructive'
                    aria-label={`Delete ${router.name}`}
                    title='Delete'
                    onClick={() => setDeletingRouter(router)}
                  >
                    <Trash2 className='size-3.5' />
                  </Button>
                </>
              )}
            />
          </div>
        </ThemeProvider>
      </Main>

      <RouterDialog
        open={isDialogOpen}
        onOpenChange={setIsDialogOpen}
        editingRouter={editingRouter}
        isSaving={isSaving}
        form={form}
        onSubmit={onSubmit}
      />
      <DeleteRouterDialog
        router={deletingRouter}
        onCancel={() => setDeletingRouter(null)}
        onConfirm={confirmDeleteRouter}
      />
    </>
  )
}
