import { useState, type FormEvent } from 'react'
import { Loader2 } from 'lucide-react'
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
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Switch } from '@/components/ui/switch'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { PasswordInput } from '@/components/password-input'
import type { GenieACSEditTarget, PasswordPolicy } from './genieacs-api'

export type ParameterField = {
  key: string
  label: string
  type?: 'text' | 'number' | 'boolean' | 'password' | 'checklist' | 'select'
  // Choices for 'checklist' and 'select'; a checklist value is a comma-separated subset.
  options?: string[]
  // Display text for option values (select only).
  optionLabels?: Record<string, string>
  // Select options per value of another field; falls back to `options`.
  optionsBy?: { key: string; map: Record<string, string[]> }
  // Options shown but not clickable.
  disabledOptions?: string[]
  // Hide the field while another field's draft equals this value.
  hideWhen?: { key: string; equals: string }
  // Shown but not editable.
  readOnly?: boolean
  // Not editable (and not sent) while another field's draft equals this value.
  disabledWhen?: { key: string; equals: string }
  // Password rules (password only).
  passwordPolicy?: PasswordRule
}

// `specials` set means the device also requires all four character classes.
export type PasswordRule = Pick<PasswordPolicy, 'min_length'> &
  Partial<Pick<PasswordPolicy, 'max_length' | 'specials'>>

const lengthText = (rule: PasswordRule) =>
  rule.max_length === undefined
    ? `At least ${rule.min_length} characters`
    : `${rule.min_length}-${rule.max_length} characters`

const passwordHint = (rule: PasswordRule) =>
  rule.specials === undefined
    ? lengthText(rule)
    : `${lengthText(rule)} with upper case, lower case, a digit and a symbol (${rule.specials}); must start with a letter or digit`

// Backend enforces the device-specific part; empty means "keep current" and is always valid.
function passwordError(password: string, rule: PasswordRule) {
  if (!password) return ''
  const { specials } = rule
  if (
    specials !== undefined &&
    ![...password].every(
      (char) => /[A-Za-z0-9]/.test(char) || specials.includes(char)
    )
  )
    return `Only letters, digits and ${specials} are allowed`
  if (specials !== undefined && !/^[A-Za-z0-9]/.test(password))
    return 'Must start with a letter or digit, not a symbol'
  if (
    password.length < rule.min_length ||
    (rule.max_length !== undefined && password.length > rule.max_length)
  )
    return `Must be ${lengthText(rule).toLowerCase()}`
  if (
    specials !== undefined &&
    ![/[A-Z]/, /[a-z]/, /[0-9]/, /[^A-Za-z0-9]/].every((pattern) =>
      pattern.test(password)
    )
  )
    return `Needs upper case, lower case, a digit and one of ${specials}`
  return ''
}

// Blank shows the rule, an invalid value shows why, a valid value confirms it.
function PasswordFeedback({
  password,
  rule,
}: {
  password: string
  rule: PasswordRule
}) {
  const error = passwordError(password, rule)
  if (error)
    return (
      <p role='alert' className='text-xs text-destructive'>
        {error}
      </p>
    )
  if (password)
    return (
      <p
        role='status'
        className='text-xs font-medium text-emerald-600 dark:text-emerald-400'
      >
        Password meets the requirements
      </p>
    )
  return <p className='text-xs text-muted-foreground'>{passwordHint(rule)}</p>
}

export type EditTarget = {
  title: string
  endpoint: GenieACSEditTarget
  path: string
  fields: ParameterField[]
  values: Record<string, string>
  // When set, fields are split into Basic (these keys) and Advanced (all) tabs.
  basicKeys?: string[]
}

type Draft = Record<string, string>

const isTrue = (value: string | undefined) =>
  ['true', '1'].includes((value ?? '').trim().toLowerCase())

const splitList = (value: string | undefined) =>
  (value ?? '')
    .split(',')
    .map((item) => item.trim())
    .filter(Boolean)

// Option order keeps the draft comparable with its initial value.
const joinInOrder = (options: string[], items: string[]) =>
  options.filter((option) => items.includes(option)).join(',')

function buildDraft(fields: ParameterField[], values: Draft): Draft {
  const draft: Draft = {}
  for (const field of fields) {
    if (field.type === 'password') draft[field.key] = ''
    else if (field.type === 'checklist')
      draft[field.key] = joinInOrder(
        field.options ?? [],
        splitList(values[field.key])
      )
    else if (field.type === 'boolean')
      draft[field.key] = isTrue(values[field.key]) ? 'true' : 'false'
    else draft[field.key] = values[field.key] ?? ''
  }
  return draft
}

type EditFormProps = {
  target: EditTarget
  isSaving: boolean
  onCancel: () => void
  onSubmit: (changed: Draft) => void
}

function EditForm({ target, isSaving, onCancel, onSubmit }: EditFormProps) {
  const [initial] = useState(() => buildDraft(target.fields, target.values))
  const [draft, setDraft] = useState(initial)

  // Password fields are write-only: blank means "keep current".
  const isHidden = (field: ParameterField) =>
    field.hideWhen !== undefined &&
    draft[field.hideWhen.key] === field.hideWhen.equals

  const isDisabled = (field: ParameterField) =>
    field.readOnly === true ||
    (field.disabledWhen !== undefined &&
      draft[field.disabledWhen.key] === field.disabledWhen.equals)

  const changed: Draft = {}
  for (const field of target.fields) {
    if (isHidden(field) || isDisabled(field)) continue
    // A field that just became enabled is sent even if unchanged, so it applies together with its switch.
    const wasDisabled =
      field.disabledWhen !== undefined &&
      initial[field.disabledWhen.key] === field.disabledWhen.equals
    const differs =
      field.type === 'password'
        ? draft[field.key] !== ''
        : draft[field.key] !== initial[field.key] || wasDisabled
    if (differs) changed[field.key] = draft[field.key]
  }
  const hasChanges = Object.keys(changed).length > 0
  const hasInvalidPassword = target.fields.some(
    (field) =>
      field.type === 'password' &&
      field.passwordPolicy !== undefined &&
      passwordError(draft[field.key], field.passwordPolicy) !== ''
  )

  const optionsFor = (field: ParameterField, values: Draft) =>
    (field.optionsBy && field.optionsBy.map[values[field.optionsBy.key]]) ||
    field.options ||
    []

  const setValue = (key: string, value: string) =>
    setDraft((current) => {
      const next = { ...current, [key]: value }
      // A dependent select whose value is no longer offered falls back to its largest option.
      for (const field of target.fields) {
        if (field.optionsBy?.key !== key) continue
        const options = optionsFor(field, next)
        if (options.length > 0 && !options.includes(next[field.key]))
          next[field.key] = options[options.length - 1]
      }
      return next
    })

  function handleSubmit(event: FormEvent) {
    event.preventDefault()
    if (hasChanges && !hasInvalidPassword) onSubmit(changed)
  }

  const renderFields = (list: ParameterField[]) => (
    <div className='space-y-3'>
      {list.map((field) => {
        if (isHidden(field)) return null
        const id = `edit-${field.key}`
        if (field.type === 'boolean') {
          return (
            <div key={field.key} className='flex items-center gap-3'>
              <Label htmlFor={id}>{field.label}</Label>
              <Switch
                id={id}
                checked={draft[field.key] === 'true'}
                onCheckedChange={(checked) =>
                  setValue(field.key, checked ? 'true' : 'false')
                }
              />
            </div>
          )
        }
        if (field.type === 'select') {
          const current = draft[field.key]
          const options = optionsFor(field, draft)
          // Keep an unlisted current value selectable so it is not shown blank.
          const items =
            current && !options.includes(current)
              ? [current, ...options]
              : options
          return (
            <div key={field.key} className='space-y-1.5'>
              <Label htmlFor={id}>{field.label}</Label>
              <Select
                value={current}
                disabled={isDisabled(field)}
                onValueChange={(value) => setValue(field.key, value)}
              >
                <SelectTrigger id={id} className='w-full'>
                  <SelectValue placeholder='Select' />
                </SelectTrigger>
                <SelectContent>
                  {items.map((option) => (
                    <SelectItem key={option} value={option}>
                      {field.optionLabels?.[option] ?? option}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
          )
        }
        if (field.type === 'checklist') {
          const options = field.options ?? []
          if (options.length === 0) return null
          const selected = splitList(draft[field.key])
          return (
            <div key={field.key} className='space-y-2'>
              <Label>{field.label}</Label>
              <div className='grid grid-cols-4 gap-x-3 gap-y-2'>
                {options.map((option) => {
                  const optionId = `${id}-${option}`
                  const isDisabled =
                    field.disabledOptions?.includes(option) ?? false
                  return (
                    <div key={option} className='flex items-center gap-2'>
                      <Checkbox
                        id={optionId}
                        disabled={isDisabled}
                        checked={selected.includes(option)}
                        onCheckedChange={(checked) =>
                          setValue(
                            field.key,
                            joinInOrder(
                              options,
                              checked === true
                                ? [...selected, option]
                                : selected.filter((item) => item !== option)
                            )
                          )
                        }
                      />
                      <Label htmlFor={optionId} className='font-normal'>
                        {option}
                      </Label>
                    </div>
                  )
                })}
              </div>
            </div>
          )
        }
        return (
          <div key={field.key} className='space-y-1.5'>
            <Label htmlFor={id}>{field.label}</Label>
            {field.type === 'password' ? (
              <PasswordInput
                id={id}
                autoComplete='new-password'
                placeholder='Leave blank to keep unchanged'
                value={draft[field.key]}
                aria-invalid={
                  field.passwordPolicy !== undefined &&
                  passwordError(draft[field.key], field.passwordPolicy) !== ''
                }
                onChange={(event) => setValue(field.key, event.target.value)}
              />
            ) : (
              <Input
                id={id}
                autoComplete='off'
                disabled={isDisabled(field)}
                inputMode={field.type === 'number' ? 'numeric' : undefined}
                value={draft[field.key]}
                onChange={(event) => setValue(field.key, event.target.value)}
              />
            )}
            {field.type === 'password' && field.passwordPolicy && (
              <PasswordFeedback
                password={draft[field.key]}
                rule={field.passwordPolicy}
              />
            )}
          </div>
        )
      })}
    </div>
  )

  const { basicKeys } = target

  return (
    <form onSubmit={handleSubmit} className='space-y-4' autoComplete='off'>
      {basicKeys ? (
        <Tabs defaultValue='basic'>
          <TabsList className='w-full'>
            <TabsTrigger value='basic'>Basic</TabsTrigger>
            <TabsTrigger value='advanced'>Advanced</TabsTrigger>
          </TabsList>
          <TabsContent value='basic' className='pt-2'>
            {renderFields(
              target.fields.filter((field) => basicKeys.includes(field.key))
            )}
          </TabsContent>
          <TabsContent value='advanced' className='pt-2'>
            {renderFields(target.fields)}
          </TabsContent>
        </Tabs>
      ) : (
        renderFields(target.fields)
      )}
      <DialogFooter>
        <Button type='button' variant='outline' onClick={onCancel}>
          Cancel
        </Button>
        <Button
          type='submit'
          disabled={isSaving || !hasChanges || hasInvalidPassword}
        >
          {isSaving && <Loader2 className='animate-spin' />}
          Save Changes
        </Button>
      </DialogFooter>
    </form>
  )
}

type EditParametersDialogProps = {
  target: EditTarget | null
  isSaving: boolean
  onClose: () => void
  onSubmit: (changed: Draft) => void
}

export function EditParametersDialog({
  target,
  isSaving,
  onClose,
  onSubmit,
}: EditParametersDialogProps) {
  return (
    <Dialog
      open={target !== null}
      onOpenChange={(open) => !open && !isSaving && onClose()}
    >
      <DialogContent className='max-h-[90vh] overflow-y-auto sm:max-w-md'>
        <DialogHeader>
          <DialogTitle>{target?.title}</DialogTitle>
          <DialogDescription>
            Saving sends these changes to the device.
          </DialogDescription>
        </DialogHeader>
        {target && (
          <EditForm
            key={`${target.endpoint}:${target.path}:${target.fields.map((field) => field.key).join(',')}`}
            target={target}
            isSaving={isSaving}
            onCancel={onClose}
            onSubmit={onSubmit}
          />
        )}
      </DialogContent>
    </Dialog>
  )
}
