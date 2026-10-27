// RouterOS menyimpan nilai queue sebagai "upload/download" (bps atau byte); helper ini
// mengubahnya jadi teks yang mudah dibaca dan sebaliknya untuk form edit.

const BITS_UNITS = ['bps', 'kbps', 'Mbps', 'Gbps', 'Tbps']
const BYTE_UNITS = ['B', 'KiB', 'MiB', 'GiB', 'TiB', 'PiB']

function splitPair(value: string): [string, string] {
  const [up = '', down = up] = value.split('/')
  return [up, down]
}

function scale(value: number, base: number, units: string[], digits: number) {
  const exponent = Math.min(
    value >= 1 ? Math.floor(Math.log(value) / Math.log(base)) : 0,
    units.length - 1
  )
  const scaled = value / base ** exponent
  return `${scaled.toFixed(exponent === 0 ? 0 : digits)} ${units[exponent]}`
}

function formatPair(
  value: string,
  format: (n: number) => string,
  zero: string
) {
  if (!value) return '-'
  const [up, down] = splitPair(value)
  const part = (text: string) => {
    const n = Number(text)
    return Number.isFinite(n) ? (n > 0 ? format(n) : zero) : text || '-'
  }
  return `${part(up)} / ${part(down)}`
}

export const formatBitsPair = (value: string) =>
  formatPair(value, (n) => scale(n, 1000, BITS_UNITS, 2), '0 bps')

export const formatBits = (value: number) =>
  value > 0 ? scale(value, 1000, BITS_UNITS, 2) : '0 bps'

export const formatBytes = (value: number) =>
  value > 0 ? scale(value, 1024, BYTE_UNITS, 1) : '0 B'

// Satu sisi dari pasangan "upload/download": 'up' = upload, 'down' = download.
export function formatBitsSide(value: string, side: 'up' | 'down') {
  if (!value) return '-'
  const [up, down] = splitPair(value)
  const text = side === 'up' ? up : down
  const n = Number(text)
  if (!Number.isFinite(n)) return text || '-'
  return n > 0 ? scale(n, 1000, BITS_UNITS, 2) : '0 bps'
}

export function pairSide(value: string, side: 'up' | 'down') {
  const [up, down] = splitPair(value)
  return Number(side === 'up' ? up : down) || 0
}

// 1 desimal supaya sama dengan Winbox (mis. 6.37 GiB tampil 6.4 GiB).
export const formatBytesPair = (value: string) =>
  formatPair(value, (n) => scale(n, 1024, BYTE_UNITS, 1), '0 B')

// Total up+down untuk pengurutan kolom.
export function pairTotal(value: string) {
  const [up, down] = splitPair(value)
  return (Number(up) || 0) + (Number(down) || 0)
}

export type RateLevel = 'low' | 'medium' | 'high'

// Seperti Winbox: rate dibanding max-limit per arah, ambil yang terpadat.
// <50% hijau, 50-75% kuning, >=75% merah. Tanpa max-limit (0 = unlimited) dianggap hijau.
export function rateLevel(rate: string, maxLimit: string): RateLevel {
  const [rateUp, rateDown] = splitPair(rate)
  const [limitUp, limitDown] = splitPair(maxLimit)
  const usage = (current: string, limit: string) => {
    const max = Number(limit)
    return max > 0 ? (Number(current) || 0) / max : 0
  }
  const ratio = Math.max(usage(rateUp, limitUp), usage(rateDown, limitDown))
  if (ratio >= 0.75) return 'high'
  if (ratio >= 0.5) return 'medium'
  return 'low'
}

// 11000000 -> "11M", 1500 -> "1500", supaya form edit sama dengan cara user mengetik.
function bitsToInput(text: string) {
  const n = Number(text)
  if (!Number.isSafeInteger(n) || n <= 0) return text
  if (n % 1_000_000_000 === 0) return `${n / 1_000_000_000}G`
  if (n % 1_000_000 === 0) return `${n / 1_000_000}M`
  if (n % 1_000 === 0) return `${n / 1_000}k`
  return text
}

// Tampilan tabel untuk limit: "11000000/11000000" -> "11M/11M", "0/0" -> "unlimited/unlimited".
export function formatLimit(value: string) {
  if (!value) return '-'
  const part = (text: string) =>
    Number(text) === 0 ? 'unlimited' : bitsToInput(text)
  const [up, down] = splitPair(value)
  return `${part(up)}/${part(down)}`
}

// Daftar target tanpa "/32" pada IPv4 tunggal; prefix lain (mis. /24 atau IPv6) tidak diubah.
export function targetToInput(value: string) {
  return value
    .split(',')
    .map((target) =>
      target.trim().replace(/^(\d{1,3}(?:\.\d{1,3}){3})\/32$/, '$1')
    )
    .filter(Boolean)
    .join(', ')
}

export const formatTarget = (value: string) => targetToInput(value) || '-'

// Nilai dari router apa adanya: "0/0" berarti unlimited/unlimited, jadi tidak dikosongkan.
// Max limit dan limit at memakai kosong untuk "tanpa limit" (keepZero=false).
export function limitToInput(value: string, keepZero = false) {
  if (!value) return ''
  const [up, down] = splitPair(value)
  if (!keepZero && Number(up) === 0 && Number(down) === 0) return ''
  return `${bitsToInput(up)}/${bitsToInput(down)}`
}

// Burst time selalu pasangan RX/TX; "0s/0s" berarti 0 detik, ditampilkan "0/0".
export function timeToInput(value: string) {
  if (!value) return ''
  const [up, down] = splitPair(value)
  if (parseInt(up) === 0 && parseInt(down) === 0) return '0/0'
  return `${up}/${down}`
}

// Priority dan queue type ditampilkan sebagai pasangan "rx/tx" apa adanya.
export function pairToInput(value: string) {
  if (!value) return ''
  const [up, down] = splitPair(value)
  return `${up}/${down}`
}

// Bucket size 3 desimal seperti Winbox ("0.1/0.1" -> "0.100/0.100").
export function bucketToInput(value: string) {
  if (!value) return ''
  const part = (text: string) =>
    Number.isFinite(Number(text)) && text !== ''
      ? Number(text).toFixed(3)
      : text
  const [up, down] = splitPair(value)
  return `${part(up)}/${part(down)}`
}
