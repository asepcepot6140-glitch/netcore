import { useState } from 'react'
import { z } from 'zod'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { Link, useNavigate } from '@tanstack/react-router'
import { Loader2, LogIn } from 'lucide-react'
import { toast } from 'sonner'
import { useAuthStore } from '@/stores/auth-store'
import { getAuthErrorMessage, loginUser } from '@/lib/auth-api'
import { cn } from '@/lib/utils'
import { Button } from '@/components/ui/button'
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

const formSchema = z.object({
  identifier: z.string().trim().min(1, 'Enter your email or WhatsApp number.'),
  // Tanpa batas minimum: aturan panjang hanya untuk membuat password, bukan login.
  password: z.string().min(1, 'Please enter your password.'),
})

type UserAuthFormProps = React.HTMLAttributes<HTMLFormElement>

export function UserAuthForm({ className, ...props }: UserAuthFormProps) {
  const [isLoading, setIsLoading] = useState(false)
  const navigate = useNavigate()
  const { auth } = useAuthStore()

  const form = useForm<z.infer<typeof formSchema>>({
    resolver: zodResolver(formSchema),
    defaultValues: {
      identifier: '',
      password: '',
    },
  })

  async function onSubmit(data: z.infer<typeof formSchema>) {
    setIsLoading(true)

    try {
      const result = await loginUser(data.identifier, data.password)
      auth.setUser(result.data.user)
      auth.setAccessToken(result.data.access_token)
      auth.setRefreshToken(result.data.refresh_token)
      toast.success(result.message)
      await navigate({ to: '/', replace: true })
    } catch (error) {
      toast.error(getAuthErrorMessage(error))
    } finally {
      setIsLoading(false)
    }
  }

  return (
    <Form {...form}>
      <form
        onSubmit={form.handleSubmit(onSubmit)}
        className={cn('grid gap-3', className)}
        {...props}
      >
        <FormField
          control={form.control}
          name='identifier'
          render={({ field }) => (
            <FormItem>
              <FormLabel>Email or WhatsApp number</FormLabel>
              <FormControl>
                <Input
                  autoComplete='username'
                  autoFocus
                  className='dark:bg-transparent'
                  placeholder='name@example.com or 08123456789'
                  {...field}
                />
              </FormControl>
              <FormMessage />
            </FormItem>
          )}
        />
        <FormField
          control={form.control}
          name='password'
          render={({ field }) => (
            <FormItem className='relative'>
              <FormLabel>Password</FormLabel>
              <FormControl>
                <PasswordInput
                  autoComplete='current-password'
                  placeholder='********'
                  {...field}
                />
              </FormControl>
              <FormMessage />
              <Link
                to='/forgot-password'
                className='absolute inset-e-0 -top-0.5 text-sm font-medium text-muted-foreground hover:opacity-75'
              >
                Forgot password?
              </Link>
            </FormItem>
          )}
        />
        <Button className='mt-2' disabled={isLoading}>
          {isLoading ? <Loader2 className='animate-spin' /> : <LogIn />}
          Sign in
        </Button>
        <p className='text-center text-sm text-muted-foreground'>
          Don't have an account?{' '}
          <Link
            to='/sign-up'
            className='text-nowrap underline underline-offset-4 hover:text-primary'
          >
            Sign Up
          </Link>
        </p>
      </form>
    </Form>
  )
}
