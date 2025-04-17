import React from 'react'
import { APIMethodBadge } from './APIMethodBadge'
import type { Endpoint } from '@/types'

interface APIRoutesListProps {
  endpoints: Endpoint[]
  apiAddress: string
}

const APIRoutesList: React.FC<APIRoutesListProps> = ({
  endpoints,
  apiAddress,
}) => {
  return (
    <div className="flex flex-col gap-y-2" data-testid="api-routes-list">
      {endpoints.map((endpoint) => (
        <div key={endpoint.id} className="grid w-full grid-cols-12 gap-4">
          <div className="col-span-3 flex">
            <APIMethodBadge method={endpoint.method} />
          </div>
          <div className="col-span-9 flex justify-start">
            <a
              target="_blank noreferrer noopener"
              className="truncate text-foreground hover:text-accent-foreground hover:underline"
              href={`${apiAddress}${endpoint.path}`}
              rel="noreferrer"
            >
              {endpoint.path}
            </a>
          </div>
        </div>
      ))}
    </div>
  )
}

export default APIRoutesList
