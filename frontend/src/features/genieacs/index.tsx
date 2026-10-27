import { useCallback, useEffect, useState } from 'react'
import { ThemeProvider } from '@mui/material'
import { Pencil, Trash2 } from 'lucide-react'
import { toast } from 'sonner'
import { getAuthErrorMessage } from '@/lib/auth-api'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { MuiDataTable, type DataTableColumn } from '@/components/data-table'
import { Header } from '@/components/layout/header'
import { Main } from '@/components/layout/main'
import { ProfileDropdown } from '@/components/profile-dropdown'
import { ThemeSwitch } from '@/components/theme-switch'
import {
  createGenieACSConfig,
  deleteGenieACSConfig,
  getGenieACSConfigs,
  type GenieACSConfig,
  type GenieACSInput,
  stripHostScheme,
  updateGenieACSConfig,
} from './genieacs-api'
import { useMuiTheme } from './use-mui-theme'

const emptyForm: GenieACSInput = { name: '', host: '', port: 7557 }

const columns: DataTableColumn<GenieACSConfig>[] = [
  {
    header: 'Name',
    cell: (config) => <span className='font-semibold'>{config.name}</span>,
    getSortValue: (config) => config.name,
  },
  { header: 'Host', cell: (config) => stripHostScheme(config.host) },
  { header: 'Port', cell: (config) => config.port, align: 'right' },
]

export function GenieACSIndex() {
  const muiTheme = useMuiTheme()
  const [configs, setConfigs] = useState<GenieACSConfig[]>([])
  const [isLoading, setIsLoading] = useState(true)
  const [isSaving, setIsSaving] = useState(false)
  const [dialogOpen, setDialogOpen] = useState(false)
  const [editingConfig, setEditingConfig] = useState<GenieACSConfig | null>(
    null
  )
  const [deletingConfig, setDeletingConfig] = useState<GenieACSConfig | null>(
    null
  )
  const [form, setForm] = useState<GenieACSInput>(emptyForm)

  const loadConfigs = useCallback(async () => {
    try {
      setConfigs(await getGenieACSConfigs())
    } catch (error) {
      toast.error(getAuthErrorMessage(error))
    } finally {
      setIsLoading(false)
    }
  }, [])

  useEffect(() => {
    const timer = window.setTimeout(() => void loadConfigs(), 0)
    return () => window.clearTimeout(timer)
  }, [loadConfigs])

  function openAddDialog() {
    setEditingConfig(null)
    setForm(emptyForm)
    setDialogOpen(true)
  }

  function openEditDialog(config: GenieACSConfig) {
    setEditingConfig(config)
    setForm({
      name: config.name,
      host: stripHostScheme(config.host),
      port: config.port,
    })
    setDialogOpen(true)
  }

  async function submitConfig(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setIsSaving(true)
    try {
      const payload: GenieACSInput = {
        ...form,
        host: stripHostScheme(form.host),
      }
      const result = editingConfig
        ? await updateGenieACSConfig(editingConfig.id, payload)
        : await createGenieACSConfig(payload)
      toast.success(result.message)
      setDialogOpen(false)
      setIsLoading(true)
      await loadConfigs()
    } catch (error) {
      toast.error(getAuthErrorMessage(error))
    } finally {
      setIsSaving(false)
    }
  }

  async function confirmDelete() {
    if (!deletingConfig) return
    try {
      const result = await deleteGenieACSConfig(deletingConfig.id)
      toast.success(result.message)
      setDeletingConfig(null)
      setIsLoading(true)
      await loadConfigs()
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
        <div className='mb-3 space-y-1'>
          <h1 className='text-2xl font-bold tracking-tight md:text-3xl'>
            GenieACS
          </h1>
        </div>
        <ThemeProvider theme={muiTheme}>
          <MuiDataTable
            columns={columns}
            rows={configs}
            getRowId={(config) => config.id}
            isLoading={isLoading}
            loadingLabel='Loading GenieACS configurations...'
            addLabel='Add GenieACS'
            onAdd={openAddDialog}
            actions={(config) => (
              <>
                <Button
                  variant='ghost'
                  size='icon'
                  className='h-7 w-7'
                  aria-label={`Edit ${config.name}`}
                  title='Edit'
                  onClick={() => openEditDialog(config)}
                >
                  <Pencil className='size-3.5' />
                </Button>
                <Button
                  variant='ghost'
                  size='icon'
                  className='h-7 w-7 text-destructive hover:text-destructive'
                  aria-label={`Delete ${config.name}`}
                  title='Delete'
                  onClick={() => setDeletingConfig(config)}
                >
                  <Trash2 className='size-3.5' />
                </Button>
              </>
            )}
          />
        </ThemeProvider>
      </Main>

      <Dialog open={dialogOpen} onOpenChange={setDialogOpen}>
        <DialogContent className='p-0 sm:max-w-md' showCloseButton={false}>
          <DialogHeader className='border-b px-6 py-3'>
            <DialogTitle>
              {editingConfig ? 'Edit GenieACS' : 'Add GenieACS'}
            </DialogTitle>
            <DialogDescription className='sr-only'>
              GenieACS connection settings form.
            </DialogDescription>
          </DialogHeader>
          <form
            onSubmit={submitConfig}
            className='space-y-4 px-6 pt-2.5 pb-6'
            autoComplete='off'
          >
            <div className='space-y-2'>
              <Label htmlFor='genieacs-name'>Name</Label>
              <Input
                id='genieacs-name'
                value={form.name}
                onChange={(event) =>
                  setForm((current) => ({
                    ...current,
                    name: event.target.value,
                  }))
                }
                minLength={2}
                maxLength={100}
                required
                className='bg-transparent dark:bg-transparent'
              />
            </div>
            <div className='space-y-2'>
              <Label htmlFor='genieacs-host'>Host</Label>
              <Input
                id='genieacs-host'
                value={form.host}
                onChange={(event) =>
                  setForm((current) => ({
                    ...current,
                    host: event.target.value,
                  }))
                }
                placeholder='192.168.1.10'
                maxLength={255}
                required
                className='bg-transparent dark:bg-transparent'
              />
            </div>
            <div className='space-y-2'>
              <Label htmlFor='genieacs-port'>Port</Label>
              <Input
                id='genieacs-port'
                type='number'
                inputMode='numeric'
                min={1}
                max={65535}
                value={form.port}
                onChange={(event) =>
                  setForm((current) => ({
                    ...current,
                    port: Number(event.target.value),
                  }))
                }
                required
                className='bg-transparent dark:bg-transparent'
              />
            </div>
            <DialogFooter className='pt-2'>
              <Button
                type='button'
                variant='outline'
                onClick={() => setDialogOpen(false)}
              >
                Cancel
              </Button>
              <Button type='submit' disabled={isSaving}>
                {isSaving
                  ? 'Saving...'
                  : editingConfig
                    ? 'Save Changes'
                    : 'Add GenieACS'}
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>

      <Dialog
        open={deletingConfig !== null}
        onOpenChange={(open) => !open && setDeletingConfig(null)}
      >
        <DialogContent className='sm:max-w-md' showCloseButton={false}>
          <DialogHeader>
            <DialogTitle>Delete GenieACS configuration?</DialogTitle>
            <DialogDescription>
              &quot;{deletingConfig?.name}&quot; will be deleted. This action
              cannot be undone.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant='outline' onClick={() => setDeletingConfig(null)}>
              Cancel
            </Button>
            <Button variant='destructive' onClick={() => void confirmDelete()}>
              Delete
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  )
}
