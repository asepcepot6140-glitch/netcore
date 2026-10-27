import { Outlet } from '@tanstack/react-router'
import { Header } from '@/components/layout/header'
import { Main } from '@/components/layout/main'
import { ProfileDropdown } from '@/components/profile-dropdown'
import { ThemeSwitch } from '@/components/theme-switch'

export function Settings() {
  return (
    <>
      {/* ===== Top Heading ===== */}
      <Header>
        <div className='ms-auto flex items-center gap-2'>
          <ThemeSwitch />
          <ProfileDropdown />
        </div>
      </Header>

      <Main fluid className='space-y-3'>
        <div>
          <h1 className='text-2xl font-bold tracking-tight'>Settings</h1>
        </div>
        <Outlet />
      </Main>
    </>
  )
}
