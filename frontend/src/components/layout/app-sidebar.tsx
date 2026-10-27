import { useLayout } from '@/context/layout-provider'
import {
  Sidebar,
  SidebarContent,
  SidebarHeader,
  SidebarRail,
} from '@/components/ui/sidebar'
import { AppTitle } from './app-title'
import { sidebarData } from './data/sidebar-data'
import { NavGroup } from './nav-group'

export function AppSidebar() {
  const { collapsible, variant } = useLayout()
  return (
    <Sidebar collapsible={collapsible} variant={variant}>
      <SidebarHeader className='md:-translate-y-0.5'>
        <AppTitle />
      </SidebarHeader>
      <SidebarContent className='md:-translate-y-0.5'>
        {sidebarData.navGroups.map((props) => (
          <NavGroup key={props.title} {...props} />
        ))}
      </SidebarContent>
      <SidebarRail />
    </Sidebar>
  )
}
