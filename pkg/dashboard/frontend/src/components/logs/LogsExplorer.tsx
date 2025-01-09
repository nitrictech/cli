import React from 'react'
import AppLayout from '../layout/AppLayout'
import { cn } from '@/lib/utils'
import { Button } from '../ui/button'
import { useLogs } from '@/lib/hooks/use-logs'
import { Tooltip, TooltipContent, TooltipTrigger } from '../ui/tooltip'

const Logs: React.FC = () => {
  const { data: logs, purgeLogs } = useLogs('/api/logs')

  return (
    <AppLayout title="Logs" routePath="/logs">
      <div className="relative flex flex-col">
        <div className="flex">
          <Button
            onClick={purgeLogs}
            variant="destructive"
            className="ml-auto"
            title="Copy logs to clipboard"
            size="sm"
          >
            Purge Logs
          </Button>
        </div>
        <div className="flex gap-3 border-b pb-2 text-lg font-semibold">
          <span className="w-[200px]">Time</span>
          <span className="w-[150px]">Service</span>
          <span>Message</span>
        </div>
        <div className="mt-1 flex flex-col font-mono text-sm">
          {logs.map(({ msg, time, serviceName }, i) => {
            const formattedLine = msg.trim()
            return (
              <div
                key={i}
                className={cn(
                  'mt-0.5 flex cursor-pointer gap-x-2 whitespace-pre-wrap rounded-sm px-1 py-[2px] hover:bg-gray-100 dark:hover:bg-gray-700',
                  {
                    'bg-red-100 hover:bg-red-200 dark:bg-red-800/70 dark:hover:bg-red-800/90':
                      msg.toLowerCase().includes('error'),
                    'bg-orange-100 hover:bg-orange-200 dark:bg-orange-500/60 dark:hover:bg-orange-500/70':
                      msg.toLowerCase().includes('warning'),
                  },
                )}
              >
                <span className="w-[200px] truncate">
                  {new Date(time).toLocaleString('en-US', {
                    month: 'short',
                    day: '2-digit',
                    hour: '2-digit',
                    minute: '2-digit',
                    second: '2-digit',
                    fractionalSecondDigits: 2,
                  })}
                </span>
                <Tooltip>
                  <TooltipTrigger className="w-[150px] truncate">
                    <span>{serviceName}</span>
                  </TooltipTrigger>
                  <TooltipContent>
                    <p>{serviceName}</p>
                  </TooltipContent>
                </Tooltip>
                <span className="truncate border-l pl-2">{formattedLine}</span>
              </div>
            )
          })}
        </div>
      </div>
    </AppLayout>
  )
}

export default Logs
