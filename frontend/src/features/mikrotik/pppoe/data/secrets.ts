export interface PppoeSecret {
  index: number
  flags: string
  name: string
  service: string
  password: string
  profile: string
  localAddress?: string
  remoteAddress?: string
  lastLoggedOut: string
  lastCallerId: string
  lastDisconnectReason: string
}

export const initialPppoeSecrets: PppoeSecret[] = [
  {
    index: 0,
    flags: '',
    name: 'adesujana',
    service: 'pppoe',
    password: '1234',
    profile: 'RT 06',
    localAddress: '',
    remoteAddress: '',
    lastLoggedOut: 'oct/15/2026 14:55:20',
    lastCallerId: 'E4:CA:12:E8:C2:3D',
    lastDisconnectReason: 'hung-up',
  },
  {
    index: 1,
    flags: '',
    name: 'sunarto',
    service: 'pppoe',
    password: '1234',
    profile: 'RT 06',
    localAddress: '',
    remoteAddress: '',
    lastLoggedOut: 'oct/15/2026 14:51:58',
    lastCallerId: '8C:E1:17:D2:FA:DC',
    lastDisconnectReason: 'hung-up',
  },
]
