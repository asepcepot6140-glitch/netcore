import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'

type ReadOnlyFieldProps = {
  id: string
  label: string
  value: string | number
}

export function ReadOnlyField({ id, label, value }: ReadOnlyFieldProps) {
  return (
    <div className='space-y-2'>
      <Label htmlFor={id}>{label}</Label>
      <Input id={id} value={value} readOnly className='bg-muted/50' />
    </div>
  )
}
