import type { FC } from 'react'
import { Badge } from '../shared'
import type { Method } from '../../types'

interface Props {
  method: Method
  className?: string
}

const methodColors = {
  DELETE: {
    bg: 'bg-red-100 dark:bg-red-900/30',
    text: 'text-red-800 dark:text-red-300'
  },
  POST: {
    bg: 'bg-green-100 dark:bg-green-900/30',
    text: 'text-green-800 dark:text-green-300'
  },
  PUT: {
    bg: 'bg-yellow-100 dark:bg-yellow-900/30',
    text: 'text-yellow-800 dark:text-yellow-300'
  },
  PATCH: {
    bg: 'bg-orange-100 dark:bg-orange-900/30',
    text: 'text-orange-800 dark:text-orange-300'
  },
  GET: {
    bg: 'bg-blue-100 dark:bg-blue-900/30',
    text: 'text-blue-800 dark:text-blue-300'
  }
}

export const APIMethodBadge: FC<Props> = ({ method, className }) => {
  return (
    <Badge
      className={className}
      status={
        (
          {
            DELETE: 'red',
            POST: 'green',
            PUT: 'yellow',
            PATCH: 'orange',
            GET: 'blue',
          } as any
        )[method]
      }
    >
      {method}
    </Badge>
  )
}
