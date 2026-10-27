import { useEffect, useMemo, useState } from 'react'
import { createTheme } from '@mui/material'

function useIsDark() {
  const [dark, setDark] = useState(() =>
    document.documentElement.classList.contains('dark')
  )

  useEffect(() => {
    const element = document.documentElement
    const observer = new MutationObserver(() =>
      setDark(element.classList.contains('dark'))
    )
    observer.observe(element, { attributes: true, attributeFilter: ['class'] })
    return () => observer.disconnect()
  }, [])

  return dark
}

export function useMuiTheme() {
  const dark = useIsDark()
  return useMemo(
    () =>
      createTheme({
        palette: { mode: dark ? 'dark' : 'light' },
        typography: { fontFamily: 'inherit' },
      }),
    [dark]
  )
}
