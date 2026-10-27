import { useMemo, useState, type ReactNode } from 'react'
import {
  Box,
  Checkbox,
  InputAdornment,
  Paper,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TablePagination,
  TableRow,
  TableSortLabel,
  TextField,
} from '@mui/material'
import { Plus, Search, Trash2 } from 'lucide-react'
import { cn } from '@/lib/utils'
import { useIsMobile } from '@/hooks/use-mobile'
import { Button } from '@/components/ui/button'
import { RefreshLine } from '@/components/refresh-line'

const ROWS_PER_PAGE_OPTIONS = [5, 10, 25, 50]

export interface DataTableColumn<T> {
  header: string
  cell: (row: T) => ReactNode
  className?: string
  align?: 'left' | 'center' | 'right'
  // Judul kolom dikosongkan di tabel desktop (tetap dipakai sebagai label di tampilan mobile).
  hideHeader?: boolean
  // Di tampilan mobile, baris label disembunyikan bila selnya kosong.
  hideEmpty?: boolean
  // Kolom sempit untuk penanda kecil, mis. satu huruf.
  narrow?: boolean
  getSortValue?: (row: T) => string | number | null | undefined
}

interface MuiDataTableProps<T> {
  columns: DataTableColumn<T>[]
  rows: T[]
  getRowId: (row: T) => string | number
  actions?: (row: T) => ReactNode
  addLabel?: string
  onAdd?: () => void
  toolbarBeforeAdd?: ReactNode
  toolbarAfterAdd?: ReactNode
  isLoading?: boolean
  isRefreshing?: boolean
  loadingLabel?: string
  // Enables the checkbox column; called with the selected rows that are still visible.
  onBulkDelete?: (rows: T[]) => void
  bulkDeleteLabel?: string
  // Hides the row number column (#) when false.
  showIndex?: boolean
}

export function MuiDataTable<T extends object>({
  columns,
  rows,
  getRowId,
  actions,
  addLabel,
  onAdd,
  toolbarBeforeAdd,
  toolbarAfterAdd,
  isLoading = false,
  isRefreshing = false,
  loadingLabel = 'Loading...',
  onBulkDelete,
  bulkDeleteLabel = 'Delete',
  showIndex = true,
}: MuiDataTableProps<T>) {
  const isMobile = useIsMobile()
  const [search, setSearch] = useState('')
  const [page, setPage] = useState(0)
  const [rowsPerPage, setRowsPerPage] = useState(10)
  const [selectedIds, setSelectedIds] = useState<Set<string | number>>(
    () => new Set()
  )
  const [sort, setSort] = useState<{
    column: DataTableColumn<T>
    direction: 'asc' | 'desc'
  } | null>(null)
  const filtered = useMemo(() => {
    const query = search.trim().toLowerCase()
    const matchingRows = query
      ? rows.filter((row) =>
          Object.values(row).some((value) =>
            String(value).toLowerCase().includes(query)
          )
        )
      : rows

    if (!sort) return matchingRows

    return [...matchingRows].sort((first, second) => {
      const firstValue = sort.column.getSortValue?.(first)
      const secondValue = sort.column.getSortValue?.(second)
      if (firstValue == null || secondValue == null) {
        if (firstValue == null && secondValue == null) return 0
        return firstValue == null ? 1 : -1
      }

      const comparison =
        typeof firstValue === 'number' && typeof secondValue === 'number'
          ? firstValue - secondValue
          : String(firstValue).localeCompare(String(secondValue), undefined, {
              numeric: true,
              sensitivity: 'base',
            })
      return sort.direction === 'asc' ? comparison : -comparison
    })
  }, [rows, search, sort])

  const lastPage = Math.max(0, Math.ceil(filtered.length / rowsPerPage) - 1)
  const currentPage = Math.min(page, lastPage)
  const visible = filtered.slice(
    currentPage * rowsPerPage,
    currentPage * rowsPerPage + rowsPerPage
  )

  const selectable = Boolean(onBulkDelete)

  // Drop selections whose rows are gone (deleted, or another router/ACS was loaded),
  // so a re-registered device or a reused ID never comes back pre-selected.
  if (selectedIds.size > 0) {
    const rowIds = new Set(rows.map(getRowId))
    if ([...selectedIds].some((id) => !rowIds.has(id))) {
      setSelectedIds(new Set([...selectedIds].filter((id) => rowIds.has(id))))
    }
  }
  // The checkbox column replaces the row-number column when bulk selection is on.
  const colCount = columns.length + (actions ? 1 : 0) + 1
  // Only rows still present after search/filter count, so hidden rows are never deleted by accident.
  const selectedRows = selectable
    ? filtered.filter((row) => selectedIds.has(getRowId(row)))
    : []
  const selectedOnPage = visible.filter((row) =>
    selectedIds.has(getRowId(row))
  ).length
  const allOnPageSelected =
    visible.length > 0 && selectedOnPage === visible.length

  function toggleRow(row: T) {
    setSelectedIds((current) => {
      const next = new Set(current)
      const id = getRowId(row)
      if (next.has(id)) next.delete(id)
      else next.add(id)
      return next
    })
  }

  function togglePage() {
    setSelectedIds((current) => {
      const next = new Set(current)
      for (const row of visible) {
        if (allOnPageSelected) next.delete(getRowId(row))
        else next.add(getRowId(row))
      }
      return next
    })
  }

  return (
    <Paper
      variant='outlined'
      sx={{
        backgroundColor: 'transparent',
        backgroundImage: 'none',
        borderRadius: '12px',
        overflow: 'hidden',
        // Prevent a flex-column parent from shrinking the table; the page scrolls instead.
        flexShrink: 0,
      }}
    >
      <Box
        sx={{
          p: 1,
          display: 'flex',
          flexDirection: { xs: 'column', sm: 'row' },
          alignItems: { xs: 'stretch', sm: 'center' },
          justifyContent: 'space-between',
          gap: 1,
        }}
      >
        <div className='flex w-full flex-col gap-2 sm:w-auto sm:flex-row sm:items-center'>
          {toolbarBeforeAdd}
          {onAdd && (
            <Button
              size='sm'
              onClick={onAdd}
              className='h-7 w-full gap-2 sm:w-auto'
            >
              <Plus className='size-4' />
              {addLabel}
            </Button>
          )}
          {toolbarAfterAdd}
          {onBulkDelete && selectedRows.length > 0 && (
            <Button
              size='sm'
              variant='destructive'
              onClick={() => onBulkDelete(selectedRows)}
              className='h-7 w-full gap-2 sm:w-auto'
            >
              <Trash2 className='size-4' />
              {bulkDeleteLabel} ({selectedRows.length})
            </Button>
          )}
        </div>
        <TextField
          size='small'
          placeholder='Search...'
          value={search}
          onChange={(event) => {
            setSearch(event.target.value)
            setPage(0)
          }}
          sx={{
            width: { xs: '100%', sm: 172 },
            marginLeft: 'auto',
            '& .MuiInputBase-root': { height: 28, fontSize: 14 },
            '& .MuiInputBase-input': { py: 0 },
          }}
          slotProps={{
            input: {
              startAdornment: (
                <InputAdornment position='start'>
                  <Search size={16} />
                </InputAdornment>
              ),
            },
          }}
        />
      </Box>
      {isMobile ? (
        // Layar kecil: tiap baris jadi kartu label/nilai agar semua kolom terlihat tanpa geser.
        <div>
          <RefreshLine active={isRefreshing} className='rounded-none' />
          {isLoading ? (
            <p className='py-2 text-center text-[13px] text-muted-foreground'>
              {loadingLabel}
            </p>
          ) : visible.length === 0 ? (
            <p className='py-2 text-center text-sm text-muted-foreground'>
              {search ? 'No results found.' : 'No data.'}
            </p>
          ) : (
            <>
              {selectable && (
                <label className='flex items-center gap-2 border-b px-3 py-1 text-xs text-muted-foreground'>
                  <Checkbox
                    size='small'
                    checked={allOnPageSelected}
                    indeterminate={selectedOnPage > 0 && !allOnPageSelected}
                    onChange={togglePage}
                    sx={{ p: 0.5, ml: -0.5 }}
                  />
                  Select all on this page
                </label>
              )}
              <ul className='divide-y'>
                {visible.map((row, index) => {
                  const isSelected =
                    selectable && selectedIds.has(getRowId(row))
                  return (
                    <li
                      key={getRowId(row)}
                      className={cn(
                        'space-y-1 px-3 py-2 text-sm',
                        isSelected && 'bg-muted/50'
                      )}
                    >
                      <div className='flex min-h-7 items-center justify-between'>
                        {selectable ? (
                          <Checkbox
                            size='small'
                            checked={isSelected}
                            onChange={() => toggleRow(row)}
                            sx={{ p: 0.5, ml: -0.5 }}
                            slotProps={{
                              input: { 'aria-label': 'Select row' },
                            }}
                          />
                        ) : (
                          <span className='text-xs text-muted-foreground'>
                            {showIndex &&
                              `#${currentPage * rowsPerPage + index + 1}`}
                          </span>
                        )}
                        {actions && (
                          <div className='flex items-center'>
                            {actions(row)}
                          </div>
                        )}
                      </div>
                      <dl className='space-y-1'>
                        {columns.map((column) => {
                          const content = column.cell(row)
                          // Kolom penanda kosong (mis. D pada baris non-dynamic) tidak perlu baris label di mobile.
                          if (
                            (column.hideHeader || column.hideEmpty) &&
                            !content
                          )
                            return null
                          return (
                            <div
                              key={column.header}
                              className='flex items-start justify-between gap-3'
                            >
                              <dt className='shrink-0 text-muted-foreground'>
                                {column.header}
                              </dt>
                              <dd className='min-w-0 text-right break-words'>
                                {content}
                              </dd>
                            </div>
                          )
                        })}
                      </dl>
                    </li>
                  )
                })}
              </ul>
            </>
          )}
        </div>
      ) : (
        <TableContainer>
          <Table size='small'>
            <TableHead
              sx={{
                backgroundColor: 'action.hover',
                '& .MuiTableCell-root': {
                  backgroundColor: 'action.hover',
                  fontWeight: 600,
                },
                // The column row has no bottom border; the refresh-line row below it carries the divider.
                '& .MuiTableRow-root:first-of-type .MuiTableCell-root': {
                  borderBottom: 0,
                },
              }}
            >
              <TableRow>
                {selectable && (
                  <TableCell padding='checkbox' sx={{ width: 40 }}>
                    <Checkbox
                      size='small'
                      checked={allOnPageSelected}
                      indeterminate={selectedOnPage > 0 && !allOnPageSelected}
                      disabled={visible.length === 0}
                      onChange={togglePage}
                      slotProps={{
                        input: { 'aria-label': 'Select all rows on this page' },
                      }}
                    />
                  </TableCell>
                )}
                {!selectable && showIndex && (
                  <TableCell sx={{ fontWeight: 600, width: 48 }}>#</TableCell>
                )}
                {columns.map((column) => (
                  <TableCell
                    key={column.header}
                    align={column.align}
                    className={column.className}
                    sx={{
                      fontWeight: 600,
                      whiteSpace: 'nowrap',
                      ...(column.narrow && { width: 32, px: 0.5 }),
                    }}
                  >
                    {column.hideHeader ? null : column.getSortValue ? (
                      <TableSortLabel
                        active={sort?.column === column}
                        direction={
                          sort?.column === column ? sort.direction : 'asc'
                        }
                        onClick={() => {
                          setSort((current) => ({
                            column,
                            direction:
                              current?.column === column &&
                              current.direction === 'asc'
                                ? 'desc'
                                : 'asc',
                          }))
                          setPage(0)
                        }}
                      >
                        {column.header}
                      </TableSortLabel>
                    ) : (
                      column.header
                    )}
                  </TableCell>
                ))}
                {actions && (
                  <TableCell align='center' sx={{ fontWeight: 600, width: 80 }}>
                    Actions
                  </TableCell>
                )}
              </TableRow>
              <TableRow>
                <TableCell
                  colSpan={colCount}
                  sx={{ p: 0, height: 2, lineHeight: 0 }}
                >
                  <RefreshLine active={isRefreshing} className='rounded-none' />
                </TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {isLoading ? (
                <TableRow>
                  <TableCell
                    colSpan={colCount}
                    align='center'
                    sx={{
                      py: '2px',
                      lineHeight: 1.2,
                      fontSize: 13,
                      color: 'text.secondary',
                    }}
                  >
                    {loadingLabel}
                  </TableCell>
                </TableRow>
              ) : (
                <>
                  {visible.map((row, index) => (
                    <TableRow
                      key={getRowId(row)}
                      hover
                      selected={selectable && selectedIds.has(getRowId(row))}
                    >
                      {selectable && (
                        <TableCell padding='checkbox'>
                          <Checkbox
                            size='small'
                            checked={selectedIds.has(getRowId(row))}
                            onChange={() => toggleRow(row)}
                            slotProps={{
                              input: { 'aria-label': 'Select row' },
                            }}
                          />
                        </TableCell>
                      )}
                      {!selectable && showIndex && (
                        <TableCell>
                          {currentPage * rowsPerPage + index + 1}
                        </TableCell>
                      )}
                      {columns.map((column) => (
                        <TableCell
                          key={column.header}
                          align={column.align}
                          sx={{
                            whiteSpace: 'nowrap',
                            ...(column.narrow && { width: 32, px: 0.5 }),
                          }}
                        >
                          {column.cell(row)}
                        </TableCell>
                      ))}
                      {actions && (
                        <TableCell align='center'>
                          <div className='flex items-center justify-center gap-0'>
                            {actions(row)}
                          </div>
                        </TableCell>
                      )}
                    </TableRow>
                  ))}
                  {visible.length === 0 && (
                    <TableRow>
                      <TableCell
                        colSpan={colCount}
                        align='center'
                        sx={{ py: 1, color: 'text.secondary' }}
                      >
                        {search ? 'No results found.' : 'No data.'}
                      </TableCell>
                    </TableRow>
                  )}
                </>
              )}
            </TableBody>
          </Table>
        </TableContainer>
      )}
      <TablePagination
        component='div'
        count={filtered.length}
        page={currentPage}
        onPageChange={(_, newPage) => setPage(newPage)}
        rowsPerPage={rowsPerPage}
        rowsPerPageOptions={ROWS_PER_PAGE_OPTIONS}
        sx={{
          '& .MuiTablePagination-toolbar': {
            minHeight: 40,
            paddingInline: 1,
            flexWrap: 'wrap',
            justifyContent: { xs: 'center', sm: 'flex-end' },
          },
          '& .MuiTablePagination-spacer': {
            display: { xs: 'none', sm: 'block' },
          },
          '& .MuiTablePagination-selectLabel, & .MuiTablePagination-displayedRows':
            {
              margin: 0,
              fontSize: '0.8rem',
            },
          '& .MuiTablePagination-select': {
            paddingTop: 0.5,
            paddingBottom: 0.5,
          },
          '& .MuiTablePagination-actions button': {
            padding: 0.5,
          },
        }}
        onRowsPerPageChange={(event) => {
          setRowsPerPage(Number.parseInt(event.target.value, 10))
          setPage(0)
        }}
      />
    </Paper>
  )
}
