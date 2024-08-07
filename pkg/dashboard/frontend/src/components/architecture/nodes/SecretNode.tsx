import { type ComponentType } from 'react'

import type { Secret } from '@/types'
import type { NodeProps } from 'reactflow'
import NodeBase, { type NodeBaseData } from './NodeBase'

export type SecretNodeData = NodeBaseData<Secret>

export const SecretNode: ComponentType<NodeProps<SecretNodeData>> = (props) => {
  const { data } = props

  return (
    <NodeBase
      {...props}
      drawerOptions={{
        title: `Secret - ${data.title}`,
        description: data.description,
        icon: data.icon,
        nodeType: 'secret',
        testHref: `/secrets`, // TODO add url param to switch to resource
        services: data.resource.requestingServices,
      }}
    />
  )
}
