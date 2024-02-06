import { type ComponentType } from 'react'

import type { Api } from '@/types'
import type { NodeProps } from 'reactflow'
import NodeBase, { type NodeBaseData } from './NodeBase'

export interface ApiNodeData extends NodeBaseData<Api> {
  address: string
}

export const APINode: ComponentType<NodeProps<ApiNodeData>> = (props) => {
  const { data } = props

  return (
    <NodeBase
      {...props}
      drawerOptions={{
        title: `API - ${data.title}`,
        description: data.description,
        icon: data.icon,
        nodeType: 'api',
        testHref: `/`, // TODO add url param to switch to resource
        children: (
          <>
            <div className="flex flex-col">
              <span className="font-bold">Address:</span>
              <a
                target="_blank"
                className="hover:underline"
                href={`http://${data.address}`} rel="noreferrer"
              >
                {data.address}
              </a>
            </div>
            <div className="flex flex-col">
              <span className="font-bold">Requested by:</span>
              <span>{data.resource.requestingServices.join(', ')}</span>
            </div>
          </>
        ),
      }}
    />
  )
}
