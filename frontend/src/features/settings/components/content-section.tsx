import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'

type ContentSectionProps = {
  title: string
  desc: string
  children: React.JSX.Element
}

export function ContentSection({ title, desc, children }: ContentSectionProps) {
  return (
    <Card className='gap-4 py-4'>
      <CardHeader className='border-b [.border-b]:pb-3'>
        <CardTitle className='text-lg'>{title}</CardTitle>
        <CardDescription>{desc}</CardDescription>
      </CardHeader>
      <CardContent>{children}</CardContent>
    </Card>
  )
}
