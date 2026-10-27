import { Link } from '@tanstack/react-router'
import { Logo } from '@/assets/logo'
import {
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  useSidebar,
} from '@/components/ui/sidebar'

export function AppTitle() {
  const { setOpenMobile } = useSidebar()
  return (
    <SidebarMenu>
      <SidebarMenuItem>
        <SidebarMenuButton
          size='lg'
          className='gap-0 py-0 hover:bg-transparent active:bg-transparent'
          asChild
        >
          <div>
            <Link
              to='/'
              onClick={() => setOpenMobile(false)}
              className='flex flex-1 items-center gap-2.5 text-start'
            >
              <div className='flex size-8 shrink-0 items-center justify-center rounded-lg bg-primary text-primary-foreground shadow-sm'>
                <Logo className='size-5' />
              </div>
              <div className='grid flex-1 text-sm leading-tight group-data-[collapsible=icon]:hidden'>
                <span className='truncate text-base font-semibold tracking-tight'>
                  APK NETWORKS
                </span>
                <span className='truncate text-[11px] font-medium tracking-wide text-muted-foreground uppercase'>
                  Network Management
                </span>
              </div>
            </Link>
          </div>
        </SidebarMenuButton>
      </SidebarMenuItem>
    </SidebarMenu>
  )
}
