import { useAuthStore } from '@/stores/auth-store'
import { ReadOnlyField } from '../components/read-only-field'

export function ProfileForm() {
  const user = useAuthStore((state) => state.auth.user)

  if (!user) {
    return <p className='text-sm text-muted-foreground'>Loading profile...</p>
  }

  return (
    <div className='grid gap-4 md:grid-cols-2'>
      <ReadOnlyField id='profile-name' label='Name' value={user.name} />
      <ReadOnlyField id='profile-phone' label='Phone' value={user.phone} />
      <ReadOnlyField id='profile-email' label='Email' value={user.email} />
    </div>
  )
}
