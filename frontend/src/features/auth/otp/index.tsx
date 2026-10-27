import { Link } from '@tanstack/react-router'
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { AuthLayout } from '../auth-layout'
import { OtpForm } from './components/otp-form'

export function Otp() {
  return (
    <AuthLayout>
      <Card className='max-w-sm gap-4 sm:min-w-sm'>
        <CardHeader>
          <CardTitle className='text-base tracking-tight'>
            Reset Password OTP
          </CardTitle>
          <CardDescription>
            Please enter the 6-digit OTP sent to your WhatsApp <br /> and your
            new password.
          </CardDescription>
        </CardHeader>
        <CardContent>
          <OtpForm />
        </CardContent>
        <CardFooter>
          <p className='px-8 text-center text-sm text-muted-foreground'>
            Haven't received it? Check your number or{' '}
            <Link
              to='/forgot-password'
              className='underline underline-offset-4 hover:text-primary'
            >
              request a new code
            </Link>
            .
          </p>
        </CardFooter>
      </Card>
    </AuthLayout>
  )
}
