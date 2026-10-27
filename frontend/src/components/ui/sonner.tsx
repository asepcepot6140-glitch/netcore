import { Toaster as Sonner, ToasterProps } from 'sonner'
import { useTheme } from '@/context/theme-provider'

// Background transparan; teks dan border mengikuti warna tipe pesan.
const tone = (color: string) => ({
  bg: 'transparent',
  border: `var(${color})`,
  text: `var(${color})`,
})

const tones = {
  success: tone('--success'),
  error: tone('--destructive'),
  warning: tone('--warning'),
  info: tone('--info'),
}

export function Toaster({ ...props }: ToasterProps) {
  const { theme = 'system' } = useTheme()

  const typeVars = Object.fromEntries(
    Object.entries(tones).flatMap(([type, { bg, border, text }]) => [
      [`--${type}-bg`, bg],
      [`--${type}-border`, border],
      [`--${type}-text`, text],
    ])
  )

  return (
    <Sonner
      theme={theme as ToasterProps['theme']}
      richColors
      className='toaster group [&_div[data-content]]:w-full'
      toastOptions={{
        // Blur menjaga teks tetap terbaca saat toast menimpa isi halaman.
        className: 'backdrop-blur-md',
      }}
      style={
        {
          '--normal-bg': 'transparent',
          '--normal-text': 'var(--popover-foreground)',
          '--normal-border': 'var(--border)',
          ...typeVars,
        } as React.CSSProperties
      }
      {...props}
    />
  )
}
