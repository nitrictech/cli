import { type ComponentType } from 'react'

import type { Api } from '@/types'
import type { NodeProps } from 'reactflow'
import NodeBase, { type NodeBaseData } from './NodeBase'

export type ApiNodeData = NodeBaseData<Api>

export const APINode: ComponentType<NodeProps<ApiNodeData>> = (props) => {
  const { data } = props

  return (
    <NodeBase
      {...props}
      drawerOptions={{
        title: `Details - ${data.title}`,
        description: data.description,
        testHref: `/`, // TODO add url param to switch to resource
        children: (
          <div className="flex flex-col">
            <span className="font-bold">Requested by:</span>
            <span>{data.resource.requestingServices.join(', ')}</span>
          </div>
        ),
      }}
    />
  )
}
