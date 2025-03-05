import React from 'react'
import DataGrid, { type CalculatedColumn } from 'react-data-grid'
import Spinner from '../shared/Spinner'
import {
  ContextMenu,
  ContextMenuContent,
  ContextMenuItem,
  ContextMenuTrigger,
} from '../ui/context-menu'
import { copyToClipboard } from '@/lib/utils/copy-to-clipboard'
import { cn } from '@/lib/utils'

interface QueryResultsProps {
  response?: string
  loading?: boolean
}

interface Result {
  table_name: string
  [key: string]: any
}

const parse = (value: string): Result[] | string => {
  try {
    return JSON.parse(value)
  } catch (e) {
    return value
  }
}

const Container: React.FC<React.PropsWithChildren> = ({ children }) => {
  return (
    <div className="rounded-lg bg-background font-mono text-foreground border border-border shadow-md">
      {children}
    </div>
  )
}

const EST_CHAR_WIDTH = 8
const MIN_COLUMN_WIDTH = 100
const MAX_COLUMN_WIDTH = 500

function isBinaryString(str: string) {
  // Regular expression to match a string with \x followed by two hexadecimal digits
  const binaryPattern = /^\\x([0-9A-Fa-f]{2})+$/
  return binaryPattern.test(str)
}

const QueryResults: React.FC<QueryResultsProps> = ({ response, loading }) => {
  if (loading) {
    return (
      <Container>
        <p className="m-0 flex items-center border-0 px-6 py-4 text-sm">
          <Spinner color="info" size={'sm'} className="mb-0.5 mr-2" />
          <span className="text-foreground">Running...</span>
        </p>
      </Container>
    )
  }

  if (!response) {
    return (
      <Container>
        <p className="m-0 border-0 px-6 py-4 text-sm">
          Click <span className="font-bold text-foreground">Run</span> to execute your query.
        </p>
      </Container>
    )
  }

  const rows = parse(response)

  // if the data is a string after parse, we can assume it's a error response
  if (typeof rows === 'string') {
    return (
      <Container>
        <p className="m-0 border-0 px-6 py-4 text-sm">{rows}</p>
      </Container>
    )
  }

  if (rows.length <= 0) {
    return (
      <Container>
        <p className="m-0 border-0 px-6 py-4 text-sm">
          Success. No rows returned
        </p>
      </Container>
    )
  }

  const formatter = (column: any, row: any) => {
    const dataType = typeof row[column]

    const isBinary = isBinaryString(row[column])

    const isEmpty = row[column] === ''

    const displayValue = isEmpty
      ? 'empty'
      : isBinary
        ? 'binary'
        : JSON.stringify(row[column]).replace(/^"|"$/g, '')

    return (
      <ContextMenu>
        <ContextMenuTrigger
          data-db-type={typeof row[column]}
          className={cn('w-full whitespace-pre font-mono', {
            uppercase:
              row[column] === null ||
              dataType === 'boolean' ||
              isBinary ||
              isEmpty,
            'text-muted-foreground': row[column] === null || isEmpty,
          })}
        >
          {displayValue}
        </ContextMenuTrigger>
        <ContextMenuContent>
          <ContextMenuItem
            onClick={() => {
              copyToClipboard(row[column])
            }}
          >
            Copy cell content
          </ContextMenuItem>
        </ContextMenuContent>
      </ContextMenu>
    )
  }

  const renderColumn = (name: string) => {
    return (
      <div className="flex h-full items-center justify-center font-mono">
        {name}
      </div>
    )
  }

  const columns: CalculatedColumn<any>[] = Object.keys(rows?.[0] ?? []).map(
    (key, idx) => {
      const columnWidth = Math.max(
        Math.min(
          rows.reduce(
            (maxLen, row) => Math.max(maxLen, String(row[key]).length),
            0,
          ) * EST_CHAR_WIDTH,
          MAX_COLUMN_WIDTH,
        ),
        MIN_COLUMN_WIDTH,
      )

      return {
        idx,
        key,
        name: key,
        resizable: true,
        parent: undefined,
        level: 0,
        width: columnWidth,
        minWidth: MIN_COLUMN_WIDTH,
        maxWidth: undefined,
        draggable: false,
        frozen: false,
        sortable: false,
        isLastFrozenColumn: false,
        renderCell: ({ row }: any) => formatter(key, row),
        renderHeaderCell: () => renderColumn(key),
      }
    },
  )

  if (import.meta.env.DEV) {
    console.log('result-rows: ', rows)
  }

  return (
    <Container>
      <DataGrid
        data-testid={'query-results'}
        columns={columns}
        rows={rows}
        className="flex-grow border-t-0"
        rowClass={() => '[&>.rdg-cell]:items-center'}
      />
    </Container>
  )
}

export default QueryResults
