export type BulkResult = {
  succeeded: number
  failed: number
  firstError: unknown
}

// Runs `task` for every item with limited concurrency and never throws,
// so one failed item doesn't stop the rest of a bulk operation.
export async function runBulk<T>(
  items: T[],
  task: (item: T) => Promise<unknown>,
  concurrency = 3
): Promise<BulkResult> {
  const result: BulkResult = { succeeded: 0, failed: 0, firstError: undefined }
  let cursor = 0

  async function worker() {
    while (cursor < items.length) {
      const item = items[cursor++]
      try {
        await task(item)
        result.succeeded++
      } catch (error) {
        if (result.failed === 0) result.firstError = error
        result.failed++
      }
    }
  }

  await Promise.all(
    Array.from({ length: Math.min(concurrency, items.length) }, worker)
  )

  return result
}
