import { type ComponentType } from 'react'

import type { SQLDatabase } from '@/types'
import type { NodeProps } from 'reactflow'
import NodeBase, { type NodeBaseData } from './NodeBase'
import React from 'react'
import { copyToClipboard } from '@/lib/utils/copy-to-clipboard'
import toast from 'react-hot-toast'
import { ClipboardIcon } from '@heroicons/react/24/outline'

export type SQLNodeData = NodeBaseData<SQLDatabase>

export const SQLNode: ComponentType<NodeProps<SQLNodeData>> = (props) => {
  const { data } = props

  return (
    <NodeBase
      {...props}
      drawerOptions={{
        title: `SQL Database - ${data.title}`,
        description: data.description,
        icon: data.icon,
        nodeType: 'sql',
        testHref: `/databases`, // TODO add url param to switch to resource
        services: data.resource.requestingServices,
        children: (
          <>
            <div className="flex flex-col">
              <span className="font-bold">Status:</span>
              <span>{data.resource.status}</span>
            </div>
            <div className="flex flex-col">
              <span className="font-bold">Connection String:</span>
              <span className="flex gap-x-0.5">
                <span className="truncate">
                  {data.resource.connectionString}
                </span>
                <button
                  type="button"
                  onClick={() => {
                    copyToClipboard(data.resource.connectionString)
                    toast.success(`Copied Connection String`)
                  }}
                >
                  <span className="sr-only">Copy connection string</span>
                  <ClipboardIcon className="h-5 w-5 text-muted-foreground hover:text-foreground" />
                </button>
              </span>
            </div>
            <div className="flex flex-col">
              <span className="font-bold">Migrations:</span>
              <span>{data.resource.migrationsPath || 'No migrations'}</span>
            </div>
          </>
        ),
      }}
    />
  )
}
