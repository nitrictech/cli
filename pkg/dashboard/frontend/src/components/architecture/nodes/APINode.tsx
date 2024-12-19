import { type ComponentType } from 'react'

import type { Api, Endpoint } from '@/types'
import type { NodeProps } from 'reactflow'
import NodeBase, { type NodeBaseData } from './NodeBase'
import APIRoutesList from '@/components/apis/APIRoutesList'

export interface ApiNodeData extends NodeBaseData<Api> {
  endpoints: Endpoint[]
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
        address: data.address,
        services: data.resource.requestingServices,
        trailingChildren: data.address ? (
          <div className="flex flex-col gap-y-1">
            <span className="font-bold">Routes:</span>
            <APIRoutesList
              apiAddress={data.address}
              endpoints={data.endpoints}
            />
          </div>
        ) : null,
      }}
    />
  )
}
