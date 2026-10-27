import { useEffect, useMemo, useState } from 'react'
import { createTheme } from '@mui/material'

export function useMuiTheme() {
  const [isDark, setIsDark] = useState(() =>
    document.documentElement.classList.contains('dark')
  )

  useEffect(() => {
    const element = document.documentElement
    const observer = new MutationObserver(() =>
      setIsDark(element.classList.contains('dark'))
    )
    observer.observe(element, { attributes: true, attributeFilter: ['class'] })
    return () => observer.disconnect()
  }, [])

  return useMemo(
    () =>
      createTheme({
        palette: { mode: isDark ? 'dark' : 'light' },
        typography: { fontFamily: 'inherit' },
      }),
    [isDark]
  )
}
