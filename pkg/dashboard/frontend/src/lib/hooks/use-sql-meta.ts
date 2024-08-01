import useSWR from 'swr'
import { fetcher } from './fetcher'
import { TABLE_QUERY } from '../constants'
import { getHost } from '../utils'

export interface SqlMetaResult {
  columns: {
    column_name: string
    data_type: string
    column_order: number
  }[]
  is_table: boolean
  qualified_name: string
  schema_name: string
  table_name: string
}

export const useSqlMeta = (connectionString?: string) => {
  const { data, mutate } = useSWR<SqlMetaResult[]>(
    connectionString ? `http://${getHost()}/api/sql` : null,
    fetcher({
      method: 'POST',
      body: JSON.stringify({ query: TABLE_QUERY, connectionString }),
    }),
  )

  return {
    data,
    mutate,
    loading: !data,
  }
}
