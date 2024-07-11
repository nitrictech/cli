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
        title: `API - ${data.title}`,
        description: data.description,
        icon: data.icon,
        nodeType: 'api',
        testHref: `/`, // TODO add url param to switch to resource
        address: data.address,
        services: data.resource.requestingServices,
      }}
    />
  )
}
