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
    <div className="flex flex-col gap-y-2">
      {endpoints.map((endpoint) => (
        <div key={endpoint.id} className="grid w-full grid-cols-12 gap-4">
          <div className="col-span-2 flex">
            <APIMethodBadge method={endpoint.method} />
          </div>
          <div className="col-span-10 flex justify-start">
            <a
              target="_blank noreferrer noopener"
              className="truncate hover:underline"
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
