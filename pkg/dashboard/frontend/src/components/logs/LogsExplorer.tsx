import React, { useEffect, useMemo, useRef } from 'react'
import AppLayout from '../layout/AppLayout'
import { cn } from '@/lib/utils'
import { Button } from '../ui/button'
import { useLogs } from '@/lib/hooks/use-logs'
import { Tooltip, TooltipContent, TooltipTrigger } from '../ui/tooltip'
import { format } from 'date-fns/format'
import { formatDistanceToNow } from 'date-fns/formatDistanceToNow'
import {
  ArrowDownOnSquareIcon,
  EllipsisVerticalIcon,
  MagnifyingGlassIcon,
  TrashIcon,
} from '@heroicons/react/24/outline'

import TextField from '../shared/TextField'
import { debounce } from 'radash'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '../ui/dropdown-menu'
import type { LogEntry } from '@/types'
import { SidebarInset, SidebarProvider } from '../ui/sidebar'
import { FilterSidebar } from './FilterSidebar'
import FilterTrigger from './FilterTrigger'
import { ParamsProvider, useParams } from '@/hooks/use-params'
import { AnsiHtml } from 'fancy-ansi/react'
import { useVirtualizer } from '@tanstack/react-virtual'
import { ScrollArea } from '../ui/scroll-area'
import { Portal as TooltipPortal } from '@radix-ui/react-tooltip'

const exportJSON = async (logs: LogEntry[]) => {
  const json = JSON.stringify(logs, null, 2)
  const blob = new Blob([json], { type: 'application/json' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = `logs-${new Date().toISOString()}.json`
  document.body.appendChild(a)
  a.click()
  document.body.removeChild(a)
  URL.revokeObjectURL(url)
}

const Logs: React.FC = () => {
  const parentRef = useRef<HTMLDivElement>(null)

  const { searchParams, setParams } = useParams()

  const {
    data: logs,
    purgeLogs,
    mutate,
  } = useLogs({
    search: searchParams.get('search') ?? undefined,
    origin: searchParams.get('origin') ?? undefined,
    level: (searchParams.get('level') as LogEntry['level']) ?? undefined,
    timeline: searchParams.get('timeline') ?? undefined,
  })

  const debouncedSearch = debounce({ delay: 500 }, (search: string) => {
    setParams('search', search)
  })

  useEffect(() => {
    mutate()
  }, [searchParams])

  const formattedLogs = useMemo(
    () =>
      logs.map((log) => ({
        ...log,
        formattedTime: format(new Date(log.time), 'MMM dd, HH:mm:ss.SS'),
        relativeTime: formatDistanceToNow(new Date(log.time), {
          addSuffix: true,
        }),
        timestamp: new Date(log.time).getTime(),
      })),
    [logs.length],
  )

  const virtualizer = useVirtualizer({
    count: formattedLogs.length,
    getScrollElement: () => parentRef.current,
    estimateSize: () => 21.5, // estimated height of each row
    overscan: 50, // number of rows to render outside of the viewport
  })

  const items = virtualizer.getVirtualItems()

  return (
    <AppLayout title="Logs" routePath="/logs" hideTitle>
      <SidebarProvider defaultOpen={false}>
        <FilterSidebar />
        <SidebarInset>
          <div className="relative flex flex-col gap-4">
            <div className="flex items-start gap-2">
              <FilterTrigger />
              <TextField
                id="log-search"
                label="Search"
                className="w-full text-lg"
                data-testid="log-search"
                hideLabel
                defaultValue={searchParams.get('search') ?? ''}
                icon={MagnifyingGlassIcon}
                placeholder="Search"
                onChange={(event) => debouncedSearch(event.target.value)}
              />
              <DropdownMenu>
                <DropdownMenuTrigger asChild>
                  <Button
                    size="icon"
                    variant="outline"
                    className="ml-auto"
                    data-testid="log-options-btn"
                  >
                    <span className="sr-only">Open log options</span>
                    <EllipsisVerticalIcon
                      className="size-6 text-foreground"
                      aria-hidden="true"
                    />
                  </Button>
                </DropdownMenuTrigger>
                <DropdownMenuContent className="w-56">
                  <DropdownMenuGroup>
                    <DropdownMenuItem onClick={() => exportJSON(logs)}>
                      <ArrowDownOnSquareIcon className="mr-2 h-4 w-4" />
                      <span>Export as JSON</span>
                    </DropdownMenuItem>
                    <DropdownMenuItem
                      onClick={purgeLogs}
                      data-testid="purge-logs-btn"
                    >
                      <TrashIcon className="mr-2 h-4 w-4" />
                      <span>Clear Logs</span>
                    </DropdownMenuItem>
                  </DropdownMenuGroup>
                </DropdownMenuContent>
              </DropdownMenu>
            </div>
            <div className="mx-1 grid grid-cols-[200px_150px_1fr] gap-x-2 border-b pb-2 text-lg font-semibold">
              <span>Time</span>
              <span>Origin</span>
              <span>Message</span>
            </div>
            <div className="mt-1 flex-col font-mono text-sm">
              <ScrollArea type="auto" ref={parentRef} className="h-screen">
                <div
                  className="relative w-full"
                  style={{
                    height: virtualizer.getTotalSize(),
                  }}
                >
                  <div
                    className="absolute left-0 top-0 w-full"
                    data-testid="logs"
                    style={{
                      transform: `translateY(${items[0]?.start ?? 0}px)`,
                    }}
                  >
                    {formattedLogs.length > 0 ? (
                      items.map(({ index: i }) => {
                        const {
                          msg,
                          level,
                          formattedTime,
                          relativeTime,
                          origin,
                          timestamp,
                        } = formattedLogs[i]
                        const formattedLine = msg.trim()
                        return (
                          <div
                            key={i}
                            className={cn(
                              'mt-0.5 grid cursor-pointer grid-cols-[200px_150px_1fr] items-start gap-x-2 whitespace-pre-wrap rounded-sm px-1 py-[2px] text-foreground hover:bg-accent/50',
                              {
                                'bg-red-500/20 hover:bg-red-500/30':
                                  level === 'error',
                                'bg-orange-100 hover:bg-orange-200 dark:bg-orange-500/60 dark:hover:bg-orange-500/70':
                                  level === 'warning',
                              },
                            )}
                          >
                            <Tooltip>
                              <TooltipTrigger className="w-[150px] truncate text-left">
                                <span className="w-[200px] truncate text-left">
                                  {formattedTime}
                                </span>
                              </TooltipTrigger>
                              <TooltipPortal>
                                <TooltipContent
                                  className="grid grid-cols-2 gap-2 font-mono text-xs"
                                  side="right"
                                >
                                  <span>
                                    {
                                      Intl.DateTimeFormat().resolvedOptions()
                                        .timeZone
                                    }
                                    :{' '}
                                  </span>
                                  <span>{formattedTime}</span>
                                  <span>Relative:</span>
                                  <span>{relativeTime}</span>
                                  <span>Timestamp:</span>
                                  <span>{timestamp}</span>
                                </TooltipContent>
                              </TooltipPortal>
                            </Tooltip>

                            <Tooltip>
                              <TooltipTrigger className="relative w-[150px] truncate text-left">
                                <span data-testid={`test-row${i}-origin`}>
                                  {origin}
                                </span>
                              </TooltipTrigger>
                              <TooltipPortal>
                                <TooltipContent className="font-mono text-xs">
                                  <p>{origin}</p>
                                </TooltipContent>
                              </TooltipPortal>
                            </Tooltip>
                            <AnsiHtml
                              className="border-l pl-2 text-left"
                              data-testid={`test-row${i}-msg`}
                              text={formattedLine}
                            />
                          </div>
                        )
                      })
                    ) : (
                      <div className="mt-4 p-1 text-center font-body text-lg font-semibold tracking-wide text-muted-foreground">
                        No logs available
                      </div>
                    )}
                  </div>
                </div>
              </ScrollArea>
            </div>
          </div>
        </SidebarInset>
      </SidebarProvider>
    </AppLayout>
  )
}

export default function LogsExplorer() {
  return (
    <ParamsProvider>
      <Logs />
    </ParamsProvider>
  )
}
