import { LayoutDashboard, Server, Router, Settings } from 'lucide-react'
import { type SidebarData } from '../types'

export const sidebarData: SidebarData = {
  navGroups: [
    {
      title: 'General',
      items: [
        {
          title: 'Dashboard',
          url: '/',
          icon: LayoutDashboard,
        },
        {
          title: 'Mikrotik',
          icon: Server,
          items: [
            {
              title: 'Router',
              url: '/mikrotik/router',
            },
            {
              title: 'Interface',
              url: '/mikrotik/interface',
            },
            {
              title: 'PPPoE',
              url: '/mikrotik/pppoe',
            },
            {
              title: 'Hotspot',
              url: '/mikrotik/hotspot',
            },
            {
              title: 'Simple Queue',
              url: '/mikrotik/simple-queue',
            },
          ],
        },
        {
          title: 'GenieACS',
          icon: Router,
          items: [
            {
              title: 'Overview',
              url: '/genieacs/',
            },
            {
              title: 'Devices',
              url: '/genieacs/devices',
            },
          ],
        },
      ],
    },
    {
      title: 'Other',
      items: [
        {
          title: 'Settings',
          url: '/settings',
          icon: Settings,
        },
      ],
    },
  ],
}
