import { type ComponentType } from 'react'

import type { HttpProxy } from '@/types'
import type { NodeProps } from 'reactflow'
import NodeBase, { type NodeBaseData } from './NodeBase'

export interface HttpProxyNodeData extends NodeBaseData<HttpProxy> {
  proxy: string
}

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
        icon: data.icon,
        nodeType: 'httpproxy',
        testHref: `http://${data.proxy}`,
        children: (
          <div className="flex flex-col">
            <span className="font-bold">Requested by:</span>
            <span>{data.resource.target.replace(/\\/g, '/')}</span>
          </div>
        ),
      }}
    />
  )
}
