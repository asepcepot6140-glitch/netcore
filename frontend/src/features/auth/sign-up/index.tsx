import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { AuthLayout } from '../auth-layout'
import { SignUpForm } from './components/sign-up-form'

export function SignUp() {
  return (
    <AuthLayout>
      <Card className='max-w-sm gap-4 sm:min-w-sm'>
        <CardHeader>
          <CardTitle className='text-lg tracking-tight'>
            Create an account
          </CardTitle>
          <CardDescription>
            Enter your name, email, WhatsApp number, and password to create an
            account.
          </CardDescription>
        </CardHeader>
        <CardContent>
          <SignUpForm />
        </CardContent>
      </Card>
    </AuthLayout>
  )
}
