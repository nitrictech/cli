import { useEffect, useMemo, useState } from 'react'
import { useWebSocket } from '../../lib/hooks/use-web-socket'
import type { SchemaObj, SQLDatabase } from '@/types'
import { Loading } from '../shared'
import { fieldRowArrToHeaders, getHost, generateResponse } from '@/lib/utils'

import AppLayout from '../layout/AppLayout'
import { copyToClipboard } from '../../lib/utils/copy-to-clipboard'
import ClipboardIcon from '@heroicons/react/24/outline/ClipboardIcon'
import toast from 'react-hot-toast'
import { Tooltip, TooltipContent, TooltipTrigger } from '../ui/tooltip'
import BreadCrumbs from '../layout/BreadCrumbs'
import DatabasesTreeView from './DatabasesTreeView'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '../ui/select'
import { Button } from '../ui/button'
import CodeEditor from '../apis/CodeEditor'
import QueryResults from './QueryResults'
import { useSqlMeta } from '@/lib/hooks/use-sql-meta'
import SectionCard from '../shared/SectionCard'
import NotFoundAlert from '../shared/NotFoundAlert'

interface QueryHistoryItem {
  query: string
  timestamp: number
  dbName: string
}

interface QueryHistory {
  [db: string]: QueryHistoryItem[]
}

const DATABASES_STORAGE_KEY = 'nitric-local-dash-database'

const getStorageHistory = (): QueryHistory | null => {
  try {
    const storage = localStorage.getItem(DATABASES_STORAGE_KEY)
    if (storage) {
      return JSON.parse(storage)
    }
  } catch (error) {
    console.error('Error parsing JSON from storage:', error)
  }
  return null
}

const setStorageHistory = (value: QueryHistory) => {
  localStorage.setItem(DATABASES_STORAGE_KEY, JSON.stringify(value))
}

const DatabasesExplorer: React.FC = () => {
  const { data, loading } = useWebSocket()
  const [callLoading, setCallLoading] = useState(false)
  const [migrationLoading, setMigrationLoading] = useState(false)

  const [response, setResponse] = useState<string>()

  const [selectedDb, setSelectedDb] = useState<SQLDatabase>()

  const { data: tables, mutate: refreshTables } = useSqlMeta(
    selectedDb?.connectionString,
  )

  // takes tables and converts it into an object of schema keys with an array of table names
  const schemaObj: SchemaObj | undefined = useMemo(() => {
    return tables?.reduce((acc, table) => {
      if (!acc) return {}

      const key = `${table.schema_name}.${table.table_name}`

      if (!acc[key]) {
        acc[key] = table.columns
          .sort((a, b) => a.column_order - b.column_order)
          .map((column) => ({
            label: column.column_name,
            type: 'property',
          }))
      }

      return acc
    }, {} as SchemaObj) // Add index signature to allow indexing with a string
  }, [tables])

  if (import.meta.env.DEV) {
    console.log('tables', tables)
    console.log('schemaObj', schemaObj)
  }

  const [sql, setSql] = useState('')

  // set selectedDb based on data.sqlDatabases
  useEffect(() => {
    if (data && data.sqlDatabases.length && !selectedDb) {
      setSelectedDb(data.sqlDatabases[0])
    } else if (selectedDb?.status === 'active') {
      // refresh tables when selectedDb is active, after migrations
      refreshTables()
    }
  }, [data])

  // clean up state when selectedDb changes
  useEffect(() => {
    setResponse(undefined)
    refreshTables()

    setSql('')
  }, [selectedDb])

  const handleRun = async (
    e: React.MouseEvent<HTMLButtonElement, MouseEvent>,
  ) => {
    if (!selectedDb) return
    setCallLoading(true)
    e.preventDefault()

    if (!sql) {
      setResponse('Error: Query should not be empty')
      setCallLoading(false)
      return
    }

    const url = `http://${getHost()}/api/sql`
    const requestOptions: RequestInit = {
      method: 'POST',
      body: JSON.stringify({
        query: sql,
        connectionString: selectedDb.connectionString,
      }),
      headers: fieldRowArrToHeaders([
        {
          key: 'Accept',
          value: '*/*',
        },
        {
          key: 'User-Agent',
          value: 'Nitric Client (https://www.nitric.io)',
        },
      ]),
    }

    const startTime = window.performance.now()
    const res = await fetch(url, requestOptions)

    const callResponse = await generateResponse(res, startTime)
    setResponse(callResponse.data)

    // refresh tables in case of DDL changes
    refreshTables()

    setTimeout(() => setCallLoading(false), 300)
  }

  const handleMigrate = async () => {
    if (!selectedDb) return

    setMigrationLoading(true)

    const loadingId = toast.loading('Migrating database')

    const url = `http://${getHost()}/api/sql/migrate`

    const requestOptions: RequestInit = {
      method: 'POST',
      body: JSON.stringify({
        databaseName: selectedDb.name,
      }),
      headers: fieldRowArrToHeaders([
        {
          key: 'Accept',
          value: '*/*',
        },
        {
          key: 'User-Agent',
          value: 'Nitric Client (https://www.nitric.io)',
        },
      ]),
    }

    const res = await fetch(url, requestOptions)

    if (res.ok) {
      toast.success('Migration successful', { id: loadingId })
    } else {
      const text = await res.text()
      toast.error('Migration failed: ' + text, { id: loadingId })
    }

    setMigrationLoading(false)
  }

  const hasData = Boolean(data && data.sqlDatabases.length)

  // Save and retrieve SQL from localStorage
  useEffect(() => {
    const queryHistory = getStorageHistory() || {}
    const queries = queryHistory[selectedDb?.name || '']
    if (queries) {
      const latestQuery = queries[queries.length - 1]
      setSql(latestQuery.query)
    } else {
      setSql('')
    }
  }, [selectedDb])

  useEffect(() => {
    if (selectedDb) {
      const queryHistory = getStorageHistory() || {}
      // TODO allow more than one saved query. const queries = queryHistory[selectedDb.name] || []
      const queries = []
      queries.push({
        query: sql,
        timestamp: Date.now(),
        dbName: selectedDb.name,
      })
      queryHistory[selectedDb.name] = queries
      setStorageHistory(queryHistory)
    }
  }, [selectedDb, sql])

  return (
    <AppLayout
      title={'Databases'}
      hideTitle
      routePath={`/databases`}
      secondLevelNav={
        data &&
        selectedDb && (
          <>
            <div className="flex min-h-12 items-center justify-between px-2 py-1">
              <span className="text-lg">Databases</span>
            </div>
            <DatabasesTreeView
              initialItem={selectedDb}
              onSelect={setSelectedDb}
              resources={data.sqlDatabases ?? []}
            />
          </>
        )
      }
    >
      <Loading delay={400} conditionToShow={!loading}>
        {selectedDb && hasData ? (
          <div className="flex max-w-[2000px] flex-col gap-8 md:pr-8">
            <div className="flex w-full flex-col gap-8">
              <div>
                <div className="lg:hidden">
                  {hasData && (
                    <Select
                      value={selectedDb.name}
                      onValueChange={(name) => {
                        setSelectedDb(
                          data?.sqlDatabases.find((b) => b.name === name),
                        )
                      }}
                    >
                      <SelectTrigger className="w-full">
                        <SelectValue placeholder={`Select Database`} />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectGroup>
                          {data?.sqlDatabases.map((db) => (
                            <SelectItem key={db.name} value={db.name}>
                              {db.name}
                            </SelectItem>
                          ))}
                        </SelectGroup>
                      </SelectContent>
                    </Select>
                  )}
                  {selectedDb.migrationsPath && (
                    <Tooltip>
                      <TooltipTrigger asChild>
                        <Button
                          disabled={migrationLoading}
                          onClick={handleMigrate}
                          className="ml-auto mt-2 flex"
                        >
                          Run Migrations
                        </Button>
                      </TooltipTrigger>
                      <TooltipContent>
                        <p>
                          Run migrations from{' '}
                          <strong>{selectedDb.migrationsPath}</strong>
                        </p>
                      </TooltipContent>
                    </Tooltip>
                  )}
                </div>
                <div className="hidden items-center gap-4 lg:flex">
                  <BreadCrumbs className="text-lg">
                    <span>Databases</span>
                    <h2 className="font-body text-lg font-semibold">
                      {selectedDb.name}
                    </h2>
                  </BreadCrumbs>
                  {selectedDb.migrationsPath && (
                    <Tooltip>
                      <TooltipTrigger asChild>
                        <Button
                          data-testid="migrate-btn"
                          disabled={migrationLoading}
                          onClick={handleMigrate}
                          className="ml-auto"
                        >
                          Run Migrations
                        </Button>
                      </TooltipTrigger>
                      <TooltipContent>
                        <p>
                          Run migrations from{' '}
                          <strong>{selectedDb.migrationsPath}</strong>
                        </p>
                      </TooltipContent>
                    </Tooltip>
                  )}
                </div>
                {!data?.sqlDatabases.some(
                  (s) => s.name === selectedDb.name,
                ) && (
                  <NotFoundAlert className="mt-4">
                    Database not found. It might have been updated or removed.
                    Select another database.
                  </NotFoundAlert>
                )}
              </div>

              <SectionCard title="Connect">
                <div className="mb-4 flex max-w-full gap-x-2 text-sm">
                  <span
                    data-testid="connection-string"
                    className="truncate font-mono text-foreground"
                  >
                    {selectedDb.connectionString}
                  </span>
                  <Tooltip>
                    <TooltipTrigger asChild>
                      <button
                        type="button"
                        onClick={() => {
                          copyToClipboard(selectedDb.connectionString)
                          toast.success(`Copied Connection String`)
                        }}
                      >
                        <span className="sr-only">Copy connection string</span>
                        <ClipboardIcon className="h-5 w-5 text-muted-foreground hover:text-foreground" />
                      </button>
                    </TooltipTrigger>
                    <TooltipContent>
                      <p>Copy Connection String</p>
                    </TooltipContent>
                  </Tooltip>
                </div>
              </SectionCard>
              <SectionCard title="SQL Editor">
                <div>
                  <CodeEditor
                    id="sql-editor"
                    value={sql}
                    enableCopy
                    sqlSchema={schemaObj}
                    contentType="text/sql"
                    onChange={(payload: string) => {
                      try {
                        setSql(payload)
                      } catch {
                        return
                      }
                    }}
                  />

                  <div className="mt-4 flex w-full items-center justify-between">
                    <div className="flex items-center gap-x-2">
                      <h3 className="text-xl font-semibold leading-6 text-foreground">
                        Results
                      </h3>
                    </div>
                    <Button
                      size="lg"
                      data-testid={`run-btn`}
                      onClick={handleRun}
                    >
                      Run
                    </Button>
                  </div>
                  <div className="mt-4">
                    <QueryResults response={response} loading={callLoading} />
                  </div>
                </div>
              </SectionCard>
            </div>
          </div>
        ) : !hasData ? (
          <div>
            Please refer to our documentation on{' '}
            <a
              className="underline text-foreground hover:text-accent-foreground"
              target="_blank"
              href="https://nitric.io/docs/sql"
              rel="noreferrer"
            >
              creating sql databases
            </a>{' '}
            as we are unable to find any existing database.
          </div>
        ) : null}
      </Loading>
    </AppLayout>
  )
}

export default DatabasesExplorer
