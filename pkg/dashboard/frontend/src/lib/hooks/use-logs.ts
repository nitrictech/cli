import { useCallback } from 'react'
import useSWR from 'swr'
import { fetcher } from './fetcher'
import type { LogEntry } from '@/types'
import { LOGS_API } from '../constants'

export const useLogs = (serviceName?: string) => {
  const { data, mutate } = useSWR<LogEntry[]>(
    `${LOGS_API}?service=${serviceName}`,
    fetcher(),
    {
      refreshInterval: 500,
    },
  )

  const purgeLogs = useCallback(async () => {
    await fetch(LOGS_API, {
      method: 'DELETE',
    })

    return mutate()
  }, [])

  return {
    data: data || [],
    mutate,
    purgeLogs,
    loading: !data,
  }
}
