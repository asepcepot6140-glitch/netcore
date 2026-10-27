import { ChevronDown } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Checkbox } from '@/components/ui/checkbox'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@/components/ui/popover'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { CodeEditor } from './code-editor'

export interface Field<T> {
  key: keyof T & string
  label: string
  placeholder: string
  required?: boolean
  options?: string[]
  // checkbox: nilai 'yes'/'no'. code: editor script berwarna. rate: input RX/TX yang digabung jadi "rx/tx".
  // pair: dua kolom RX/TX "rx/tx" (Select bila ada options, selain itu Input).
  // multi: pilih banyak dari options, disimpan dipisah koma.
  type?: 'checkbox' | 'code' | 'rate' | 'pair' | 'multi'
  // rate: tampilkan "unlimited" untuk nilai 0 dan sediakan dropdown untuk memilihnya (seperti Winbox).
  unlimited?: boolean
  // rate: pilihan nilai cepat di dropdown (mis. 64k, 1M); input manual tetap bisa.
  presets?: string[]
  // Field tetap tampil tetapi tidak bisa diisi bila fungsi ini true (mis. bergantung pada checkbox lain).
  disabled?: (data: Partial<T>) => boolean
}

export interface FormTab {
  value: string
  label: string
  keys: string[]
}

export function FormDialog<T extends object>({
  open,
  onOpenChange,
  title,
  idPrefix,
  fields,
  data,
  onChange,
  onSubmit,
  submitLabel,
  passwordOptional = false,
  basicKeys,
  tabs,
  wide = false,
  readOnly = false,
}: {
  open: boolean
  onOpenChange: (open: boolean) => void
  title: string
  idPrefix: string
  fields: Field<T>[]
  data: Partial<T>
  onChange: (data: Partial<T>) => void
  onSubmit: (event: React.FormEvent) => void
  submitLabel: string
  passwordOptional?: boolean
  // When set, fields are split into Basic (these keys) and Advanced (all) tabs.
  basicKeys?: string[]
  // Custom tabs, each showing only its own keys. Takes precedence over basicKeys.
  tabs?: FormTab[]
  // Dialog lebih lebar, untuk field berisi script.
  wide?: boolean
  // Tampilkan data saja: semua field nonaktif dan tombol simpan disembunyikan.
  readOnly?: boolean
}) {
  const renderField = (field: Field<T>) => {
    const id = `${idPrefix}-${field.key}`
    const value = String(data[field.key] ?? '')
    const required =
      field.required || (field.key === 'password' && !passwordOptional)
    const setValue = (next: string) =>
      onChange({ ...data, [field.key]: next } as Partial<T>)
    const isDisabled = readOnly || (field.disabled?.(data) ?? false)

    if (field.type === 'checkbox') {
      return (
        <div key={field.key} className='flex items-center gap-2'>
          <Checkbox
            id={id}
            checked={value === 'yes'}
            disabled={isDisabled}
            onCheckedChange={(checked) => setValue(checked ? 'yes' : 'no')}
          />
          <Label htmlFor={id}>{field.label}</Label>
        </div>
      )
    }

    if (field.type === 'rate') {
      // Format RouterOS: "rx/tx" lalu opsi burst dipisah spasi; opsi tambahan dipertahankan.
      const [rates = '', ...extras] = value.split(' ')
      const [rx = '', tx = ''] = rates.split('/')
      const setRate = (nextRx: string, nextTx: string) => {
        const base = nextTx ? `${nextRx}/${nextTx}` : nextRx
        setValue(base ? [base, ...extras].join(' ') : '')
      }
      const clean = (text: string) => text.replace(/\s/g, '')
      // "unlimited" disimpan sebagai 0 (arti RouterOS); angka 0 ditampilkan kembali sebagai unlimited.
      const toStored = (text: string) =>
        field.unlimited && text.toLowerCase() === 'unlimited' ? '0' : text
      const toShown = (text: string) =>
        field.unlimited && text === '0' ? 'unlimited' : text
      const sides = [
        { side: 'rx' as const, label: 'RX (upload)', current: rx },
        { side: 'tx' as const, label: 'TX (download)', current: tx },
      ]
      const update = (side: 'rx' | 'tx', text: string) => {
        const stored = side === 'rx' ? rx : tx
        // Backspace pada "unlimited" langsung mengosongkan, bukan menyisakan "unlimite".
        const erasing =
          field.unlimited &&
          stored === '0' &&
          text.length < 'unlimited'.length &&
          'unlimited'.startsWith(text.toLowerCase())
        const next = erasing ? '' : toStored(clean(text))
        if (side === 'rx') setRate(next, tx)
        else setRate(rx, next)
      }
      return (
        <div key={field.key} className='space-y-2'>
          <Label htmlFor={id}>{field.label}</Label>
          <div className='grid grid-cols-2 gap-3'>
            {sides.map(({ side, label, current }) => (
              <div key={side} className='min-w-0 space-y-1'>
                <span className='text-xs text-muted-foreground'>{label}</span>
                <div className='relative'>
                  <Input
                    id={side === 'rx' ? id : undefined}
                    aria-label={`${field.label} ${side.toUpperCase()}`}
                    value={toShown(current)}
                    onChange={(event) => update(side, event.target.value)}
                    placeholder={field.unlimited ? 'unlimited' : '5M'}
                    className={
                      field.unlimited || field.presets ? 'pr-9' : undefined
                    }
                    disabled={isDisabled}
                    autoComplete='off'
                  />
                  {(field.unlimited || field.presets) && (
                    <DropdownMenu>
                      <DropdownMenuTrigger asChild>
                        <Button
                          type='button'
                          variant='ghost'
                          size='icon'
                          aria-label={`${field.label} ${side.toUpperCase()} options`}
                          disabled={isDisabled}
                          className='absolute top-0 right-0 size-9 text-muted-foreground'
                        >
                          <ChevronDown className='size-4' />
                        </Button>
                      </DropdownMenuTrigger>
                      <DropdownMenuContent
                        align='end'
                        className='max-h-64 min-w-24'
                      >
                        {[
                          ...(field.presets ?? []),
                          ...(field.unlimited ? ['unlimited'] : []),
                        ].map((preset) => (
                          <DropdownMenuItem
                            key={preset}
                            onSelect={() => update(side, preset)}
                          >
                            {preset}
                          </DropdownMenuItem>
                        ))}
                      </DropdownMenuContent>
                    </DropdownMenu>
                  )}
                </div>
              </div>
            ))}
          </div>
        </div>
      )
    }

    if (field.type === 'pair') {
      const [rx = '', tx = ''] = value.split('/')
      const setPair = (nextRx: string, nextTx: string) =>
        setValue(nextTx ? `${nextRx}/${nextTx}` : nextRx)
      // Select: memilih satu sisi saat sisi lain kosong mengisi keduanya (RouterOS memakai pasangan).
      const pickSide = (side: 'rx' | 'tx', next: string) => {
        const other = side === 'rx' ? tx : rx
        const mirrored = other || next
        if (side === 'rx') setPair(next, mirrored)
        else setPair(mirrored, next)
      }
      const clean = (text: string) => text.replace(/\s/g, '')
      const sides = [
        { side: 'rx' as const, label: 'RX (upload)', current: rx },
        { side: 'tx' as const, label: 'TX (download)', current: tx },
      ]
      return (
        <div key={field.key} className='space-y-2'>
          <Label htmlFor={id}>{field.label}</Label>
          <div className='grid grid-cols-2 gap-3'>
            {sides.map(({ side, label, current }) => (
              <div key={side} className='min-w-0 space-y-1'>
                <span className='text-xs text-muted-foreground'>{label}</span>
                {field.options ? (
                  <Select
                    value={current}
                    disabled={isDisabled}
                    onValueChange={(next) => pickSide(side, next)}
                  >
                    <SelectTrigger
                      id={side === 'rx' ? id : undefined}
                      aria-label={`${field.label} ${side.toUpperCase()}`}
                      className='w-full'
                    >
                      <SelectValue placeholder={field.placeholder} />
                    </SelectTrigger>
                    <SelectContent>
                      {(current && !field.options.includes(current)
                        ? [current, ...field.options]
                        : field.options
                      ).map((option) => (
                        <SelectItem key={option} value={option}>
                          {option}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                ) : (
                  <Input
                    id={side === 'rx' ? id : undefined}
                    aria-label={`${field.label} ${side.toUpperCase()}`}
                    value={current}
                    onChange={(event) => {
                      const next = clean(event.target.value)
                      if (side === 'rx') setPair(next, tx)
                      else setPair(rx, next)
                    }}
                    placeholder={field.placeholder}
                    disabled={isDisabled}
                    autoComplete='off'
                  />
                )}
              </div>
            ))}
          </div>
        </div>
      )
    }

    if (field.type === 'multi') {
      const selected = value ? value.split(',').filter(Boolean) : []
      // Nilai yang sudah tersimpan di router tetap tampil walau tidak ada di daftar opsi.
      const options = [
        ...(field.options ?? []),
        ...selected.filter((item) => !field.options?.includes(item)),
      ]
      const toggle = (option: string, checked: boolean) =>
        setValue(
          options
            .filter((item) =>
              item === option ? checked : selected.includes(item)
            )
            .join(',')
        )
      return (
        <div key={field.key} className='space-y-2'>
          <Label htmlFor={id}>{field.label}</Label>
          <Popover modal>
            <PopoverTrigger asChild>
              <Button
                id={id}
                type='button'
                variant='outline'
                disabled={isDisabled}
                className='w-full justify-between font-normal'
              >
                <span
                  className={`truncate ${selected.length === 0 ? 'text-muted-foreground' : ''}`}
                >
                  {selected.length > 0
                    ? selected.join(', ')
                    : field.placeholder}
                </span>
                <ChevronDown className='size-4 shrink-0 opacity-50' />
              </Button>
            </PopoverTrigger>
            <PopoverContent
              align='start'
              className='max-h-60 w-(--radix-popover-trigger-width) overflow-y-auto p-1'
            >
              {options.map((option) => (
                <Label
                  key={option}
                  className='flex cursor-pointer items-center gap-2 rounded-sm px-2 py-1.5 font-normal hover:bg-accent'
                >
                  <Checkbox
                    checked={selected.includes(option)}
                    onCheckedChange={(checked) => toggle(option, !!checked)}
                  />
                  <span className='truncate'>{option}</span>
                </Label>
              ))}
            </PopoverContent>
          </Popover>
        </div>
      )
    }

    if (field.type === 'code') {
      return (
        <div key={field.key} className='space-y-2'>
          <Label htmlFor={id}>{field.label}</Label>
          <CodeEditor
            id={id}
            value={value}
            onChange={setValue}
            placeholder={field.placeholder}
          />
        </div>
      )
    }

    return (
      <div key={field.key} className='space-y-2'>
        <Label htmlFor={id}>
          {field.label}
          {required && <span className='text-destructive'> *</span>}
        </Label>
        {field.options ? (
          <Select
            value={value}
            disabled={isDisabled}
            onValueChange={(next) =>
              onChange({ ...data, [field.key]: next } as Partial<T>)
            }
          >
            <SelectTrigger id={id} className='w-full'>
              <SelectValue placeholder={field.placeholder} />
            </SelectTrigger>
            <SelectContent>
              {(value && !field.options.includes(value)
                ? [value, ...field.options]
                : field.options
              ).map((option) => (
                <SelectItem key={option} value={option}>
                  {option}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        ) : (
          <Input
            id={id}
            value={value}
            onChange={(event) =>
              onChange({
                ...data,
                [field.key]: event.target.value,
              } as Partial<T>)
            }
            placeholder={field.placeholder}
            required={required}
            disabled={isDisabled}
            autoComplete={field.key === 'password' ? 'new-password' : 'off'}
            type='text'
          />
        )}
      </div>
    )
  }

  const renderFields = (list: Field<T>[]) => (
    <fieldset
      disabled={readOnly}
      className='m-0 flex min-w-0 flex-col gap-4 border-0 p-0'
    >
      {list.map(renderField)}
    </fieldset>
  )

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent
        // Dikunci di atas (bukan di tengah) agar tab list tidak bergeser saat tinggi dialog berubah.
        className={`top-[8vh] max-h-[84vh] translate-y-0 overflow-y-auto p-0 ${wide ? 'sm:max-w-2xl' : 'sm:max-w-md'}`}
        showCloseButton={false}
      >
        <DialogHeader className='border-b px-6 py-3'>
          <DialogTitle>{title}</DialogTitle>
          <DialogDescription className='sr-only'>
            {title} form
          </DialogDescription>
        </DialogHeader>
        <form
          onSubmit={onSubmit}
          className='space-y-4 px-6 pt-2.5 pb-6'
          autoComplete='off'
        >
          {tabs ? (
            <Tabs defaultValue={tabs[0]?.value}>
              <TabsList className='w-full'>
                {tabs.map((tab) => (
                  <TabsTrigger key={tab.value} value={tab.value}>
                    {tab.label}
                  </TabsTrigger>
                ))}
              </TabsList>
              {tabs.map((tab) => (
                <TabsContent key={tab.value} value={tab.value} className='pt-2'>
                  {renderFields(
                    fields.filter((field) => tab.keys.includes(field.key))
                  )}
                </TabsContent>
              ))}
            </Tabs>
          ) : basicKeys ? (
            <Tabs defaultValue='basic'>
              <TabsList className='w-full'>
                <TabsTrigger value='basic'>Basic</TabsTrigger>
                <TabsTrigger value='advanced'>Advanced</TabsTrigger>
              </TabsList>
              <TabsContent value='basic' className='pt-2'>
                {renderFields(
                  fields.filter((field) => basicKeys.includes(field.key))
                )}
              </TabsContent>
              <TabsContent value='advanced' className='pt-2'>
                {renderFields(fields)}
              </TabsContent>
            </Tabs>
          ) : (
            renderFields(fields)
          )}
          <DialogFooter className='pt-2'>
            <Button
              type='button'
              variant='outline'
              onClick={() => onOpenChange(false)}
            >
              {readOnly ? 'Close' : 'Cancel'}
            </Button>
            {!readOnly && <Button type='submit'>{submitLabel}</Button>}
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
