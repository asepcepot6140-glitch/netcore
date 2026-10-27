import { type ReactNode } from 'react'
import { cn } from '@/lib/utils'

type TokenKind =
  | 'comment'
  | 'string'
  | 'variable'
  | 'command'
  | 'param'
  | 'path'
  | 'keyword'
  | 'number'
  | 'punct'

// Urutan harus sama dengan grup capture di tokenPattern.
const tokenKinds: TokenKind[] = [
  'comment',
  'string',
  'variable',
  'command',
  'param',
  'path',
  'keyword',
  'number',
  'punct',
]

const tokenPattern = new RegExp(
  [
    String.raw`(#[^\n]*)`,
    String.raw`("(?:\\[\s\S]|[^"\\])*"?)`,
    String.raw`(\$(?:\{[^}\n]*\}|[A-Za-z_][\w-]*))`,
    String.raw`(:[A-Za-z][\w-]*)`,
    String.raw`([A-Za-z][\w-]*(?==))`,
    String.raw`((?<![\w"$.-])\/[\w-]+(?:\/[\w-]+)*)`,
    String.raw`\b(add|set|remove|print|get|find|enable|disable|foreach|do|if|else|while|for|in|on-error|and|or)\b`,
    String.raw`(\b\d+(?:\.\d+)?[a-z]*\b)`,
    String.raw`([{}()[\];])`,
  ].join('|'),
  'g'
)

const tokenClass: Record<TokenKind, string> = {
  comment: 'italic text-muted-foreground',
  string: 'text-emerald-600 dark:text-emerald-400',
  variable: 'text-sky-600 dark:text-sky-400',
  command: 'font-semibold text-violet-600 dark:text-violet-400',
  param: 'text-orange-600 dark:text-orange-400',
  path: 'font-semibold text-amber-600 dark:text-amber-400',
  keyword: 'text-fuchsia-600 dark:text-fuchsia-400',
  number: 'text-cyan-600 dark:text-cyan-400',
  punct: 'text-muted-foreground',
}

// Mewarnai script RouterOS secara sederhana (bukan parser penuh).
function highlight(source: string): ReactNode[] {
  const nodes: ReactNode[] = []
  let last = 0
  for (const match of source.matchAll(tokenPattern)) {
    const start = match.index ?? 0
    if (start > last) nodes.push(source.slice(last, start))
    const kind = tokenKinds[match.slice(1).findIndex((g) => g !== undefined)]
    nodes.push(
      <span key={start} className={tokenClass[kind]}>
        {match[0]}
      </span>
    )
    last = start + match[0].length
  }
  if (last < source.length) nodes.push(source.slice(last))
  return nodes
}

// Kedua lapisan harus memakai metrik teks yang identik agar posisi caret sejajar.
const layer =
  'block w-full min-h-28 px-3 py-2 font-mono text-xs leading-5 whitespace-pre-wrap [overflow-wrap:anywhere] [tab-size:2]'

export function CodeEditor({
  id,
  value,
  onChange,
  placeholder,
  className,
}: {
  id?: string
  value: string
  onChange: (value: string) => void
  placeholder?: string
  className?: string
}) {
  return (
    <div
      className={cn(
        'relative overflow-hidden rounded-md border bg-muted/30 shadow-xs transition-shadow focus-within:border-ring focus-within:ring-[3px] focus-within:ring-ring/50',
        className
      )}
    >
      {/* Lapisan berwarna; tingginya mengikuti isi sehingga editor otomatis melebar ke bawah. */}
      <pre aria-hidden className={cn(layer, 'pointer-events-none m-0')}>
        {highlight(value)}
        {'\n'}
      </pre>
      <textarea
        id={id}
        value={value}
        onChange={(event) => onChange(event.target.value)}
        placeholder={placeholder}
        spellCheck={false}
        autoComplete='off'
        autoCapitalize='off'
        className={cn(
          layer,
          'absolute inset-0 h-full resize-none overflow-hidden border-0 bg-transparent text-transparent caret-foreground outline-none selection:bg-primary/25 placeholder:text-muted-foreground'
        )}
      />
    </div>
  )
}
