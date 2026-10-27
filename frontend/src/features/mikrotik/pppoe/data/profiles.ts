export interface ProfileItem {
  index: number
  flags: string
  name: string
  localAddress: string
  remoteAddress: string
  rateLimit: string
}

export const initialProfiles: ProfileItem[] = [
  {
    index: 0,
    flags: '*',
    name: 'default',
    localAddress: '-',
    remoteAddress: '-',
    rateLimit: '-',
  },
  {
    index: 1,
    flags: '',
    name: '7mb',
    localAddress: '192.168.252.1',
    remoteAddress: 'pool-PPPoE',
    rateLimit: '7M/7M',
  },
]
