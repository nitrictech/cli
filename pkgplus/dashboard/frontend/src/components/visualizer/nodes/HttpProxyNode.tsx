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
        title: `HTTP Proxy ${data.title}`,
        description: data.description,
        // testHref: `/proxies`, // TODO add url param to switch to resource
        children: (
          <div className="flex flex-col">
            <span className="font-bold">Requested by:</span>
            <span>{data.resource.target}</span>
          </div>
        ),
      }}
    />
  )
}
