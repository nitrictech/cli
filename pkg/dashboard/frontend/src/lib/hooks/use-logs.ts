import { useCallback } from 'react'
import useSWR from 'swr'
import { fetcher } from './fetcher'
import type { LogEntry } from '@/types'
import { LOGS_API } from '../constants'

interface LogQueryParams {
  origin?: string
  timeline?: string
  level?: LogEntry['level']
  search?: string
}

const buildQueryString = (params: LogQueryParams) => {
  const searchParams = new URLSearchParams()
  Object.entries(params).forEach(([key, value]) => {
    if (value !== undefined && value !== null) {
      searchParams.append(key, String(value))
    }
  })
  return searchParams.toString()
}

export const useLogs = (query: LogQueryParams) => {
  // Build query string dynamically
  const queryString = buildQueryString(query)

  // build query string
  const { data, mutate } = useSWR<LogEntry[]>(
    `${LOGS_API}?${queryString}`,
    fetcher(),
    {
      refreshInterval: 250,
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
