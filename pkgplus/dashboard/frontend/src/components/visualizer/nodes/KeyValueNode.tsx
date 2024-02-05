import { type ComponentType } from 'react'

import type { KeyValue } from '@/types'
import type { NodeProps } from 'reactflow'
import NodeBase, { type NodeBaseData } from './NodeBase'

export type KeyValueNodeData = NodeBaseData<KeyValue>

export const KeyValueNode: ComponentType<NodeProps<KeyValueNodeData>> = (
  props,
) => {
  const { data } = props

  return (
    <NodeBase
      {...props}
      drawerOptions={{
        title: `Details - ${data.title}`,
        description: data.description,
        // testHref: `/stores`, // TODO add url param to switch to resource
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
