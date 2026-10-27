import { useState } from 'react'
import { z } from 'zod'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { useNavigate } from '@tanstack/react-router'
import { toast } from 'sonner'
import { resetPassword, getAuthErrorMessage } from '@/lib/auth-api'
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
  otp: z
    .string()
    .min(6, 'Please enter the 6-digit code.')
    .max(6, 'Please enter the 6-digit code.'),
  newPassword: z
    .string()
    .min(1, 'Please enter your new password.')
    .min(8, 'Password must be at least 8 characters long.'),
})

type OtpFormProps = React.HTMLAttributes<HTMLFormElement>

export function OtpForm({ className, ...props }: OtpFormProps) {
  const navigate = useNavigate()
  const [isLoading, setIsLoading] = useState(false)

  const form = useForm<z.infer<typeof formSchema>>({
    resolver: zodResolver(formSchema),
    defaultValues: { otp: '', newPassword: '' },
  })

  // eslint-disable-next-line react-hooks/incompatible-library
  const otp = form.watch('otp')

  async function onSubmit(data: z.infer<typeof formSchema>) {
    const phone = sessionStorage.getItem('reset_phone')
    if (!phone) {
      toast.error(
        'Session expired or phone number not found. Please request password reset again.'
      )
      navigate({ to: '/forgot-password' })
      return
    }

    setIsLoading(true)
    try {
      const res = await resetPassword(phone, data.otp, data.newPassword)
      toast.success(res.message || 'Password has been reset successfully.')
      sessionStorage.removeItem('reset_phone')
      navigate({ to: '/sign-in' })
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
          name='otp'
          render={({ field }) => (
            <FormItem>
              <FormLabel>One-Time Password (OTP)</FormLabel>
              <FormControl>
                <Input
                  inputMode='numeric'
                  autoComplete='one-time-code'
                  maxLength={6}
                  placeholder='123456'
                  className='text-center font-mono text-lg tracking-[0.5em]'
                  {...field}
                  onChange={(e) =>
                    field.onChange(
                      e.target.value.replace(/\D/g, '').slice(0, 6)
                    )
                  }
                />
              </FormControl>
              <FormMessage />
            </FormItem>
          )}
        />
        <FormField
          control={form.control}
          name='newPassword'
          render={({ field }) => (
            <FormItem>
              <FormLabel>New Password</FormLabel>
              <FormControl>
                <PasswordInput placeholder='Enter new password' {...field} />
              </FormControl>
              <FormMessage />
            </FormItem>
          )}
        />
        <Button className='mt-2' disabled={otp.length < 6 || isLoading}>
          Reset Password
        </Button>
      </form>
    </Form>
  )
}
