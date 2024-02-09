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
        icon: data.icon,
        nodeType: 'keyvaluestore',
        services: data.resource.requestingServices,
        // testHref: `/stores`, // TODO add url param to switch to resource
      }}
    />
  )
}
