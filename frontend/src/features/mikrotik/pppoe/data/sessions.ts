export interface ActiveSession {
  index: number
  flags: string
  name: string
  service: string
  callerId: string
  address: string
  uptime: string
  sessionId: string
}

export const initialActiveSessions: ActiveSession[] = [
  {
    index: 0,
    flags: '',
    name: 'Mama raya csr rt4',
    service: 'pppoe',
    callerId: 'E8:6E:44:75:25:D7',
    address: '192.168.253.155',
    uptime: '3w5d44s',
    sessionId: '0x81C21750',
  },
  {
    index: 1,
    flags: '',
    name: 'Sugiyanto csr rt1',
    service: 'pppoe',
    callerId: 'F0:3F:95:59:8A:1C',
    address: '192.168.253.231',
    uptime: '3w5d30s',
    sessionId: '0x81C2176F',
  },
]
