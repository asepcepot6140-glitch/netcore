import type { SubmitHandler, UseFormReturn } from 'react-hook-form'
import { Loader2 } from 'lucide-react'
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
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import { PasswordInput } from '@/components/password-input'
import type { MikrotikRouter } from '../mikrotik-api'
import type { RouterFormValues } from './dialog-schema'

type RouterDialogProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
  editingRouter: MikrotikRouter | null
  isSaving: boolean
  form: UseFormReturn<RouterFormValues>
  onSubmit: SubmitHandler<RouterFormValues>
}

export function RouterDialog({
  open,
  onOpenChange,
  editingRouter,
  isSaving,
  form,
  onSubmit,
}: RouterDialogProps) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className='p-0 sm:max-w-md' showCloseButton={false}>
        <DialogHeader className='border-b px-6 py-3'>
          <DialogTitle>
            {editingRouter ? 'Edit MikroTik' : 'Add MikroTik'}
          </DialogTitle>
          <DialogDescription className='sr-only'>
            {editingRouter
              ? 'Edit MikroTik router form.'
              : 'Add MikroTik router form.'}
          </DialogDescription>
        </DialogHeader>
        <Form {...form}>
          <form
            onSubmit={form.handleSubmit(onSubmit)}
            className='space-y-4 px-6 pt-2.5 pb-6'
            autoComplete='off'
          >
            <div className='flex flex-col gap-4'>
              <FormField
                control={form.control}
                name='name'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>Name</FormLabel>
                    <FormControl>
                      <Input
                        autoComplete='off'
                        placeholder='Office router'
                        className='bg-transparent dark:bg-transparent'
                        {...field}
                      />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
              <FormField
                control={form.control}
                name='host'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>IP address</FormLabel>
                    <FormControl>
                      <Input
                        autoComplete='off'
                        placeholder='192.168.88.1'
                        className='bg-transparent dark:bg-transparent'
                        {...field}
                      />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
              <FormField
                control={form.control}
                name='username'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>Username</FormLabel>
                    <FormControl>
                      <Input
                        autoComplete='off'
                        placeholder='admin'
                        className='bg-transparent dark:bg-transparent'
                        {...field}
                      />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
              <div className='grid gap-4'>
                <FormField
                  control={form.control}
                  name='password'
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>
                        Password{editingRouter ? ' (optional)' : ''}
                      </FormLabel>
                      <FormControl>
                        <PasswordInput
                          autoComplete='new-password'
                          placeholder={
                            editingRouter
                              ? 'Leave blank to keep unchanged'
                              : 'Router password'
                          }
                          {...field}
                        />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />
                <FormField
                  control={form.control}
                  name='port'
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>Port API</FormLabel>
                      <FormControl>
                        <Input
                          autoComplete='off'
                          inputMode='numeric'
                          placeholder='8728'
                          className='bg-transparent dark:bg-transparent'
                          {...field}
                        />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />
              </div>
            </div>
            <div className='flex flex-col-reverse gap-2 pt-2 sm:flex-row sm:justify-end'>
              <Button
                type='button'
                variant='outline'
                onClick={() => onOpenChange(false)}
              >
                Cancel
              </Button>
              <Button type='submit' disabled={isSaving}>
                {isSaving && <Loader2 className='animate-spin' />}
                {editingRouter ? 'Save Changes' : 'Add MikroTik'}
              </Button>
            </div>
          </form>
        </Form>
      </DialogContent>
    </Dialog>
  )
}

type DeleteRouterDialogProps = {
  router: MikrotikRouter | null
  onCancel: () => void
  onConfirm: () => void
}

export function DeleteRouterDialog({
  router,
  onCancel,
  onConfirm,
}: DeleteRouterDialogProps) {
  return (
    <Dialog open={router !== null} onOpenChange={(open) => !open && onCancel()}>
      <DialogContent className='sm:max-w-md' showCloseButton={false}>
        <DialogHeader>
          <DialogTitle>Delete router?</DialogTitle>
          <DialogDescription>
            Router &quot;{router?.name}&quot; will be deleted. This action
            cannot be undone.
          </DialogDescription>
        </DialogHeader>
        <DialogFooter>
          <Button variant='outline' onClick={onCancel}>
            Cancel
          </Button>
          <Button variant='destructive' onClick={onConfirm}>
            Delete
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
