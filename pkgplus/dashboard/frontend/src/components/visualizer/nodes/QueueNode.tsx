import { type ComponentType } from 'react'

import type { Queue } from '@/types'
import type { NodeProps } from 'reactflow'
import NodeBase, { type NodeBaseData } from './NodeBase'

export type QueueNodeData = NodeBaseData<Queue>

export const QueueNode: ComponentType<NodeProps<QueueNodeData>> = (
  props,
) => {
  const { data } = props

  return (
    <NodeBase
      {...props}
      drawerOptions={{
        title: `Details - ${data.title}`,
        description: data.description,
        icon: data.icon,
        nodeType: 'queue',
        services: data.resource.requestingServices,
        // testHref: `/stores`, // TODO add url param to switch to resource
      }}
    />
  )
}
