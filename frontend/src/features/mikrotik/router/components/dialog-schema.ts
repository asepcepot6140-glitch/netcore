import { z } from 'zod'

export const routerSchema = z.object({
  name: z.string().trim().min(1, 'Router name is required.'),
  host: z.string().trim().min(1, 'Router host is required.'),
  username: z.string().trim().min(1, 'Username is required.'),
  password: z.string(),
  port: z
    .string()
    .trim()
    .refine((value) => {
      const port = Number(value)
      return Number.isInteger(port) && port >= 1 && port <= 65535
    }, 'Port must be a number between 1 and 65535.'),
})

export type RouterFormValues = z.infer<typeof routerSchema>

export const defaultValues: RouterFormValues = {
  name: '',
  host: '',
  username: '',
  password: '',
  port: '',
}
