import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { AuthLayout } from '../auth-layout'
import { UserAuthForm } from './components/user-auth-form'

export function SignIn() {
  return (
    <AuthLayout>
      <Card className='max-w-sm gap-4 sm:min-w-sm'>
        <CardHeader>
          <CardTitle className='text-lg tracking-tight'>Sign in</CardTitle>
          <CardDescription>
            Enter your email and password below to log into your account.
          </CardDescription>
        </CardHeader>
        <CardContent>
          <UserAuthForm />
        </CardContent>
      </Card>
    </AuthLayout>
  )
}
