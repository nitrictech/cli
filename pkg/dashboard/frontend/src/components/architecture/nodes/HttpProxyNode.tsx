import { type ComponentType } from 'react'

import type { HttpProxy } from '@/types'
import type { NodeProps } from 'reactflow'
import NodeBase, { type NodeBaseData } from './NodeBase'

export type HttpProxyNodeData = NodeBaseData<HttpProxy>

export const HttpProxyNode: ComponentType<NodeProps<HttpProxyNodeData>> = (
  props,
) => {
  const { data } = props

  return (
    <NodeBase
      {...props}
      drawerOptions={{
        title: `HTTP Proxy - ${data.title}`,
        description: data.description,
        icon: data.icon,
        nodeType: 'httpproxy',
        testHref: data.address,
        address: data.address,
        services: [data.resource.target],
      }}
    />
  )
}
